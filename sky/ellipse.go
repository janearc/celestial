package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Ellipse stores a positional error ellipse: major and minor semi-axes and the
// major axis’s position angle east of north.
type Ellipse struct{ major, minor, angle measure.Angle }

// NewEllipse constructs an error ellipse. It rejects a negative minor axis, a
// minor axis longer than the major, a major axis over half a turn, or a
// position angle outside one turn in either direction. A zero minor axis is
// allowed for a catalogue’s degenerate line ellipse.
func NewEllipse(major, minor, angle measure.Angle) (*Ellipse, error) {
	semiMajor, semiMinor := major.InRadians(), minor.InRadians()
	fromNorth := angle.InDegrees()
	if !within(semiMinor, noAxis, semiMajor) {
		return nil, fmt.Errorf("sky: a semi-minor axis of %v rad, "+
			"outside %v to the semi-major's %v", semiMinor, noAxis,
			semiMajor)
	}
	if !within(semiMajor, semiMinor, math.Pi) {
		return nil, fmt.Errorf("sky: a semi-major axis of %v rad, "+
			"past the far side of the sky at %v", semiMajor,
			math.Pi)
	}
	if !within(fromNorth, -degreesInATurn, degreesInATurn) {
		return nil, fmt.Errorf("sky: a position angle of %v degrees, "+
			"more than a turn from north", fromNorth)
	}
	return &Ellipse{major: major, minor: minor, angle: angle}, nil
}

// noAxis is an axis of no length. Defined by the scale.
const noAxis = 0.0

// fewestPoints is the fewest corners a polygon has, defined by
// geometry: three, a triangle, the least outline that encloses an area.
const fewestPoints = 3

// Outline returns evenly spaced points around the ellipse as east and north
// offsets, beginning at the major-axis end selected by the position angle.
// Fewer than three points returns an error.
func (ellipse Ellipse) Outline(points int) ([]Offset, error) {
	if points < fewestPoints {
		return nil, fmt.Errorf("sky: an outline of %d points, fewer "+
			"than %d", points, fewestPoints)
	}
	long, short := ellipse.major.InRadians(), ellipse.minor.InRadians()
	sine, cosine := math.Sincos(ellipse.angle.InRadians())
	turn := measure.Degrees(degreesInATurn).InRadians()
	outline := make([]Offset, points)
	for step := range outline {
		around := float64(step) / float64(points) * turn
		major := long * math.Cos(around)
		minor := short * math.Sin(around)
		east := major*sine + minor*cosine
		north := major*cosine - minor*sine
		outline[step] = Offset{east: measure.Radians(east),
			north: measure.Radians(north)}
	}
	return outline, nil
}
