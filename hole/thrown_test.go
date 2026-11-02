package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// thrownOut and thrownIn are the signs of a throw along x, away from the
// hole and toward it. Defined by the arithmetic.
const thrownOut, thrownIn = 1, -1

// thrown is a body thrown along x, out for a sign above nought and in for
// one below, from a radius on the x axis of a still hole by a still
// observer there, at the speed that takes it out as far as top.
//
// Its energy is gamma root(1 - 2M/r0), and it turns where that squared is
// 1 - 2M/r, so v^2 = 1 - (1 - 2M/r0) / (1 - 2M/top).
func thrown(black *hole.Hole, from, top, sign float64) hole.Ray {
	at := onX(from)
	twoM := schwarzschildTwo * black.Mass()
	speedSquared := lorentzOne - (schwarzschildOne-twoM/from)/
		(schwarzschildOne-twoM/top)
	gamma := lorentzOne / math.Sqrt(lorentzOne-speedSquared)
	push := sign * math.Sqrt(speedSquared)
	still := black.Static(at)
	out := black.FrameOf(at, still).XAxis()
	velocity := hole.NewVec4(
		gamma*(still.Time()+push*out.Time()),
		gamma*(still.AlongX()+push*out.AlongX()),
		gamma*(still.AlongY()+push*out.AlongY()),
		gamma*(still.AlongZ()+push*out.AlongZ()),
	)
	where := hole.NewVec4(none, at.AlongX(), at.AlongY(), at.AlongZ())
	return hole.NewRay(where, black.Lower(at, velocity))
}

// The far floor is in M: a body thrown out from 8 M of a hole of half the
// unit mass, rising to 100 M, is classed as escaping, since it passes 40
// M, though it falls back.
func TestABodyPastTheFarFloorEscapes(t *testing.T) {
	numbers := pickedNumbers(t)
	given := numbers.Thrown
	light := build(t, given.Hole)
	body := thrown(light, given.From, given.HighTop, thrownOut)
	if got := light.ClassifyRay(body); got != hole.RayEscapes {
		t.Errorf("a body rising to %v is %v, not escaping", given.HighTop,
			got)
	}
}

// The anti-test: one rising to 36 M, past four times where it started but
// short of the floor, is not: it falls back and plunges.
func TestABodyShortOfTheFarFloorPlunges(t *testing.T) {
	numbers := pickedNumbers(t)
	given := numbers.Thrown
	light := build(t, given.Hole)
	body := thrown(light, given.From, given.LowTop, thrownOut)
	if got := light.ClassifyRay(body); got != hole.RayPlunges {
		t.Errorf("a body rising to %v is %v, not plunging", given.LowTop,
			got)
	}
}

// A body on its way out is nearest where it starts, though it will come
// back and fall in: the closest approach stops once it has turned back.
func TestABodyOnItsWayOutIsNearestWhereItStarts(t *testing.T) {
	numbers := pickedNumbers(t)
	given := numbers.Thrown
	light := build(t, given.Hole)
	body := thrown(light, given.From, given.HighTop, thrownOut)
	if got := light.ClosestApproach(body, given.Steps); got != given.From {
		t.Errorf("a body thrown out from %v comes nearest at %v",
			given.From, got)
	}
}

// A ray set at infinity cannot be marched, so its closest approach is not
// a number, as for a ray that is not one; never infinity, which would say
// it came no nearer, and light coming in from infinity does.
func TestARayAtInfinityHasNoClosestApproach(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	far := hole.NewVec4(none, math.Inf(aboveAll), none, none)
	ray := hole.NewRay(far, vec4(numbers.Edges.FromInfinity))
	if got := still.ClosestApproach(ray, numbers.Thrown.Steps); !math.IsNaN(got) {
		t.Errorf("a ray at infinity comes nearest at %v", got)
	}
}

// The anti-test: the same body thrown in comes nearer than it started,
// inside the photon sphere, where it is given up.
func TestABodyOnItsWayInComesNearer(t *testing.T) {
	numbers := pickedNumbers(t)
	given := numbers.Thrown
	light := build(t, given.Hole)
	body := thrown(light, given.From, given.HighTop, thrownIn)
	got := light.ClosestApproach(body, given.Steps)
	if !(got < light.PhotonSphere()) {
		t.Errorf("a body thrown in from %v comes nearest at %v", given.From,
			got)
	}
}
