package atmosphere

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// Composition describes the dry-air mixture used for Rayleigh scattering,
// parameterized by its carbon-dioxide fraction.
type Composition struct{ carbonDioxide float64 }

// The numbers the air's composition is counted in, each defined.
const (
	none            = 0.0   // defined by the scale: no co2 at all
	partsInAMillion = 1e6   // defined by the unit: parts a million
	percent         = 100.0 // defined by the unit: parts a hundred
	whole           = 1.0   // defined: the index of no refraction
)

// DryAir constructs dry-air composition from CO2 in parts per million. Values
// outside the fraction from zero to one million ppm return an error.
func DryAir(partsAMillion float64) (*Composition, error) {
	if !(partsAMillion >= none && partsAMillion <= partsInAMillion) {
		return nil, fmt.Errorf("atmosphere: %g ppm of co2",
			partsAMillion)
	}
	return &Composition{carbonDioxide: partsAMillion / partsInAMillion},
		nil
}

// The parts of the cross-section's formula, each defined by it.
const (
	crossScale   = 24 // defined by rayleigh's cross-section: 24 pi^3
	piPower      = 3  // defined by rayleigh's cross-section: pi cubed
	wavePower    = 4  // defined by rayleigh's cross-section: lambda^4
	squared      = 2  // defined by the formula: a square
	lorentzShift = 2  // defined by the lorentz-lorenz term, n^2 + 2
)

// CrossSection returns molecular Rayleigh scattering area from Bodhaine et
// al.’s equation 22. Wavelengths outside the refractive index model’s supported
// interval return an error.
func (air Composition) CrossSection(wavelength measure.Length) (
	*measure.Area, error) {
	known := measured()
	microns := wavelength.InMetres() / si.Micro(oneMetre)
	if !(microns >= known.shortest && microns <= known.longest) {
		return nil, fmt.Errorf("atmosphere: %g um, outside the %g "+
			"to %g um peck and reeder's index holds over", microns,
			known.shortest, known.longest)
	}
	index := whole + air.refraction(microns)
	square := index * index
	centimetres := wavelength.InMetres() / si.Centi(oneMetre)
	resonance := math.Pow(square-whole, squared)
	crowd := known.molecules * known.molecules *
		math.Pow(square+lorentzShift, squared)
	scattered := crossScale * math.Pow(math.Pi, piPower) * resonance /
		(math.Pow(centimetres, wavePower) * crowd)
	squareCentimetre := si.Centi(oneMetre) * si.Centi(oneMetre)
	area := measure.SquareMetres(scattered * air.king(microns) *
		squareCentimetre)
	return &area, nil
}

// oneMetre is a metre, the unit SI's prefixes are taken of. Defined.
const oneMetre = 1.0

// peckReederScale is the power of ten Peck and Reeder write their index less
// one in, (n - 1) x 10^8. Defined by their formula.
const peckReederScale = 1e-8

// peckReederCO2 is the share of CO2 Peck and Reeder's air holds, 300 ppm, which
// Bodhaine's equation 19 scales from. Defined by their air.
const peckReederCO2 = 300 / partsInAMillion

// refraction returns refractive index minus one using Peck and Reeder’s 300 ppm
// CO2 model, scaled to this composition by Bodhaine et al., equations 18–19.
func (air Composition) refraction(microns float64) float64 {
	known := measured()
	inverse := whole / (microns * microns)
	atThreeHundred := (known.constant +
		known.first/(known.firstPole-inverse) +
		known.second/(known.secondPole-inverse)) * peckReederScale
	more := known.co2Scale * (air.carbonDioxide - peckReederCO2)
	return atThreeHundred * (whole + more)
}

// king computes the mixture’s King correction factor, accounting for molecular
// anisotropy. Nitrogen and oxygen use Bates’s expressions; argon and CO2 are
// volume-weighted as in Bodhaine et al., equations 5, 6, and 23.
func (air Composition) king(microns float64) float64 {
	known := measured()
	inverse := whole / (microns * microns)
	nitrogen := known.nitrogenConstant + known.nitrogenUp*inverse
	oxygen := known.oxygenConstant + known.oxygenUp*inverse +
		known.oxygenCurve*inverse*inverse
	co2 := air.carbonDioxide * percent
	weighed := known.nitrogen*nitrogen + known.oxygen*oxygen +
		known.argon*known.argonKing + co2*known.co2King
	return weighed / (known.nitrogen + known.oxygen + known.argon + co2)
}
