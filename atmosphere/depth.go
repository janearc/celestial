package atmosphere

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// The units bodhaine and the others' equation 25 is written in, each
// defined by cgs and the si.
const (
	// dynesInAPascal is defined: a pascal is ten dynes a square
	// centimetre.
	dynesInAPascal = 10
	// twiceLatitude is defined by list's formula, which takes the cosine
	// of twice the latitude.
	twiceLatitude = 2
)

// DepthAbove is the rayleigh optical depth of the air above a place:
// how many times over the light of a wavelength coming straight down is
// scattered on its way, bodhaine and the others' equation 25.
//
// The pressure is the place's. Gravity is taken at the column's
// mass-weighted height, their equation 26. DepthAbove refuses what
// CrossSection refuses, and then gives nothing.
func (air Composition) DepthAbove(wavelength measure.Length,
	pressure Pressure, latitude measure.Angle,
	height measure.Length) (float64, error) {
	area, err := air.CrossSection(wavelength)
	if err != nil {
		return none, err
	}
	known := measured()
	molar := known.molarCO2*air.carbonDioxide + known.molarDry
	column := known.columnSlope*height.InMetres() + known.columnOffset
	dynes := pressure.pascals * dynesInAPascal
	squareCentimetres := area.InSquareMetres() /
		(si.Centi(oneMetre) * si.Centi(oneMetre))
	return squareCentimetres * dynes * si.Avogadro /
		(molar * gravityAt(latitude, column)), nil
}

// gravityAt is the pull of gravity in cm/s^2 at a latitude and a height
// in metres: list's (1968) formula, bodhaine and the others' 10 and 11.
func gravityAt(latitude measure.Angle, metres float64) float64 {
	known := measured()
	twice := math.Cos(twiceLatitude * latitude.InRadians())
	sea := known.seaGravity * (whole - known.seaLatitude*twice +
		known.seaLatitudeSquared*twice*twice)
	first := (known.height + known.heightLatitude*twice) * metres
	second := (known.heightSquared + known.heightSquaredL*twice) *
		metres * metres
	third := (known.heightCubed + known.heightCubedL*twice) *
		metres * metres * metres
	return sea - first + second - third
}
