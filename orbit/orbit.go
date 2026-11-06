// Package orbit steps stars through a potential and reports their
// nearest pass to the galactic centre. It is the reference, in 64-bit
// floats, that the kernel in gpu/orbit is held to.
//
// Inside, lengths are kiloparsecs and speeds kilometres per second, so
// the unit of time is one kiloparsec per kilometre per second, about 978
// million years, and a potential's gm is in kiloparsecs times kilometres
// per second squared. The surface is measure's types.
package orbit

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// kiloparsec is the metres in one, and kilometre the metres in one.
var kiloparsec, kilometre = si.Kilo(length.Parsec), si.Kilo(length.Metre)

// unitTime is the seconds in one kiloparsec per kilometre per second.
var unitTime = kiloparsec / kilometre

// The package's own numbers, each defined.
const (
	firstStep = 1   // defined: steps are counted from one
	kicks     = 2   // defined by the leapfrog: a kick split in two
	nothing   = 0.0 // defined: no span, no time
	either    = 0   // defined by math.IsInf: an infinity of either sign
)

// Kiloparsecs is a length of so many kiloparsecs.
func Kiloparsecs(count float64) measure.Length {
	return measure.Metres(count * kiloparsec)
}

// Units is a span of so many units of time, each a kiloparsec per
// kilometre per second.
func Units(count float64) measure.Duration {
	return measure.Seconds(count * unitTime)
}

// vec is three numbers along the galactic axes.
type vec struct{ x, y, z float64 }

// plus is the sum of two vectors.
func (triple vec) plus(other vec) vec {
	return vec{triple.x + other.x, triple.y + other.y, triple.z + other.z}
}

// times is the vector scaled.
func (triple vec) times(by float64) vec {
	return vec{triple.x * by, triple.y * by, triple.z * by}
}

// over is the vector divided.
func (triple vec) over(by float64) vec {
	return vec{triple.x / by, triple.y / by, triple.z / by}
}

// reversed is the vector the other way.
func (triple vec) reversed() vec {
	return vec{-triple.x, -triple.y, -triple.z}
}

// dot is the scalar product of two vectors.
func (triple vec) dot(other vec) float64 {
	return triple.x*other.x + triple.y*other.y + triple.z*other.z
}

// norm is the vector's length.
func (triple vec) norm() float64 { return math.Sqrt(triple.dot(triple)) }

// finite says whether every number is a number, and not infinite.
func (triple vec) finite() bool {
	return finite(triple.x) && finite(triple.y) && finite(triple.z)
}

// finite says whether a number is one, and not infinite.
func finite(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, either)
}

// Potential is what pulls on a star.
type Potential interface {
	// Pull is the acceleration at a place: the change in a star's
	// velocity there over one unit of time.
	Pull(at Place) Pull
}

// Place is a position from the galactic centre.
type Place struct{ at vec }

// PlaceOf is the place x, y and z kiloparsecs from the centre along
// the axes.
func PlaceOf(x, y, z float64) Place { return Place{vec{x, y, z}} }

// Coord is the place as a coordinate from the centre.
func (place Place) Coord() measure.Coord { return coord(place.at) }

// Numbers is the place in kiloparsecs along each axis.
func (place Place) Numbers() (x, y, z float64) {
	return place.at.x, place.at.y, place.at.z
}

// Pull is an acceleration: the change of velocity over one unit of time
// along each galactic axis, in kilometres per second.
type Pull struct{ along vec }

// NewPull is an acceleration of so many kilometres per second over one
// unit of time along each axis.
func NewPull(x, y, z float64) Pull { return Pull{vec{x, y, z}} }

// Along is the acceleration along each axis, in kilometres per second
// over one unit of time.
func (pull Pull) Along() (x, y, z float64) {
	return pull.along.x, pull.along.y, pull.along.z
}

// State is a star's place and motion.
type State struct{ at, moving vec }

// NewState is a star at a place, moving at a velocity.
func NewState(at measure.Coord, moving measure.Velocity) State {
	by := at.Minus(measure.Origin())
	return State{
		at: vec{by.AlongX().InMetres(), by.AlongY().InMetres(),
			by.AlongZ().InMetres()}.over(kiloparsec),
		moving: vec{moving.AlongX().InKilometresPerSecond(),
			moving.AlongY().InKilometresPerSecond(),
			moving.AlongZ().InKilometresPerSecond()},
	}
}

// StateOf is a star at x, y and z kiloparsecs from the centre, moving
// vx, vy and vz kilometres per second along the axes.
func StateOf(x, y, z, vx, vy, vz float64) State {
	return State{vec{x, y, z}, vec{vx, vy, vz}}
}

// Place is where the star is.
func (star State) Place() measure.Coord { return coord(star.at) }

// Motion is how the star moves. Nothing stepped here moves faster than
// light, so the error is for a state built from numbers that do.
func (star State) Motion() (*measure.Velocity, error) {
	return measure.KilometresPerSecond(star.moving.x, star.moving.y,
		star.moving.z)
}

// Numbers is the star's numbers, as the stepping runs on them.
func (star State) Numbers() Numbers { return Numbers(star) }

// Numbers is a star's place in kiloparsecs and its motion in kilometres
// per second along each axis.
type Numbers struct{ at, moving vec }

// At is the place, in kiloparsecs along each axis.
func (numbers Numbers) At() (x, y, z float64) {
	return numbers.at.x, numbers.at.y, numbers.at.z
}

// Moving is the motion, in kilometres per second along each axis.
func (numbers Numbers) Moving() (vx, vy, vz float64) {
	return numbers.moving.x, numbers.moving.y, numbers.moving.z
}

// coord is a place in kiloparsecs as a coordinate.
func coord(at vec) measure.Coord {
	return measure.At(Kiloparsecs(at.x), Kiloparsecs(at.y),
		Kiloparsecs(at.z))
}

// Pass is a star's nearest sampled pass to the centre over a span: how
// near it came, how long ago, and the step's shares of the pass there.
//
// Positions are sampled at each step's end, so the pass is the nearest
// of those, not the orbit's own pericentre; more steps bring the two
// together.
//
// How many steps is set by the nearest pass itself: a step dt resolves
// a pass at a distance r at a speed v under a pull a when v dt over r
// and a dt squared over r are both small, and the leapfrog's errors are
// of the size of the first's square and of the second.
//
// Those two are the step's shares, and the pass carries them from the
// step it was found at, so a caller can tell a resolved pass from a
// plunge the steps missed: a star that fell through the inner parsecs
// between two samples has a pass with large shares.
//
// orbittest.Resolved is the same measure from a point mass's closed
// form, and the tests hold the kernel to this reference where it is
// under a tenth. A run that wants the passes through the inner parsecs
// sizes its steps by the nearest pass it means to see.
type Pass struct{ near, ago, moved, pulled float64 }

// NewPass is a pass so many kiloparsecs from the centre, so many units
// of time ago, with the step's shares of it in the star's motion and in
// the pull.
func NewPass(near, ago, moved, pulled float64) Pass {
	return Pass{near, ago, moved, pulled}
}

// Near is how near the centre the star came.
func (pass Pass) Near() measure.Length { return Kiloparsecs(pass.near) }

// Ago is how long ago it came that near.
func (pass Pass) Ago() measure.Duration { return Units(pass.ago) }

// Numbers is the pass in kiloparsecs and units of time.
func (pass Pass) Numbers() (near, ago float64) {
	return pass.near, pass.ago
}

// Steps is the step's shares of the pass: how far the star moved in one
// step, and how much the pull moved it over one step squared, each as a
// share of the pass's distance. Both small means the steps resolved the
// pass.
//
// A pass through the centre itself has shares that are not finite,
// since no step resolves it.
func (pass Pass) Steps() (moved, pulled float64) {
	return pass.moved, pass.pulled
}

// passAt is the pass at a distance from the centre, so long ago, for a
// star moving so and pulled so there, in steps of a length.
func passAt(near, ago float64, moving, pull vec, dt float64) Pass {
	return Pass{near, ago, moving.norm() * dt / near,
		pull.norm() * dt * dt / near}
}

// Step is the length of one of so many steps over a span, in units of
// time. It refuses a span that is not positive, and no steps.
func Step(span measure.Duration, steps int) (float64, error) {
	if steps < firstStep {
		return nothing, fmt.Errorf("orbit: %d steps", steps)
	}
	long := span.InSeconds() / unitTime
	if !(long > nothing) {
		return nothing, fmt.Errorf("orbit: a span of %g seconds",
			span.InSeconds())
	}
	return long / float64(steps), nil
}

// Back runs a star's orbit back over a span, in so many steps, and
// reports its nearest pass to the centre over them, and its state at the
// start of the span.
//
// It refuses a span that is not positive, no steps, and an orbit that
// ran away, through the centre or past what a float holds.
func Back(field Potential, star State, span measure.Duration,
	steps int) (*Pass, *State, error) {
	dt, err := Step(span, steps)
	if err != nil {
		return nil, nil, err
	}
	at, moving := star.at, star.moving.reversed()
	pull := field.Pull(Place{at}).along
	best := passAt(at.norm(), nothing, moving, pull, dt)
	for step := range steps {
		at, moving, pull = leapfrog(field, at, moving, pull, dt)
		if near := at.norm(); near < best.near {
			ago := float64(step+firstStep) * dt
			best = passAt(near, ago, moving, pull, dt)
		}
	}
	if !at.finite() || !moving.finite() {
		return nil, nil, errors.New("orbit: the orbit ran away")
	}
	return &best, &State{at, moving.reversed()}, nil
}

// leapfrog is one step of the kick, drift, kick scheme: half a step of
// velocity, a step of position, half a step of velocity from the pull
// there. It keeps an orbit's energy bounded however many steps it takes,
// and run backward it retraces itself, which the pull alone does not.
func leapfrog(field Potential, at, moving, pull vec, dt float64) (vec,
	vec, vec) {
	moving = moving.plus(pull.times(dt / kicks))
	at = at.plus(moving.times(dt))
	pull = field.Pull(Place{at}).along
	moving = moving.plus(pull.times(dt / kicks))
	return at, moving, pull
}
