package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// nearly is whether two numbers are the same to a tolerance relative to
// the size of the wanted one, or to one if that is smaller.
func nearly(got, want, tolerance float64) bool {
	size := math.Abs(want)
	return within(got, want, tolerance*math.Max(relativeFloor, size))
}

// A ray marched past a hundred M's worth of bending keeps what a geodesic
// keeps: H zero for light, the energy, the angular momentum and the Carter
// constant.
func TestAMarchedRayKeepsWhatItShould(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Kerr)
	keeps := numbers.Keeps
	start := sent(black, numbers.Geodesic.Start, numbers.Geodesic.Aim)
	if got := black.Hamiltonian(start); !within(got, none, keeps.StartH) {
		t.Fatalf("at the start H is %v", got)
	}
	end, fate := black.March(start, keeps.FarRadius, keeps.Steps)
	if fate != hole.Escaped {
		t.Fatalf("the ray's fate is %v", fate)
	}
	if got := black.Hamiltonian(end); !within(got, none, keeps.EndH) {
		t.Errorf("at the end H is %v", got)
	}
	for _, kept := range []kept{
		{"energy", black.Energy(start), black.Energy(end)},
		{"angular momentum", black.Lz(start), black.Lz(end)},
		{"carter constant", black.Carter(start), black.Carter(end)},
	} {
		if !nearly(kept.after, kept.before, keeps.Relative) {
			t.Errorf("%s went from %v to %v", kept.name, kept.before,
				kept.after)
		}
	}
}

// perTime is the time part of a velocity given in space per unit of time.
const perTime = 1 // defined: a velocity in space per unit of time has one unit of time

// kept is a quantity a geodesic keeps, before and after a march.
type kept struct {
	name          string
	before, after float64
}

// The anti-test: a step far too large does not keep H: the tolerance
// above has teeth.
func TestAStepFarTooLargeDoesNotKeepH(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Kerr)
	start := sent(black, numbers.Places.Near, numbers.Frame.Aim)
	stepped := black.Step(start, numbers.Keeps.BigStep)
	if got := black.Hamiltonian(stepped); within(got, none,
		numbers.Keeps.EndH) {
		t.Errorf("a step of %v kept H at %v", numbers.Keeps.BigStep, got)
	}
}

// The Carter constant of a hole with no spin is the square of the angular
// momentum less the square of its part about z.
func TestCarterOfAStillHoleIsTheAngularMomentumLessLz(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	position, momentum := ray.Position(), ray.Momentum()
	across := position.AlongY()*momentum.AlongZ() -
		position.AlongZ()*momentum.AlongY()
	over := position.AlongZ()*momentum.AlongX() -
		position.AlongX()*momentum.AlongZ()
	want := across*across + over*over
	if got := black.Carter(ray); !nearly(got, want, numbers.Keeps.Relative) {
		t.Errorf("carter %v, want %v", got, want)
	}
}

// The anti-test: early capture must agree with following a ray to the horizon.
// Compare both methods across impact parameters on both sides of a spinning
// hole. Schwarzschild-only cases miss this because the no-return photon radius
// is always 3M there.
func TestMarchCapturesOnlyWhatFallsInRoundASpinningHole(t *testing.T) {
	numbers := pickedNumbers(t)
	fast := build(t, numbers.Holes.Fast)
	bend := numbers.Anti.SpinBend
	for _, side := range []float64{turnedWith, turnedAgainst} {
		for _, impact := range numbers.Anti.CaptureSweep.Impacts {
			ray := pass(fast, place{AlongX: bend.Start},
				place{AlongY: side * impact}, numbers.Straight.Outward)
			_, fate := fast.March(ray, bend.Far, bend.Steps)
			falls, settled := fallsIn(fast, ray, bend.Far, bend.Steps)
			if !settled || (fate != hole.Captured && fate != hole.Escaped) {
				t.Fatalf("side %v, impact %v: fate %v, stepped to the end "+
					"settled %v", side, impact, fate, settled)
			}
			if falls != (fate == hole.Captured) {
				t.Errorf("side %v, impact %v: the march says %v, and the ray "+
					"falls in: %v", side, impact, fate, falls)
			}
		}
	}
}

// fallsIn is whether a ray, stepped by its own pace and not given up on, goes
// inside the horizon before it passes the far radius, and whether it did
// either in the steps.
func fallsIn(black *hole.Hole, ray hole.Ray, far float64, steps int) (falls, settled bool) {
	for range steps {
		radius := black.Radius(ray.Space())
		if radius < black.Horizon() {
			return true, true
		}
		if radius > far {
			return false, true
		}
		ray = black.Step(ray, black.PaceIn(ray))
	}
	return false, false
}

// The anti-test: Carter’s constant includes the body’s mass term. A unit-mass
// trajectory moving in polar angle around a spinning hole must conserve it too;
// treating every ray as light misses this term.
func TestCarterOfABodyIsKept(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Kerr)
	given := numbers.Anti.BodyCarter
	at := vec3(given.Position)
	velocity := hole.NewVec4(perTime, given.Velocity.AlongX,
		given.Velocity.AlongY, given.Velocity.AlongZ)
	// The velocity is made a body's four-velocity, whose squared length in
	// the metric is minus one
	lowered := black.Lower(at, velocity)
	length := hole.Dot(lowered, velocity)
	scale := math.Sqrt(timelikeNorm / length)
	unit := hole.NewVec4(scale*velocity.Time(), scale*velocity.AlongX(),
		scale*velocity.AlongY(), scale*velocity.AlongZ())
	where := hole.NewVec4(none, at.AlongX(), at.AlongY(), at.AlongZ())
	start := hole.NewRay(where, black.Lower(at, unit))
	end, inside := black.Advance(start, given.ProperTime)
	if inside {
		t.Fatalf("the body fell inside the horizon, to %v", end.Position())
	}
	if got, want := black.Carter(end), black.Carter(start); !nearly(got, want,
		given.Relative) {
		t.Errorf("the body's carter constant went from %v to %v", want, got)
	}
}

// The test and the anti-test: on the disc inside the ring the radius is zero
// and the Carter constant is not a number; anywhere else it is one.
func TestCarterIsNotANumberOnTheDisc(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	disc := hole.NewRay(vec4(four{}), vec4(numbers.Geodesic.Momentum))
	if got := black.Carter(disc); !math.IsNaN(got) {
		t.Errorf("on the disc the carter constant is %v", got)
	}
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	if got := black.Carter(ray); math.IsNaN(got) {
		t.Errorf("off the disc the carter constant is not a number")
	}
}

// A ray aimed at the hole is captured and one aimed away escapes, from an
// eye near or far, and seventeen degrees off the hole from six M is still
// inside the shadow.
func TestMarchCapturesWhatTheShadowTakes(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	for _, radius := range march.EyeRadii {
		eye := onEquator(still, radius)
		frame := still.FrameOf(eye, still.Static(eye))
		in := still.Arriving(eye, frame, vec3(march.TowardHole))
		out := still.Arriving(eye, frame, vec3(march.AwayFromHole))
		if _, fate := still.March(in, march.Far, march.Steps); fate !=
			hole.Captured {
			t.Errorf("eye at %v, looking at the hole: %v", radius, fate)
		}
		if _, fate := still.March(out, march.Far, march.Steps); fate !=
			hole.Escaped {
			t.Errorf("eye at %v, looking away: %v", radius, fate)
		}
	}
}

// Off the hole by seventeen degrees from six M a ray is inside the shadow;
// by sixty it is outside it.
func TestMarchSeesTheShadowsEdge(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	eye := onEquator(still, march.EdgeEye)
	frame := still.FrameOf(eye, still.Static(eye))
	inside := still.Arriving(eye, frame, vec3(march.Inside))
	outside := still.Arriving(eye, frame, vec3(march.Outside))
	if _, fate := still.March(inside, march.Far, march.Steps); fate !=
		hole.Captured {
		t.Errorf("a ray inside the shadow: %v", fate)
	}
	if _, fate := still.March(outside, march.Far, march.Steps); fate !=
		hole.Escaped {
		t.Errorf("a ray outside the shadow: %v", fate)
	}
}
