package hole_test

import (
	"testing"

	"github.com/janearc/celestial/hole"
)

// rayOf is a ray the data gives, as a hole's Ray.
func rayOf(position, momentum four) hole.Ray {
	return hole.NewRay(vec4(position), vec4(momentum))
}

// louder is a ray at the same place with its momentum multiplied by a
// factor: the same path, with the factor times the energy.
func louder(ray hole.Ray, factor float64) hole.Ray {
	momentum := ray.Momentum()
	scaled := hole.NewVec4(
		momentum.Time()*factor, momentum.AlongX()*factor,
		momentum.AlongY()*factor, momentum.AlongZ()*factor,
	)
	return hole.NewRay(ray.Position(), scaled)
}

// sent is a ray of light sent from a place by a still observer there, in
// the direction given.
func sent(black *hole.Hole, from, aim place) hole.Ray {
	position := vec3(from)
	frame := black.FrameOf(position, black.Static(position))
	return black.Null(position, frame, vec3(aim))
}

// A ray holds the position and momentum it was made with, and its space is
// the part of the position in space.
func TestRayHoldsWhatItWasMadeWith(t *testing.T) {
	given := pickedNumbers(t).Geodesic
	ray := rayOf(given.Position, given.Momentum)
	if ray.Position() != vec4(given.Position) ||
		ray.Momentum() != vec4(given.Momentum) {
		t.Errorf("the ray is at %v with %v", ray.Position(), ray.Momentum())
	}
	if ray.Space() != vec4(given.Position).Space() {
		t.Errorf("the ray's space is %v", ray.Space())
	}
}

// The energy and the angular momentum are what the momentum says: minus
// its time part, and x times its y part less y times its x part.
func TestEnergyAndAngularMomentumAreWhatTheMomentumSays(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	if got := black.Energy(ray); got != numbers.Geodesic.Energy {
		t.Errorf("energy %v, want %v", got, numbers.Geodesic.Energy)
	}
	if got := black.Lz(ray); got != numbers.Geodesic.Lz {
		t.Errorf("angular momentum %v, want %v", got, numbers.Geodesic.Lz)
	}
}

// The anti-test: anything but the time part of the momentum leaves the energy alone, and
// anything but the x and y parts leaves the angular momentum alone.
func TestEnergyIgnoresSpaceAndLzIgnoresTimeAndZ(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	space := rayOf(numbers.Geodesic.Position, numbers.Geodesic.SpaceChanged)
	time := rayOf(numbers.Geodesic.Position, numbers.Geodesic.TimeChanged)
	if black.Energy(ray) != black.Energy(space) {
		t.Errorf("energy followed a space part: %v then %v",
			black.Energy(ray), black.Energy(space))
	}
	if black.Lz(ray) != black.Lz(time) {
		t.Errorf("angular momentum followed time or z: %v then %v",
			black.Lz(ray), black.Lz(time))
	}
}

// The velocity along a geodesic is the inverse metric on the momentum, and
// the hamiltonian half their product.
func TestVelocityIsTheInverseMetricOnTheMomentum(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	got := black.Velocity(ray)
	want := black.Raise(ray.Space(), ray.Momentum())
	if got != want {
		t.Errorf("velocity %v, want %v", got, want)
	}
	half := hamiltonianHalf * hole.Dot(ray.Momentum(), got)
	if black.Hamiltonian(ray) != half {
		t.Errorf("H is %v, want %v", black.Hamiltonian(ray), half)
	}
}

// With a hole of next to no mass a step carries a ray along its velocity:
// the position moves by the step times the flat velocity, which is minus
// the time momentum and the space momentum.
func TestAStepInFlatSpaceGoesStraight(t *testing.T) {
	numbers := pickedNumbers(t)
	flat := build(t, numbers.Holes.Flat)
	ray := rayOf(numbers.Geodesic.Position, numbers.Geodesic.Momentum)
	step := numbers.Geodesic.Step
	stepped := flat.Step(ray, step)
	momentum, position := ray.Momentum(), ray.Position()
	want := hole.NewVec4(
		position.Time()-momentum.Time()*step,
		position.AlongX()+momentum.AlongX()*step,
		position.AlongY()+momentum.AlongY()*step,
		position.AlongZ()+momentum.AlongZ()*step,
	)
	got := partsOf(stepped.Position())
	for place, part := range partsOf(want) {
		if !within(got[place], part, numbers.Geodesic.FlatTolerance) {
			t.Errorf("place %d is %v after a step, want %v", place,
				got[place], part)
		}
	}
}

// The test and the anti-test: near a hole a step bends: the same step that went straight in flat space
// does not.
func TestAStepNearTheHoleBends(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	ray := sent(black, numbers.Places.Near, numbers.Frame.Aim)
	stepped := black.Step(ray, numbers.Geodesic.Step)
	if stepped.Momentum() == ray.Momentum() {
		t.Errorf("the momentum did not change near the hole")
	}
	flat := build(t, numbers.Holes.Flat)
	same := flat.Step(sent(flat, numbers.Places.Near, numbers.Frame.Aim),
		numbers.Geodesic.Step)
	if !within(same.Momentum().AlongY(), sent(flat, numbers.Places.Near,
		numbers.Frame.Aim).Momentum().AlongY(), numbers.Geodesic.FlatTolerance) {
		t.Errorf("far from any mass the momentum changed: %v", same)
	}
}

// Halving a step and taking two changes where a ray ends by little: the
// runge-kutta step of order four converges.
func TestHalvingTheStepChangesLittle(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Kerr)
	ray := sent(black, numbers.Geodesic.Start, numbers.Geodesic.Aim)
	step := numbers.Geodesic.Step
	whole := black.Step(ray, step)
	halved := black.Step(black.Step(ray, step*midpoint), step*midpoint)
	got, want := partsOf(whole.Position()), partsOf(halved.Position())
	for place := range got {
		if !within(got[place], want[place], numbers.Geodesic.HalvingTolerance) {
			t.Errorf("place %d: %v for one step, %v for two halves", place,
				got[place], want[place])
		}
	}
}

// The anti-test: a step of nothing does not move a ray. The ray comes back
// as it went in, near the hole where the rates are not nought.
func TestAStepOfNothingDoesNotMove(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Kerr)
	ray := sent(black, numbers.Places.Near, numbers.Frame.Aim)
	if stepped := black.Step(ray, none); stepped != ray {
		t.Errorf("a step of nothing took the ray from %v to %v",
			ray.Position(), stepped.Position())
	}
}
