package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// Round a hole with no spin a ray from far away has the impact parameter it
// is aimed with, in whatever plane it lies.
func TestImpactParameterInAnyPlane(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	weak := numbers.Weak
	aim := numbers.Straight.Outward
	for _, across := range []place{weak.XY, weak.XZ, weak.Tilted} {
		scaled := place{
			AlongY: across.AlongY * weak.Impact,
			AlongZ: across.AlongZ * weak.Impact,
		}
		ray := pass(still, place{AlongX: weak.Start}, scaled, aim)
		if got := still.ImpactParameter(ray); !within(got, weak.Impact,
			numbers.Approach.ImpactTolerance*weak.Impact) {
			t.Errorf("aimed at %v the impact parameter is %v", scaled, got)
		}
	}
}

// The test and the anti-test: a ray sent just inside the shadow's radius
// has an impact parameter below it, and one sent just outside has one
// above it. No ray is marched here; that a ray below is captured, and one
// above is not, is held by the march and deflection tests.
func TestImpactParameterAgainstTheShadow(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	sun := numbers.Sun
	aim := numbers.Straight.Outward
	start := place{AlongX: numbers.Weak.Start}
	inside := pass(still, start, place{AlongY: sun.JustInside}, aim)
	outside := pass(still, start, place{AlongY: sun.JustOutside}, aim)
	if got := still.ImpactParameter(inside); !(got < still.Shadow()) {
		t.Errorf("a ray the shadow takes has impact parameter %v", got)
	}
	if got := still.ImpactParameter(outside); !(got > still.Shadow()) {
		t.Errorf("a ray the shadow lets by has impact parameter %v", got)
	}
}

// Round a spinning hole a ray down the spin axis has no impact parameter,
// and one aimed across it at ten M has ten.
func TestImpactParameterWithSpin(t *testing.T) {
	numbers := pickedNumbers(t)
	spinning := build(t, numbers.Holes.Spinning)
	approach := numbers.Approach
	down := sent(spinning, approach.AxisFrom, approach.AxisAim)
	if got := spinning.ImpactParameter(down); !within(got, none,
		approach.AxisTolerance) {
		t.Errorf("a ray down the axis has impact parameter %v", got)
	}
	aim := numbers.Straight.Outward
	across := pass(spinning, place{AlongX: numbers.Weak.Start},
		place{AlongY: approach.Across}, aim)
	if got := spinning.ImpactParameter(across); !within(got, approach.Across,
		approach.ImpactTolerance*approach.Across) {
		t.Errorf("a ray aimed across at %v has impact parameter %v",
			approach.Across, got)
	}
}

// The test and the anti-test: light with no energy has no finite impact parameter: the answer is
// infinite, not the nought that would say it was aimed at the hole.
func TestImpactParameterOfARayWithNoEnergy(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	given := numbers.Geodesic
	none4 := given.TimeChanged
	none4.Time = none
	dead := rayOf(given.Position, none4)
	if got := still.ImpactParameter(dead); !math.IsInf(got, positive) {
		t.Errorf("a ray with no energy has impact parameter %v", got)
	}
	live := rayOf(given.Position, given.Momentum)
	if got := still.ImpactParameter(live); math.IsInf(got, positive) {
		t.Errorf("a ray with energy has an infinite impact parameter")
	}
}

// The nearest a ray comes round a hole with no spin is the large root of
// r^3 - b^2 r + 2 M b^2, to within the step's overshoot.
func TestClosestApproachIsTheTurningRadius(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	approach := numbers.Approach
	aim := numbers.Straight.Outward
	ray := pass(still, place{AlongX: approach.From},
		place{AlongY: approach.Impact}, aim)
	got := still.ClosestApproach(ray, approach.Steps)
	if !within(got, approach.Turning, approach.Tolerance) {
		t.Errorf("the closest approach is %v, want %v", got, approach.Turning)
	}
	if got < approach.Turning-approach.ExactTolerance {
		t.Errorf("the closest approach %v is under the true %v", got,
			approach.Turning)
	}
}

// A ray on its way out is nearest where it starts; a ray that falls in
// comes nearer than the photon sphere, where it is given up; a ray with no
// steps is nearest where it stands; a ray that is not a number has no
// nearest.
func TestClosestApproachOfRaysThatLeaveFallOrStand(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	approach := numbers.Approach
	out := sent(still, approach.OutFrom, numbers.Straight.Outward)
	if got := still.ClosestApproach(out, approach.Steps); !within(got,
		approach.OutFrom.AlongX, approach.ExactTolerance) {
		t.Errorf("a ray on its way out is nearest at %v", got)
	}
	in := sent(still, approach.InFrom, numbers.Straight.Inward)
	got := still.ClosestApproach(in, approach.Steps)
	if !(got < still.PhotonSphere() && got > approach.GivenUp) {
		t.Errorf("a ray that falls in is given up at %v", got)
	}
	if got := still.ClosestApproach(in, none); !within(got,
		approach.InFrom.AlongX, approach.ExactTolerance) {
		t.Errorf("a ray with no steps is nearest at %v", got)
	}
	nan := notANumber(numbers.Geodesic.Momentum)
	if got := still.ClosestApproach(nan, approach.Steps); !math.IsNaN(got) {
		t.Errorf("a ray that is not a number comes no nearer than %v", got)
	}
}

// On the disc inside the ring the metric is flat, and a step there goes
// straight: the ray stays on the disc, and the rates are the flat ones.
func TestAStepOnTheDiscGoesStraight(t *testing.T) {
	numbers := pickedNumbers(t)
	spinning := build(t, numbers.Holes.Spinning)
	disc := numbers.Approach.Disc
	ray := rayOf(disc.Position, disc.Momentum)
	stepped := spinning.Step(ray, disc.Step)
	momentum, position := ray.Momentum(), ray.Position()
	want := hole.NewVec4(
		position.Time()-momentum.Time()*disc.Step,
		position.AlongX()+momentum.AlongX()*disc.Step,
		position.AlongY()+momentum.AlongY()*disc.Step,
		position.AlongZ()+momentum.AlongZ()*disc.Step,
	)
	got := partsOf(stepped.Position())
	for place, part := range partsOf(want) {
		if !within(got[place], part, disc.Tolerance) {
			t.Errorf("place %d is %v, want %v", place, got[place], part)
		}
	}
	if stepped.Momentum() != ray.Momentum() {
		t.Errorf("the momentum changed on the disc: %v", stepped.Momentum())
	}
}

// The anti-test: the closest approach takes no more steps than it is given.
// A ray falling in, given one step, is nearest where that step leaves it.
func TestClosestApproachTakesNoMoreStepsThanItIsGiven(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	ray := sent(still, numbers.Approach.InFrom, numbers.Straight.Inward)
	stepped := still.Step(ray, still.PaceIn(ray))
	want := still.Radius(stepped.Space())
	got := still.ClosestApproach(ray, numbers.Anti.OneStep)
	if !within(got, want, numbers.Anti.PositionsAgree) {
		t.Errorf("with one step the closest approach is %v, want %v", got, want)
	}
}

// The anti-test: the impact parameter is where a ray would pass, and not
// how loud it is. A ray with three times the momentum goes the same way and
// has the same impact parameter.
func TestImpactParameterDoesNotFollowTheEnergy(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	aim := numbers.Straight.Outward
	ray := pass(still, place{AlongX: numbers.Weak.Start},
		place{AlongY: numbers.Approach.Across}, aim)
	loud := louder(ray, numbers.Anti.MomentumScale)
	got, want := still.ImpactParameter(loud), still.ImpactParameter(ray)
	if !within(got, want, numbers.Approach.ImpactTolerance*want) {
		t.Errorf("at %v times the momentum the impact parameter is %v, not %v",
			numbers.Anti.MomentumScale, got, want)
	}
}
