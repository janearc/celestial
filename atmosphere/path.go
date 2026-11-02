package atmosphere

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// Column stores the molecular column density along a path and whether that path
// reaches the ground.
type Column struct {
	molecules float64
	grounded  bool
}

// Molecules returns the number of molecules per square metre perpendicular to
// the path.
func (column Column) Molecules() float64 { return column.molecules }

// Grounded reports whether the path ends at the ground rather than space.
func (column Column) Grounded() bool { return column.grounded }

// Along integrates air along a straight path from height at elevation rise,
// ending at the ground or the standard atmosphere’s top. Refraction is not
// modelled. Heights outside the standard return an error and no column.
func Along(height measure.Length, rise measure.Angle) (*Column, error) {
	start := height.InMetres()
	if !(start >= seaLevel && start <= topHeight) {
		return nil, fmt.Errorf("atmosphere: %g m, outside the "+
			"standard's %g to %g m", start, seaLevel, topHeight)
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

// columnOf integrates molecular density from geocentric distance from at the
// elevation whose sine is supplied. Composite Simpson integration requires an
// even piece count.
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

// reach returns distance to the first ground intersection or the standard’s
// top, and whether ground was reached.
//
// A downward path starting on the ground ends immediately. Handling that
// case explicitly avoids a result determined by rounding.
func reach(from, sine float64) (float64, bool) {
	radius := measured().meanRadius
	if sine < none && from <= radius {
		return none, true
	}
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

// density computes molecules per cubic metre at a path sample. Bodhaine et
// al.’s reference count at 288.15 K and 1013.25 mb is scaled by the standard’s
// local pressure and temperature.
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

// Depth returns Rayleigh optical depth through a column at the given
// wavelength. Unsupported wavelengths return CrossSection’s error.
func (air Composition) Depth(column Column, wavelength measure.Length) (
	*float64, error) {
	area, err := air.CrossSection(wavelength)
	if err != nil {
		return nil, err
	}
	depth := area.InSquareMetres() * column.molecules
	return &depth, nil
}

// Through returns Beer–Lambert transmission through the column, exp(-optical
// depth). Unsupported wavelengths return CrossSection’s error.
func (air Composition) Through(column Column, wavelength measure.Length) (
	*float64, error) {
	depth, err := air.Depth(column, wavelength)
	if err != nil {
		return nil, err
	}
	through := math.Exp(-*depth)
	return &through, nil
}
