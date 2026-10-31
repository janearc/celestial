package atmosphere

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// errWavelength is a wavelength the air's refractive index is not known
// at here.
var errWavelength = errors.New("atmosphere: no refractive index there")

// ErrWavelength is a wavelength outside the range peck and reeder's
// refractive index of air holds over.
func ErrWavelength() error { return errWavelength }

// Composition is dry air with some share of co2, its other gases as
// bodhaine and the others' table 1 has them by volume.
type Composition struct{ carbonDioxide float64 }

// The numbers the air's composition is counted in, each defined.
const (
	none            = 0.0   // defined by the scale: no co2 at all
	partsInAMillion = 1e6   // defined by the unit: parts a million
	percent         = 100.0 // defined by the unit: parts a hundred
	whole           = 1.0   // defined: the index of no refraction
)

// DryAir is dry air of so many parts a million of co2. It refuses a share
// below nothing or above all the air, and then gives no air.
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

// CrossSection is how much light of a wavelength one molecule of the air
// scatters, as an area: bodhaine and the others' equation 22. It refuses
// a wavelength outside what the refractive index holds over, and then
// gives no area.
func (air Composition) CrossSection(wavelength measure.Length) (
	*measure.Area, error) {
	known := measured()
	microns := wavelength.InMetres() / si.Micro(oneMetre)
	if !(microns >= known.shortest && microns <= known.longest) {
		return nil, fmt.Errorf("%w: %g um", errWavelength, microns)
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

// oneMetre is a metre, the unit si's prefixes are taken of. Defined.
const oneMetre = 1.0

// peckReederScale is the power of ten peck and reeder write their index
// less one in, (n - 1) x 10^8. Defined by their formula.
const peckReederScale = 1e-8

// peckReederCO2 is the share of co2 peck and reeder's air holds, 300 ppm,
// which bodhaine's equation 19 scales from. Defined by their air.
const peckReederCO2 = 300 / partsInAMillion

// refraction is the air's refractive index less one: peck and reeder's
// for 300 ppm of co2, scaled for this air's, bodhaine and the others'
// equations 18 and 19.
func (air Composition) refraction(microns float64) float64 {
	known := measured()
	inverse := whole / (microns * microns)
	atThreeHundred := (known.constant +
		known.first/(known.firstPole-inverse) +
		known.second/(known.secondPole-inverse)) * peckReederScale
	more := known.co2Scale * (air.carbonDioxide - peckReederCO2)
	return atThreeHundred * (whole + more)
}

// king is the air's king factor, how far its molecules' shape adds to
// what they scatter: bates's for nitrogen and oxygen, argon's and co2's,
// weighed by volume, bodhaine and the others' equations 5, 6 and 23.
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
