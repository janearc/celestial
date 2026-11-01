package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Ellipse is an error ellipse on the sky, as catalogues give a
// position's: its semi-major and semi-minor axes, and the position angle
// of the major axis, east of north.
type Ellipse struct{ major, minor, angle measure.Angle }

// NewEllipse is the ellipse of two semi-axes and the position angle of
// the major one, east of north. returns nil and error for:
//   - a semi-minor axis below none, or longer than the semi-major
//   - a semi-major axis past the far side of the sky, half a turn
//   - a position angle more than a turn from north
//
// A semi-minor axis of none is an ellipse squashed to a line, which a
// catalogue may give.
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

// Outline is so many points evenly round the ellipse, as offsets east
// and north of its centre, starting at the end of the major axis the
// position angle points to. It refuses fewer than three points, and then
// gives none.
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
