package measure

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
)

// Positions and displacements are different quantities: subtracting positions
// gives a displacement; adding a displacement gives a new position. Adding two
// positions has no defined meaning here.

// The numbers the shapes below are built on, each defined.
const (
	// metresInAKilometre is defined by the SI's prefix kilo, 10^3.
	metresInAKilometre = 1e3
	// unitLength is a direction's length. Defined: a direction says
	// only which way, so it is one long.
	unitLength = 1.0
)

// Direction is a unit vector with dimensionless components.
type Direction struct{ x, y, z float64 }

// NewDirection normalizes three components. A zero vector, NaN, or infinite
// component returns an error and no direction.
//
// Components are scaled by their largest magnitude before squaring, so
// overflow or underflow in their squares does not lose a valid direction.
func NewDirection(x, y, z float64) (*Direction, error) {
	largest := math.Max(math.Abs(x), math.Abs(y))
	largest = math.Max(largest, math.Abs(z))
	if !(largest > nothing) || math.IsInf(largest, eitherSign) {
		return nil, fmt.Errorf("measure: a direction of (%v, %v, %v)",
			x, y, z)
	}
	x, y, z = x/largest, y/largest, z/largest
	size := math.Sqrt(x*x + y*y + z*z)
	return &Direction{x / size, y / size, z / size}, nil
}

// Up returns the positive y direction used by these coordinates.
func Up() Direction { return Direction{y: unitLength} }

// Dot returns the cosine between directions: one when aligned, zero when
// perpendicular, and minus one when opposed.
func (way Direction) Dot(other Direction) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// Times is a displacement of a length this way.
func (way Direction) Times(length Length) Displacement {
	size := length.natural
	return Displacement{way.x * size, way.y * size, way.z * size}
}

// Coord stores a position as three natural-unit coordinates.
type Coord struct{ x, y, z float64 }

// At is the place at three positions.
func At(x, y, z Length) Coord { return Coord{x.natural, y.natural, z.natural} }

// Origin returns the coordinate origin of the caller’s frame.
func Origin() Coord { return Coord{} }

// Minus is how far this place is from another.
func (place Coord) Minus(from Coord) Displacement {
	return Displacement{place.x - from.x, place.y - from.y,
		place.z - from.z}
}

// Plus is the place a displacement from this one.
func (place Coord) Plus(by Displacement) Coord {
	return Coord{place.x + by.x, place.y + by.y, place.z + by.z}
}

// Displacement stores the separation vector between positions.
type Displacement struct{ x, y, z float64 }

// Offset is a displacement of three lengths.
func Offset(x, y, z Length) Displacement {
	return Displacement{x.natural, y.natural, z.natural}
}

// AlongX is the displacement's part along x.
func (by Displacement) AlongX() Length { return Length{by.x} }

// AlongY is the displacement's part along y, which is up.
func (by Displacement) AlongY() Length { return Length{by.y} }

// AlongZ is the displacement's part along z.
func (by Displacement) AlongZ() Length { return Length{by.z} }

// Length is how far the displacement goes.
func (by Displacement) Length() Length {
	return Length{math.Sqrt(by.x*by.x + by.y*by.y + by.z*by.z)}
}

// Direction normalizes a displacement and rejects zero separation.
func (by Displacement) Direction() (*Direction, error) {
	return NewDirection(by.x, by.y, by.z)
}

// Plus is this displacement and another, one after the other.
func (by Displacement) Plus(other Displacement) Displacement {
	return Displacement{by.x + other.x, by.y + other.y, by.z + other.z}
}

// Minus is this displacement less another.
func (by Displacement) Minus(other Displacement) Displacement {
	return Displacement{by.x - other.x, by.y - other.y, by.z - other.z}
}

// Times is the displacement scaled by a share.
func (by Displacement) Times(share float64) Displacement {
	return Displacement{by.x * share, by.y * share, by.z * share}
}

// Dot is the displacement against another: their lengths times how far
// one goes along the other, an area.
func (by Displacement) Dot(other Displacement) Area {
	return Area{by.x*other.x + by.y*other.y + by.z*other.z}
}

// Cross is the sweep of something this far out moving at a velocity:
// the plane of its motion, and twice the area it sweeps each moment.
func (by Displacement) Cross(moving Velocity) Sweep {
	return Sweep{by.y*moving.z - by.z*moving.y,
		by.z*moving.x - by.x*moving.z,
		by.x*moving.y - by.y*moving.x}
}

// LightFraction expresses speed as a fraction of c. With c = 1 this is the
// dimensionless speed used internally.
type LightFraction struct{ natural float64 }

// InLightFraction is the speed as a share of light's.
func (speed LightFraction) InLightFraction() float64 {
	return speed.natural
}

// InKilometresPerSecond is the speed in km/s.
func (speed LightFraction) InKilometresPerSecond() float64 {
	return speed.natural * si.Speed / metresInAKilometre
}

// Velocity stores three components as fractions of c. Arithmetic uses ordinary
// vector addition, not relativistic velocity composition; use it only where
// that approximation is appropriate.
type Velocity struct{ x, y, z float64 }

// KilometresPerSecond constructs a velocity from three km/s components.
// Nonfinite components and speeds at or above c return an error.
//
// The comparison uses c in km/s directly: converting c to a fraction of
// itself first would round its own speed to just below one.
func KilometresPerSecond(x, y, z float64) (*Velocity, error) {
	light := si.Speed / metresInAKilometre
	if speed := math.Sqrt(x*x + y*y + z*z); !(speed < light) {
		return nil, fmt.Errorf("measure: a velocity of (%v, %v, %v) "+
			"km/s", x, y, z)
	}
	perLight := metresInAKilometre / si.Speed
	moving := Velocity{x * perLight, y * perLight, z * perLight}
	return &moving, nil
}

// AlongX is the velocity's part along x.
func (moving Velocity) AlongX() LightFraction {
	return LightFraction{moving.x}
}

// AlongY is the velocity's part along y, which is up.
func (moving Velocity) AlongY() LightFraction {
	return LightFraction{moving.y}
}

// AlongZ is the velocity's part along z.
func (moving Velocity) AlongZ() LightFraction {
	return LightFraction{moving.z}
}

// Speed is how fast, whichever way.
func (moving Velocity) Speed() LightFraction {
	return LightFraction{math.Sqrt(moving.Dot(moving))}
}

// Plus is this velocity and another together.
func (moving Velocity) Plus(other Velocity) Velocity {
	return Velocity{moving.x + other.x, moving.y + other.y,
		moving.z + other.z}
}

// Minus is this velocity seen from something moving at another.
func (moving Velocity) Minus(other Velocity) Velocity {
	return Velocity{moving.x - other.x, moving.y - other.y,
		moving.z - other.z}
}

// Times is the velocity scaled by a share.
func (moving Velocity) Times(share float64) Velocity {
	return Velocity{moving.x * share, moving.y * share, moving.z * share}
}

// Dot is the velocity against another, with no unit.
func (moving Velocity) Dot(other Velocity) float64 {
	return moving.x*other.x + moving.y*other.y + moving.z*other.z
}

// Sweep is displacement crossed with velocity: it sets the orbital plane’s
// normal and twice the area swept per unit time, the quantity preserved by
// Kepler’s second law.
type Sweep struct{ x, y, z float64 }

// Size is how much the sweep is: in natural units a length.
func (sweep Sweep) Size() Length {
	squared := sweep.x*sweep.x + sweep.y*sweep.y + sweep.z*sweep.z
	return Length{math.Sqrt(squared)}
}

// North returns the sweep’s unit normal. Zero sweep means radial motion with no
// defined orbital plane and returns an error.
func (sweep Sweep) North() (*Direction, error) {
	return NewDirection(sweep.x, sweep.y, sweep.z)
}
