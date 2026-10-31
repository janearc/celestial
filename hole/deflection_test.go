package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// arcsecond is an arc second in radians. It is defined by the angle units:
// a degree is a hundred and eightieth of pi, and an arc second a
// three-thousand-six-hundredth of a degree.
const arcsecond = math.Pi / 180 / 3600

// The weak-field deflection of light by a mass M passing at b, to second
// order: 4 M / b + (15 pi / 4) (M / b)^2. Both terms read on 2026-10-31 in
// sereno and de luca's eq. 38 (phys. rev. d 74, 123009, 2006), whose first
// two terms with no spin are 2 r_Sch / b and (15 pi / 16) (r_Sch / b)^2;
// with r_Sch = 2M they are these. game run extract prints the equation.
const (
	// weakFirst is the 4 of 4 M / b. It is defined by eq. 38's 2 r_Sch / b.
	weakFirst = 4
	// weakSecond is the 15 pi / 4 of the second-order term. It is defined
	// by eq. 38's (15 pi / 16) (r_Sch / b)^2.
	weakSecond = 15 * math.Pi / 4
)

// pass is a ray of light sent from far away toward a hole at an impact
// parameter, with the distance it was sent from.
func pass(black *hole.Hole, from, across, aim place) hole.Ray {
	return sent(black, place{
		AlongX: from.AlongX, AlongY: across.AlongY, AlongZ: across.AlongZ,
	}, aim)
}

// distanceFor is how far out a ray at an impact parameter is sent from: far
// enough that the hole is a point.
func distanceFor(sun sunOf, impact float64) float64 {
	return math.Max(sun.Distance, sun.Reach*impact)
}

// A ray grazing the sun bends by 1.75 seconds of arc, Einstein's value.
func TestTheSunBendsLight(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	sun := numbers.Sun
	impact := sun.RadiusKm / sun.MassKm
	distance := distanceFor(sun, impact)
	aim := numbers.Straight.Outward
	ray := pass(black, place{AlongX: -distance}, place{AlongY: impact}, aim)
	angle := black.DeflectionAngle(ray, sun.Beyond*distance, sun.Steps)
	if got := angle / arcsecond; !within(got, sun.Arcseconds, sun.Tolerance) {
		t.Errorf("a ray grazing the sun bends %.3f seconds of arc, want %v",
			got, sun.Arcseconds)
	}
}

// The test and the anti-test: a ray at the shadow's edge is just captured, and just escapes a hair
// further out.
func TestTheShadowCapturesJustInsideItsEdge(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	sun := numbers.Sun
	for name, impact := range map[string]float64{
		"inside": sun.JustInside, "outside": sun.JustOutside,
	} {
		distance := distanceFor(sun, impact)
		aim := numbers.Straight.Outward
		ray := pass(black, place{AlongX: -distance},
			place{AlongY: impact}, aim)
		_, fate := black.March(ray, sun.Beyond*distance, sun.Steps)
		if name == "inside" && fate != hole.Captured {
			t.Errorf("inside the shadow's edge a ray was %v", fate)
		}
		if name == "outside" && fate != hole.Escaped {
			t.Errorf("outside the shadow's edge a ray was %v", fate)
		}
	}
}

// At a hundred M the bending of light is the weak field's, and it is the
// same in any plane through a hole with no spin.
func TestDeflectionIsTheWeakFieldsInAnyPlane(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	weak := numbers.Weak
	want := weakFirst/weak.Impact + weakSecond/(weak.Impact*weak.Impact)
	lowest, highest := math.Inf(positive), math.Inf(-positive)
	for name, across := range map[string]place{
		"the xy plane": weak.XY, "the xz plane": weak.XZ,
		"a tilted plane": weak.Tilted,
	} {
		scaled := place{
			AlongY: across.AlongY * weak.Impact,
			AlongZ: across.AlongZ * weak.Impact,
		}
		aim := numbers.Straight.Outward
		ray := pass(black, place{AlongX: weak.Start}, scaled, aim)
		angle := black.DeflectionAngle(ray, weak.Far, weak.Steps)
		if !within(angle, want, weak.Relative*want) {
			t.Errorf("in %s the ray bends by %v; the weak field says %v",
				name, angle, want)
		}
		lowest, highest = math.Min(lowest, angle), math.Max(highest, angle)
	}
	if highest-lowest > weak.PlanesAgree {
		t.Errorf("the bending differs between planes, from %v to %v",
			lowest, highest)
	}
}

// The test and the anti-test: a ray going straight out is not bent, and a ray that falls in has no
// deflection at all: the answer is not a number, and not the nought that
// would say it went straight.
func TestDeflectionOfRaysThatAreNotBent(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	straight := numbers.Straight
	out := sent(black, straight.From, straight.Outward)
	in := sent(black, straight.From, straight.Inward)
	angle := black.DeflectionAngle(out, straight.Far, straight.Steps)
	if !within(angle, none, straight.Tolerance) {
		t.Errorf("a ray going straight out bends by %v", angle)
	}
	if got := black.DeflectionAngle(in, straight.Far,
		straight.Steps); !math.IsNaN(got) {
		t.Errorf("a ray that falls in has a deflection of %v", got)
	}
}

// The anti-test: a ray that has not escaped for want of steps has no
// deflection either: the answer is not a number, and not the nought of a ray
// that has not yet turned.
func TestDeflectionOfARayOutOfStepsIsNotANumber(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	straight := numbers.Straight
	out := sent(black, straight.From, straight.Outward)
	if got := black.DeflectionAngle(out, straight.Far, none); !math.IsNaN(got) {
		t.Errorf("a ray with no steps has a deflection of %v", got)
	}
}

// The anti-test: spin is felt by light. Passing a hole with no spin, a ray
// bends the same on either side; passing one that turns, it bends less on
// the side the hole turns with than on the side it turns against.
func TestSpinBendsLightDifferentlyOnEitherSide(t *testing.T) {
	numbers := pickedNumbers(t)
	bend := numbers.Anti.SpinBend
	still := build(t, numbers.Holes.Still)
	stillWith := bentPassing(still, numbers, turnedWith)
	stillAgainst := bentPassing(still, numbers, turnedAgainst)
	if !within(stillWith, stillAgainst, bend.SidesAgree) {
		t.Errorf("a hole with no spin bends %v one side and %v the other",
			stillWith, stillAgainst)
	}
	fast := build(t, numbers.Holes.Fast)
	fastWith := bentPassing(fast, numbers, turnedWith)
	fastAgainst := bentPassing(fast, numbers, turnedAgainst)
	if !(fastAgainst-fastWith > bend.Gap) {
		t.Errorf("a spinning hole bends %v with it and %v against it",
			fastWith, fastAgainst)
	}
}

// The anti-test: the deflection is the whole angle between the ray's two
// ways, and not only the size of their cross product. A ray bent by more than
// a right angle reads as more than a right angle, the same as the arccosine of
// the cosine between its two ways gives. The weak-field tests bend a ray by a
// hair, where the size of the cross product and the angle are the same.
func TestDeflectionOfARayBentPastARightAngle(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	bend := numbers.Anti.SpinBend
	large := numbers.Anti.LargeBend
	ray := pass(black, place{AlongX: bend.Start},
		place{AlongY: large.Impact}, numbers.Straight.Outward)
	end, fate := black.March(ray, bend.Far, bend.Steps)
	if fate != hole.Escaped {
		t.Fatalf("the ray's fate is %v", fate)
	}
	setOut, leaves := black.Velocity(ray).Space(), black.Velocity(end).Space()
	dot := setOut.AlongX()*leaves.AlongX() + setOut.AlongY()*leaves.AlongY() +
		setOut.AlongZ()*leaves.AlongZ()
	lengths := math.Sqrt(setOut.AlongX()*setOut.AlongX()+
		setOut.AlongY()*setOut.AlongY()+setOut.AlongZ()*setOut.AlongZ()) *
		math.Sqrt(leaves.AlongX()*leaves.AlongX()+
			leaves.AlongY()*leaves.AlongY()+leaves.AlongZ()*leaves.AlongZ())
	want := math.Acos(dot / lengths)
	got := black.DeflectionAngle(ray, bend.Far, bend.Steps)
	if !(want > math.Pi/rightAngleHalves) {
		t.Fatalf("the ray is bent by %v, under a right angle", want)
	}
	if !within(got, want, large.Agree) {
		t.Errorf("the deflection is %v, and the angle between the ways is %v",
			got, want)
	}
}

// rightAngleHalves is the 2 of a right angle, pi over 2. It is defined by
// the angle: a right angle is half of a half turn.
const rightAngleHalves = 2

// The sides a ray can pass a hole that turns from x toward y, along x.
const (
	// turnedWith is the side of y the hole turns with. It is defined by the
	// hole's sense: the ray's angular momentum about z is above nought.
	turnedWith = -1
	// turnedAgainst is the side of y the hole turns against. It is defined
	// by the hole's sense: the ray's angular momentum about z is below nought.
	turnedAgainst = 1
)

// bentPassing is how far a hole bends a ray sent along x across it, at the
// impact parameter the data picks, on one side of y.
func bentPassing(black *hole.Hole, numbers *picked, side float64) float64 {
	bend := numbers.Anti.SpinBend
	ray := pass(black, place{AlongX: bend.Start},
		place{AlongY: side * bend.Impact}, numbers.Straight.Outward)
	return black.DeflectionAngle(ray, bend.Far, bend.Steps)
}

// Far out, the spin's bend is the published series's. Passing a hole of
// spin 0.9 at 300 M, light against its turning is bent more than light
// with it by sereno and de luca's 8 a m / b^2 + 20 pi a m^2 / b^3, to half
// a percent. TestSpinBendsLightDifferentlyOnEitherSide holds only that
// the sides differ, and passes a spin effect cut to half (a deceit review
// of 8de5148 found that); this does not.
func TestFarOutTheSpinsBendIsThePublishedSeries(t *testing.T) {
	numbers := pickedNumbers(t)
	weak := numbers.WeakSpinBend
	asked := numbers.Holes.Fast
	fast := build(t, asked)
	with := bentFarOut(fast, numbers, turnedWith)
	against := bentFarOut(fast, numbers, turnedAgainst)
	// the mass and spin asked for, not the hole's own account of them,
	// which a hole that kept the wrong spin would give back as right
	mass, spin, impact := asked.Mass, asked.Spin, weak.Impact
	series := spinFirst*spin*mass/(impact*impact) +
		spinSecond*spin*mass*mass/(impact*impact*impact)
	if !within(against-with, series, weak.Share*series) {
		t.Errorf("at %v M the sides differ by %v rad; the series says %v",
			impact, against-with, series)
	}
}

// The series' two terms in the spin, from sereno and de luca's eq. 38
// with r_Sch = 2m: against the turning less with it is twice each of
// their 2 a r_Sch / b^2 and (5 pi / 2) a r_Sch^2 / b^3.
const (
	spinFirst  = 8            // defined by eq. 38: twice 2 times r_Sch's 2
	spinSecond = 20 * math.Pi // defined by eq. 38: twice 5 pi / 2 times 4
)

// bentFarOut is how far a hole bends a ray sent along x across it far
// out, at the weak spin bend's impact parameter, on one side of y.
func bentFarOut(black *hole.Hole, numbers *picked, side float64) float64 {
	weak := numbers.WeakSpinBend
	ray := pass(black, place{AlongX: weak.Start},
		place{AlongY: side * weak.Impact}, numbers.Straight.Outward)
	return black.DeflectionAngle(ray, weak.Far, weak.Steps)
}
