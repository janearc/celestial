package atmosphere

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// The units Bodhaine and the others' equation 25 is written in, each defined by
// CGS and the si.
const (
	// dynesInAPascal is defined: a pascal is ten dynes a square
	// centimetre.
	dynesInAPascal = 10
	// twiceLatitude is defined by list's formula, which takes the cosine
	// of twice the latitude.
	twiceLatitude = 2
)

// DepthAbove returns vertical Rayleigh optical depth using local pressure and
// Bodhaine et al.’s equation 25. Gravity is evaluated at the column’s
// mass-weighted height, following equation 26. Unsupported wavelengths return
// CrossSection’s error.
func (air Composition) DepthAbove(wavelength measure.Length,
	pressure Pressure, latitude measure.Angle,
	height measure.Length) (*float64, error) {
	area, err := air.CrossSection(wavelength)
	if err != nil {
		return nil, err
	}
	known := measured()
	molar := known.molarCO2*air.carbonDioxide + known.molarDry
	column := known.columnSlope*height.InMetres() + known.columnOffset
	dynes := pressure.pascals * dynesInAPascal
	squareCentimetres := area.InSquareMetres() /
		(si.Centi(oneMetre) * si.Centi(oneMetre))
	depth := squareCentimetres * dynes * si.Avogadro /
		(molar * gravityAt(latitude, column))
	return &depth, nil
}

// gravityAt evaluates gravity in cm/s² at latitude and height in metres, using
// List (1968) as reproduced in Bodhaine et al., equations 10–11.
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
