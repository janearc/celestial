package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// onEquator is the point in the equator at a Kerr-schild radius, on the x axis:
// round a spinning hole x is the root of the radius squared plus the spin
// squared.
func onEquator(black *hole.Hole, radius float64) hole.Vec3 {
	spin := black.Spin()
	return hole.NewVec3(math.Sqrt(radius*radius+spin*spin), none, none)
}

// lengthSquared is the squared length of a four-velocity in a hole's
// metric at a point.
func lengthSquared(black *hole.Hole, at hole.Vec3, vector hole.Vec4) float64 {
	return hole.Dot(black.Lower(at, vector), vector)
}

// isNumber is whether every part of a vector is a number.
func isNumber(vector hole.Vec4) bool {
	for _, part := range partsOf(vector) {
		if math.IsNaN(part) {
			return false
		}
	}
	return true
}

// An observer held still has a four-velocity of unit length, and round a
// still hole its time part is one over the root of one less two M over
// the radius.
func TestStaticIsAUnitObserver(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	at := vec3(numbers.Places.Eye)
	velocity := still.Static(at)
	if got := lengthSquared(still, at, velocity); !within(got,
		timelikeNorm, numbers.Frame.Tolerance) {
		t.Errorf("a still observer's length squared is %v", got)
	}
	radius := still.Radius(at)
	want := schwarzschildOne / redshiftRoot(still, at)
	if !within(velocity.Time(), want, numbers.Frame.Tolerance) {
		t.Errorf("a still observer at %v has time part %v, want %v",
			radius, velocity.Time(), want)
	}
}

// The test and the anti-test: nothing is still inside the horizon of a
// hole with no spin, or inside the ergosurface of a spinning one: Static
// answers parts that are not numbers there, and numbers just outside.
func TestStaticIsNotANumberWhereNothingIsStill(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	fast := build(t, numbers.Holes.Fast)
	swallowed := vec3(numbers.Frame.InsideHorizon)
	dragged := vec3(numbers.Frame.Ergosphere)
	far := vec3(numbers.Places.Eye)
	if isNumber(still.Static(swallowed)) {
		t.Error("a still observer inside the horizon is a number")
	}
	if isNumber(fast.Static(dragged)) {
		t.Error("a still observer inside the ergosurface is a number")
	}
	if !isNumber(fast.Static(far)) {
		t.Error("a still observer far from a spinning hole is no number")
	}
}

// Matter on a circle has a four-velocity of unit length, goes round at Kepler's
// rate with the hole's correction, and is prograde.
func TestCircularGoesRoundAtKeplersRate(t *testing.T) {
	numbers := pickedNumbers(t)
	for _, from := range []spec{numbers.Holes.Still, numbers.Holes.Fast} {
		black := build(t, from)
		at := onEquator(black, numbers.Frame.CircleRadius)
		velocity := black.Circular(at)
		if got := lengthSquared(black, at, velocity); !within(got,
			timelikeNorm, numbers.Frame.Tolerance) {
			t.Errorf("spin %v: circular length squared %v", from.Spin, got)
		}
		root := math.Sqrt(black.Mass())
		radius := numbers.Frame.CircleRadius
		rate := root / (radius*math.Sqrt(radius) + black.Spin()*root)
		got := velocity.AlongY() / velocity.Time() / at.AlongX()
		if !within(got, rate, numbers.Frame.Tolerance) {
			t.Errorf("spin %v: angular rate %v, want %v", from.Spin, got,
				rate)
		}
		if !(velocity.AlongY() > none) {
			t.Errorf("spin %v: matter on a circle goes retrograde",
				from.Spin)
		}
	}
}

// The test and the anti-test: inside the prograde photon orbit nothing
// goes round at that rate, and Circular answers parts that are not
// numbers; just outside it does not. Inside the ergosphere, where Static
// fails, it still can.
func TestCircularStopsAtThePhotonOrbit(t *testing.T) {
	numbers := pickedNumbers(t)
	fast := build(t, numbers.Holes.Fast)
	orbit := fast.PhotonOrbit(true)
	inside := onEquator(fast, orbit*numbers.Frame.InsideShare)
	outside := onEquator(fast, orbit*numbers.Frame.OutsideShare)
	if isNumber(fast.Circular(inside)) {
		t.Errorf("matter goes round just inside the photon orbit %v", orbit)
	}
	if !isNumber(fast.Circular(outside)) {
		t.Errorf("matter cannot go round just outside the photon orbit")
	}
	inErgosphere := vec3(numbers.Frame.Ergosphere)
	if !isNumber(fast.Circular(inErgosphere)) || isNumber(
		fast.Static(inErgosphere)) {
		t.Errorf("in the ergosphere Circular exists and Static does not")
	}
}

// The normal observer exists outside the horizon, the ergosphere too, has
// unit length and no angular momentum. That it is not the still observer
// is held by the anti-test below.
func TestNormalIsAUnitObserverWithoutAngularMomentum(t *testing.T) {
	numbers := pickedNumbers(t)
	fast := build(t, numbers.Holes.Fast)
	for _, at := range []place{
		numbers.Frame.Ergosphere, numbers.Places.Eye,
	} {
		position := vec3(at)
		velocity := fast.Normal(position)
		if got := lengthSquared(fast, position, velocity); !within(got,
			timelikeNorm, numbers.Frame.Tolerance) {
			t.Errorf("at %v the normal observer's length squared is %v",
				at, got)
		}
		covector := fast.Lower(position, velocity)
		momentum := hole.NewRay(hole.NewVec4(none, at.AlongX, at.AlongY,
			at.AlongZ), covector)
		if !within(fast.Lz(momentum), none, numbers.Frame.Tolerance) {
			t.Errorf("at %v the normal observer has angular momentum %v",
				at, fast.Lz(momentum))
		}
	}
}

// The anti-test: the normal observer falls: round a still hole, where a
// still observer has no spatial velocity, the normal one has an inward
// one.
func TestNormalIsNotStill(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	at := vec3(numbers.Places.Near)
	if still.Static(at).AlongX() != none {
		t.Fatalf("a still observer moves: %v", still.Static(at))
	}
	if !(still.Normal(at).AlongX() < none) {
		t.Errorf("the normal observer does not fall: %v", still.Normal(at))
	}
}
