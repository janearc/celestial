package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Tangent is the gnomonic plane touching the sky sphere at a chosen centre.
// Projection follows the line from the sphere’s centre, as used to fit
// catalogue positions in a small field.
//
// East and north are standard coordinates xi and eta. They are stored as Angle
// values, but are tangents rather than angular separations; the distinction
// matters away from the centre.
//
// The small-angle shortcut uses ΔRA*cos(dec) and Δdec. At declination 60
// degrees it misses the exact plane by 0.015 arcseconds for a point one
// arcminute east, and by 54 arcseconds at one degree. Tests retain both
// comparisons.
type Tangent struct {
	centre         Equatorial
	sineNorth      float64
	cosineNorth    float64
	centreEastward float64
}

// NewTangent constructs the plane at a validated equatorial centre. Every
// Equatorial, including its zero value, is a valid direction.
func NewTangent(centre Equatorial) Tangent {
	north := centre.declination.InRadians()
	sineNorth, cosineNorth := math.Sin(north), math.Cos(north)
	eastward := centre.ascension.InRadians()
	return Tangent{centre: centre, sineNorth: sineNorth,
		cosineNorth: cosineNorth, centreEastward: eastward}
}

// Offset stores east and north coordinates on a tangent plane.
type Offset struct{ east, north measure.Angle }

// NewOffset constructs east and north tangent-plane coordinates.
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

// Onto projects a direction onto the tangent plane. Directions a quarter-turn
// or more from the centre have no finite projection and return an error,
// allowing for floating-point rounding at the boundary.
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

// Back reverses Onto. Every finite point on the plane corresponds to a sky
// direction.
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

// intoATurn is an angle brought into one turn, from zero up to but not
// including a whole one, in whatever unit turn is given in. math.Mod keeps the
// sign of what it divides, so adding a turn and taking the remainder again
// brings a negative angle round.
func intoATurn(angle, turn float64) float64 {
	return math.Mod(math.Mod(angle, turn)+turn, turn)
}

// degreesInATurn is a whole turn. Defined by the degree.
const degreesInATurn = 360

// Projector maps sky directions onto a field around a centre and back. Tangent
// implements it; callers can substitute generated mocks when testing catalogue
// fitting or plotting independently of projection.
type Projector interface {
	Onto(at Equatorial) (*Offset, error)
	Back(offset Offset) Equatorial
}

// Tangent must stay a Projector: the build fails here if it stops being
// one.
var _ Projector = Tangent{}
