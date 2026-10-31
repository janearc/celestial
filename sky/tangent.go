package sky

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Tangent is the plane that touches the sky's sphere at a centre, and
// the gnomonic projection onto it: each direction goes along the line
// from the sphere's middle to where that line meets the plane.
//
// It is the standard projection for small fields, and the one a
// catalogue's positions near a source are drawn and fit on.
//
// East and north on the plane are the standard coordinates, xi and eta,
// each the tangent of an angle; here they are kept as angles, which is
// what they are near the centre, where the projection is used.
//
// The small-angle shortcut, east as the ascension apart times the cosine
// of the centre's declination and north as the declination apart, misses
// the plane by 0.015 arcseconds an arcminute east of a centre at sixty
// degrees north, and by 54 a degree east; the tests measure both.
type Tangent struct {
	centre         Equatorial
	sineNorth      float64
	cosineNorth    float64
	centreEastward float64
}

// NewTangent is the tangent plane at a centre.
func NewTangent(centre Equatorial) Tangent {
	north := centre.declination.InRadians()
	return Tangent{centre: centre, sineNorth: math.Sin(north),
		cosineNorth:    math.Cos(north),
		centreEastward: centre.ascension.InRadians()}
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

// errBehind is a direction a tangent plane cannot reach.
var errBehind = errors.New("celestial: not in front of the plane")

// ErrBehind is a direction a quarter turn or more from a tangent plane's
// centre, which the line from the sphere's middle never takes to the
// plane.
func ErrBehind() error { return errBehind }

// nothing is no amount at all. Defined by the scale.
const nothing = 0.0

// roundingOfOne is the gap between one and the next float, which is how
// far the cosine between two directions can be off. Defined by float64.
// A direction a quarter turn from the centre has a cosine of nothing,
// which rounding can make a little more; under this it is behind.
const roundingOfOne = 0x1p-52

// Onto is where a direction lands on the plane. It refuses a direction a
// quarter turn or more from the centre, to within rounding, and then
// gives no offset.
func (plane Tangent) Onto(at Equatorial) (*Offset, error) {
	apart := at.ascension.InRadians() - plane.centreEastward
	north := at.declination.InRadians()
	sine, cosine := math.Sin(north), math.Cos(north)
	towards := plane.sineNorth*sine + plane.cosineNorth*cosine*
		math.Cos(apart)
	if !(towards > roundingOfOne) {
		return nil, fmt.Errorf("%w: %v and %v", errBehind, at,
			plane.centre)
	}
	east := cosine * math.Sin(apart) / towards
	up := (plane.cosineNorth*sine - plane.sineNorth*cosine*
		math.Cos(apart)) / towards
	return &Offset{east: measure.Radians(east),
		north: measure.Radians(up)}, nil
}

// Back is the direction a place on the plane stands for: Onto, undone.
// Every place on the plane is some direction.
func (plane Tangent) Back(offset Offset) Equatorial {
	east, north := offset.east.InRadians(), offset.north.InRadians()
	across := plane.cosineNorth - north*plane.sineNorth
	eastward := plane.centreEastward + math.Atan2(east, across)
	up := math.Atan2(plane.sineNorth+north*plane.cosineNorth,
		math.Hypot(east, across))
	return Equatorial{ascension: measure.Radians(turned(eastward)),
		declination: measure.Radians(up)}
}

// turned is an angle in radians brought into the first turn, from
// nought up to but not including a whole turn.
func turned(radians float64) float64 {
	turn := measure.Degrees(degreesInATurn).InRadians()
	within := math.Mod(radians, turn)
	if within < nothing {
		within += turn
	}
	return within
}

// degreesInATurn is a whole turn. Defined by the degree.
const degreesInATurn = 360

// Projector takes directions onto a small field about a centre, and
// back: a Tangent is one. A caller that draws or fits positions takes a
// Projector, so its tests stand a mock in (celestial/mocks) and expect
// its calls; the projection is tested once, here.
type Projector interface {
	Onto(at Equatorial) (*Offset, error)
	Back(offset Offset) Equatorial
}

// A Tangent is a Projector; the compiler holds it to that here.
var _ Projector = Tangent{}
