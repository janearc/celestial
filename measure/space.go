package measure

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
)

// Where things are, which way, and how they move. A place is not a
// length: two places subtract to a Displacement, a place plus a
// Displacement is a place, and two places don't add.

// The numbers the shapes below are built on, each defined.
const (
	// metresInAKilometre is defined by the si's prefix kilo, 10^3.
	metresInAKilometre = 1e3
	// unitLength is a direction's length. Defined: a direction says
	// only which way, so it is one long.
	unitLength = 1.0
	// lightsSpeed is the fastest a velocity may go, as a share of
	// light's. Defined: light's own speed is one.
	lightsSpeed = 1.0
)

// Direction is a way to point: three parts with no unit, a length of
// one, so it says only which way.
type Direction struct{ x, y, z float64 }

// NewDirection is the way three numbers point. It refuses no way at
// all, and any part that is not a number or is endless, and then gives
// no direction.
func NewDirection(x, y, z float64) (*Direction, error) {
	size := math.Sqrt(x*x + y*y + z*z)
	if !(size > nothing) || math.IsInf(size, eitherSign) {
		return nil, refuse("a direction of (%v, %v, %v)", x, y, z)
	}
	return &Direction{x / size, y / size, z / size}, nil
}

// Up is the kit's up, y.
func Up() Direction { return Direction{y: unitLength} }

// Dot is how far this way goes along another: one for the same way,
// nought for square to it, minus one for opposite.
func (way Direction) Dot(other Direction) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// Times is a displacement of a length this way.
func (way Direction) Times(length Length) Displacement {
	size := length.natural
	return Displacement{way.x * size, way.y * size, way.z * size}
}

// Coord is a place: x, y and z, each a position in natural units.
type Coord struct{ x, y, z float64 }

// At is the place at three positions.
func At(x, y, z Length) Coord { return Coord{x.natural, y.natural, z.natural} }

// Origin is the place every frame measures from.
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

// Displacement is how far, and which way, one place is from another.
type Displacement struct{ x, y, z float64 }

// Offset is a displacement of three lengths.
func Offset(x, y, z Length) Displacement {
	return Displacement{x.natural, y.natural, z.natural}
}

// AlongX is the displacement's part along x.
func (by Displacement) AlongX() Length { return Length{by.x} }

// AlongY is the displacement's part along y, the kit's up.
func (by Displacement) AlongY() Length { return Length{by.y} }

// AlongZ is the displacement's part along z.
func (by Displacement) AlongZ() Length { return Length{by.z} }

// Length is how far the displacement goes.
func (by Displacement) Length() Length {
	return Length{math.Sqrt(by.x*by.x + by.y*by.y + by.z*by.z)}
}

// Direction is which way the displacement goes; it refuses one that goes
// nowhere, and then gives no direction.
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

// LightFraction is a speed, as a share of light's: with the speed of
// light one, that is what a speed is, and it has no unit.
type LightFraction struct{ natural float64 }

// InLightFraction is the speed as a share of light's.
func (speed LightFraction) InLightFraction() float64 {
	return speed.natural
}

// InKilometresPerSecond is the speed in km/s.
func (speed LightFraction) InKilometresPerSecond() float64 {
	return speed.natural * si.Speed / metresInAKilometre
}

// Velocity is how fast and which way, each part a share of light's
// speed. Velocities here add the way slow things add; near light's speed
// that is wrong, and nothing here moves that fast.
type Velocity struct{ x, y, z float64 }

// KilometresPerSecond is a velocity of three parts in km/s. It refuses a
// part that is not a number, and a speed at light's or past it, and then
// gives no velocity.
func KilometresPerSecond(x, y, z float64) (*Velocity, error) {
	perLight := metresInAKilometre / si.Speed
	moving := Velocity{x * perLight, y * perLight, z * perLight}
	if speed := moving.Speed().natural; !(speed < lightsSpeed) {
		return nil, refuse("a velocity of (%v, %v, %v) km/s", x, y, z)
	}
	return &moving, nil
}

// AlongX is the velocity's part along x.
func (moving Velocity) AlongX() LightFraction {
	return LightFraction{moving.x}
}

// AlongY is the velocity's part along y, the kit's up.
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

// Sweep is a distance crossed with a velocity: the north of the plane a
// body moves in, and twice the area it sweeps each moment, which
// kepler's second law says an orbit keeps.
type Sweep struct{ x, y, z float64 }

// Size is how much the sweep is: in natural units a length.
func (sweep Sweep) Size() Length {
	squared := sweep.x*sweep.x + sweep.y*sweep.y + sweep.z*sweep.z
	return Length{math.Sqrt(squared)}
}

// North is the way the plane of the motion faces. It refuses a sweep of
// nought, which is motion straight toward or away, with no plane, and
// then gives no direction.
func (sweep Sweep) North() (*Direction, error) {
	return NewDirection(sweep.x, sweep.y, sweep.z)
}

// refuse is a constructor's error, saying what it would not make.
func refuse(format string, about ...any) error {
	return fmt.Errorf("measure: "+format, about...)
}
