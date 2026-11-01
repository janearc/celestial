package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Tangent is a gnomonic projection: the plane that touches the sky's
// sphere at a centre, onto which each direction is carried along the
// line from the sphere's middle.
//
// It is the usual projection for a small field, and the one a
// catalogue's positions near a source are drawn and fitted on.
//
// East and north on the plane are the standard coordinates, xi and eta.
// Each is the tangent of an angle, and Tangent keeps them as angles,
// which they nearly are close to the centre, where the plane is used.
//
// The small-angle shortcut takes east as the difference in ascension
// times the cosine of the centre's declination, and north as the
// difference in declination.
//
// At a centre sixty degrees north, the shortcut puts a point an
// arcminute east 0.015 arcseconds off the plane, and a point a degree
// east 54 arcseconds off. The tests measure both.
type Tangent struct {
	centre         Equatorial
	sineNorth      float64
	cosineNorth    float64
	centreEastward float64
}

// NewTangent is the tangent plane at a centre. It refuses none: every
// Equatorial is a direction on the sky, since NewEquatorial refuses any
// that is not, and the zero Equatorial is one too.
func NewTangent(centre Equatorial) Tangent {
	north := centre.declination.InRadians()
	sineNorth, cosineNorth := math.Sin(north), math.Cos(north)
	eastward := centre.ascension.InRadians()
	return Tangent{centre: centre, sineNorth: sineNorth,
		cosineNorth: cosineNorth, centreEastward: eastward}
}

// Offset is a place on a tangent plane: how far east and north of the
// centre it is.
type Offset struct{ east, north measure.Angle }

// NewOffset is the place so far east and north of a tangent plane's
// centre.
func NewOffset(east, north measure.Angle) Offset {
	return Offset{east: east, north: north}
}

// East is how far east of the centre the offset is.
func (offset Offset) East() measure.Angle { return offset.east }

// North is how far north of the centre the offset is.
func (offset Offset) North() measure.Angle { return offset.north }

// roundingOfOne is the gap between one and the next float, which is how
// far the cosine between two directions can be off. Defined by float64.
// A direction a quarter turn from the centre has a cosine of nothing,
// which rounding can make a little more; under this it is behind.
const roundingOfOne = 0x1p-52

// Onto is where a direction lands on the plane. returns nil and error
// for a direction a quarter turn or more from the centre, to within
// rounding, which the line from the sphere's middle never carries to
// the plane.
func (plane Tangent) Onto(at Equatorial) (*Offset, error) {
	apart := at.ascension.InRadians() - plane.centreEastward
	north := at.declination.InRadians()
	sine, cosine := math.Sin(north), math.Cos(north)
	towards := plane.sineNorth*sine + plane.cosineNorth*cosine*
		math.Cos(apart)
	if !(towards > roundingOfOne) {
		return nil, fmt.Errorf("sky: %v is a quarter turn or more "+
			"from the plane's centre at %v", at, plane.centre)
	}
	east := cosine * math.Sin(apart) / towards
	up := (plane.cosineNorth*sine - plane.sineNorth*cosine*
		math.Cos(apart)) / towards
	return &Offset{east: measure.Radians(east),
		north: measure.Radians(up)}, nil
}

// Back is the direction a place on the plane stands for, Onto undone.
// Every place on the plane is some direction, so it refuses none.
func (plane Tangent) Back(offset Offset) Equatorial {
	east, north := offset.east.InRadians(), offset.north.InRadians()
	across := plane.cosineNorth - north*plane.sineNorth
	eastward := plane.centreEastward + math.Atan2(east, across)
	up := math.Atan2(plane.sineNorth+north*plane.cosineNorth,
		math.Hypot(east, across))
	turn := measure.Degrees(degreesInATurn).InRadians()
	return Equatorial{ascension: measure.Radians(intoATurn(eastward, turn)),
		declination: measure.Radians(up)}
}

// intoATurn is an angle brought into one turn, from nought up to but not
// including a whole one, in whatever unit turn is given in. math.Mod
// keeps the sign of what it divides, so adding a turn and taking the
// remainder again brings a negative angle round.
func intoATurn(angle, turn float64) float64 {
	return math.Mod(math.Mod(angle, turn)+turn, turn)
}

// degreesInATurn is a whole turn. Defined by the degree.
const degreesInATurn = 360

// Projector carries directions onto a small field about a centre, and
// back. Tangent is one. A caller that draws or fits positions takes a
// Projector, so its own tests can stand the generated mock in; the
// projection itself is tested here, once.
type Projector interface {
	Onto(at Equatorial) (*Offset, error)
	Back(offset Offset) Equatorial
}

// Tangent must stay a Projector: the build fails here if it stops being
// one.
var _ Projector = Tangent{}
