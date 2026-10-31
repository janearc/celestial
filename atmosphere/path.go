package atmosphere

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// Column is the air along a way: how many molecules stand on each square
// metre across it, and whether the way ends on the ground.
type Column struct {
	molecules float64
	grounded  bool
}

// Molecules is how many molecules of air stand on each square metre
// across the way.
func (column Column) Molecules() float64 { return column.molecules }

// Grounded is whether the way ends on the ground, not in space.
func (column Column) Grounded() bool { return column.grounded }

// Along is the air along a straight way from a height above sea level,
// rising at an angle above level: out to the standard's top, or down to
// the ground. Light bends on such a way, and that is not here yet. It
// refuses a height outside the standard, and then gives no column.
func Along(height measure.Length, rise measure.Angle) (*Column, error) {
	start := height.InMetres()
	if !(start >= seaLevel && start <= topHeight) {
		return nil, fmt.Errorf("%w: %g m", errHeight, start)
	}
	known := measured()
	sine := math.Sin(rise.InRadians())
	column := columnOf(known.meanRadius+start, sine, known.pathPieces)
	return &column, nil
}

// Simpson's rule's weights, each defined by the rule: an inner piece at
// an even place counts twice, at an odd place four times, and the sum is
// over three.
const (
	evenWeight  = 2.0 // defined by simpson's rule
	oddWeight   = 4.0 // defined by simpson's rule
	simpsonOver = 3   // defined by simpson's rule
	halves      = 2   // defined by counting: odd and even
	odd         = 1   // defined by counting: what is left of an odd place
	firstPiece  = 1   // defined by simpson's rule: the first inner piece
)

// columnOf is the air along a way from a distance off the earth's
// middle, rising at an angle whose sine is given, summed in so many
// pieces by simpson's rule; the count is even, as the rule needs.
func columnOf(from, sine float64, count int) Column {
	length, grounded := reach(from, sine)
	sum := density(from, sine, none) + density(from, sine, length)
	piece := length / float64(count)
	for place := firstPiece; place < count; place++ {
		weight := evenWeight
		if place%halves == odd {
			weight = oddWeight
		}
		sum += weight * density(from, sine, float64(place)*piece)
	}
	return Column{molecules: sum * piece / simpsonOver, grounded: grounded}
}

// reach is how far a way from a distance off the earth's middle, rising
// at an angle whose sine is given, goes before the ground or the
// standard's top, and whether it is the ground.
func reach(from, sine float64) (float64, bool) {
	radius := measured().meanRadius
	along := from * sine
	ground := along*along - (from*from - radius*radius)
	if sine < none && ground >= none {
		if hit := -along - math.Sqrt(ground); hit > none {
			return hit, true
		}
	}
	top := radius + topHeight
	return -along + math.Sqrt(along*along-from*from+top*top), false
}

// density is the molecules in a cubic metre of air so far along a way:
// bodhaine and the others' count at 288.15 K and 1013.25 mb, scaled by
// the standard's pressure and temperature at the way's height there.
func density(from, sine, along float64) float64 {
	known := measured()
	distance := math.Sqrt(from*from + along*along +
		cosineLaw*from*along*sine)
	height := math.Max(seaLevel, math.Min(distance-known.meanRadius,
		topHeight))
	air, err := Standard(measure.Metres(height))
	if err != nil {
		return none
	}
	ratio := air.Pressure().InPascals() / seaPressure *
		seaTemperature / air.Temperature().InKelvin()
	cubicCentimetre := math.Pow(si.Centi(oneMetre), cubed)
	return known.molecules / cubicCentimetre * ratio
}

// cubed is the power of a cubic centimetre. Defined by the unit.
const cubed = 3

// cosineLaw is the factor of the cross term in the law of cosines, a^2 +
// b^2 + 2ab cos. Defined by the law.
const cosineLaw = 2

// Depth is a column's rayleigh optical depth at a wavelength: how many
// times over its light is scattered on the way. It refuses what
// CrossSection refuses, and then gives nothing.
func (air Composition) Depth(column Column, wavelength measure.Length) (
	float64, error) {
	area, err := air.CrossSection(wavelength)
	if err != nil {
		return none, err
	}
	return area.InSquareMetres() * column.molecules, nil
}

// Through is how much of the light of a wavelength gets through a
// column, by beer and lambert. It refuses what CrossSection refuses, and
// then gives nothing.
func (air Composition) Through(column Column, wavelength measure.Length) (
	float64, error) {
	depth, err := air.Depth(column, wavelength)
	if err != nil {
		return none, err
	}
	return math.Exp(-depth), nil
}
