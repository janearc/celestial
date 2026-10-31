package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// In flat space, as near as a hole of next to no mass is, light from gas
// coming toward the eye at a speed is shifted by one over gamma times one
// less the speed, the relativistic doppler factor; gas going away is
// shifted by the other sign.
func TestShiftIsTheDopplerFactorInFlatSpace(t *testing.T) {
	numbers := pickedNumbers(t)
	flat := build(t, numbers.Holes.Flat)
	at := vec3(numbers.Places.Eye)
	frame := flat.FrameOf(at, flat.Static(at))
	ray := flat.Arriving(at, frame, vec3(numbers.Frame.Outward))
	speed := numbers.Frame.Speed
	gamma := lorentzOne / math.Sqrt(lorentzOne-speed*speed)
	toward := hole.NewVec4(gamma, -gamma*speed, none, none)
	away := hole.NewVec4(gamma, gamma*speed, none, none)
	momentum := ray.Momentum()
	got := hole.Shift(momentum, flat.Static(at), momentum, toward)
	want := lorentzOne / (gamma * (lorentzOne - speed))
	if !within(got, want, numbers.Frame.DopplerTolerance) {
		t.Errorf("gas coming toward the eye shifts by %v, want %v", got, want)
	}
	receding := hole.Shift(momentum, flat.Static(at), momentum, away)
	if !(receding < lorentzOne && got > lorentzOne) {
		t.Errorf("approaching gas shifts by %v and receding by %v: one "+
			"should be blue and the other red", got, receding)
	}
}

// Light from matter going round the hole on the side that comes toward the
// eye is bluer than from the side that goes away.
func TestShiftFavoursTheApproachingSide(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye := vec4(numbers.Frame.FarObserver)
	matter := still.Circular(onEquator(still, numbers.Frame.CircleRadius))
	toward, away := vec4(numbers.Frame.Toward), vec4(numbers.Frame.Away)
	blue := hole.Shift(toward, eye, toward, matter)
	red := hole.Shift(away, eye, away, matter)
	if !(blue > red) {
		t.Errorf("approaching %v, receding %v", blue, red)
	}
}

// Gravity reddens light from near the hole even from rest: a photon from a
// still emitter reaches a far observer with less than its energy, and more
// than none.
func TestShiftReddensLightFromTheWell(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye := vec4(numbers.Frame.FarObserver)
	photon := vec4(numbers.Frame.Photon)
	at := vec3(numbers.Frame.Emitter)
	got := hole.Shift(photon, eye, photon, still.Static(at))
	if !(got < lorentzOne && got > none) {
		t.Errorf("light from rest at %v arrives at %v of its energy",
			numbers.Frame.Emitter, got)
	}
}

// The anti-test: the emitter's momentum is its own, and is not the eye's. A
// photon's energy at the emitter is linear in the momentum there, so the
// same light with a momentum three times as great at the emitter, and the
// same at the eye, arrives at a third of the shift. A shift that took the
// eye's momentum for the emitter's, as the tests above do, would not change.
func TestShiftTakesTheMomentumAtTheEmitterFromTheEmitter(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye := vec4(numbers.Frame.FarObserver)
	matter := still.Circular(onEquator(still, numbers.Frame.CircleRadius))
	toward := vec4(numbers.Frame.Toward)
	scale := numbers.Anti.MomentumScale
	loud := hole.NewVec4(scale*toward.Time(), scale*toward.AlongX(),
		scale*toward.AlongY(), scale*toward.AlongZ())
	same := hole.Shift(toward, eye, toward, matter)
	got := hole.Shift(toward, eye, loud, matter)
	if !within(got*scale, same, numbers.Frame.Tolerance*same) {
		t.Errorf("at %v times the momentum at the emitter the shift is %v, "+
			"not %v over %v", scale, got, same, scale)
	}
}

// The test and the anti-test: an observer measures a photon's energy against itself as one, whatever
// the photon and the observer; against another observer it is not one.
func TestShiftOfAnObserverWithItselfIsOne(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	photon := vec4(numbers.Frame.Photon)
	here := still.Static(vec3(numbers.Places.Eye))
	there := still.Static(vec3(numbers.Frame.Emitter))
	if got := hole.Shift(photon, here, photon, here); !within(got,
		lorentzOne, numbers.Frame.Tolerance) {
		t.Errorf("an observer's shift against itself is %v", got)
	}
	if got := hole.Shift(photon, here, photon, there); within(got,
		lorentzOne, numbers.Frame.Tolerance) {
		t.Errorf("two observers at different radii shift by %v", got)
	}
}
