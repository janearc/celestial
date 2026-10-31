package sky

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Ellipse is an error ellipse on the sky, as catalogues give a
// position's: its semi-major and semi-minor axes, and the position angle
// of the major axis, east of north.
type Ellipse struct{ major, minor, angle measure.Angle }

// errEllipse is an ellipse that is not one.
var errEllipse = errors.New("celestial: no such ellipse")

// ErrEllipse is axes that make no ellipse, or an outline of too few
// points to draw one.
func ErrEllipse() error { return errEllipse }

// NewEllipse is the ellipse of two semi-axes and the position angle of
// the major one, east of north. It refuses an axis below nothing or not
// finite, a minor axis longer than the major, and an angle not finite,
// and then gives no ellipse.
func NewEllipse(major, minor, angle measure.Angle) (*Ellipse, error) {
	long, short := major.InRadians(), minor.InRadians()
	turn := angle.InRadians()
	axed := short >= nothing && short <= long && !endless(long)
	if !axed || math.IsNaN(turn) || endless(turn) {
		return nil, fmt.Errorf("%w: axes %v and %v rad at %v rad",
			errEllipse, long, short, turn)
	}
	return &Ellipse{major: major, minor: minor, angle: angle}, nil
}

// endless is whether a number is one of the infinities.
func endless(number float64) bool { return math.IsInf(number, eitherSign) }

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// fewestPoints is the fewest points an outline can have and still close
// round an area. Defined by geometry: a triangle.
const fewestPoints = 3

// Outline is so many points evenly round the ellipse, as offsets east
// and north of its centre, starting at the end of the major axis the
// position angle points to. It refuses fewer than three points, and then
// gives none.
func (ellipse Ellipse) Outline(points int) ([]Offset, error) {
	if points < fewestPoints {
		return nil, fmt.Errorf("%w: an outline of %d points",
			errEllipse, points)
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
