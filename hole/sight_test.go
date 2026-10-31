package hole_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/hole"
)

// unit is a direction scaled to length one.
func unit(direction hole.Vec3) hole.Vec3 {
	length := math.Sqrt(direction.AlongX()*direction.AlongX() +
		direction.AlongY()*direction.AlongY() +
		direction.AlongZ()*direction.AlongZ())
	return hole.NewVec3(
		direction.AlongX()/length, direction.AlongY()/length,
		direction.AlongZ()/length,
	)
}

// reversed is a direction turned about.
func reversed(direction hole.Vec3) hole.Vec3 {
	return hole.NewVec3(
		-direction.AlongX(), -direction.AlongY(), -direction.AlongZ(),
	)
}

// An observer sees an arriving ray come from the direction it was made
// from, and the photon it stands for, going the other way, come from the
// opposite one.
func TestSeenIsWhereTheLightComesFrom(t *testing.T) {
	rapid.Check(t, seenIsWhereTheLightComesFrom)
}

// seenIsWhereTheLightComesFrom is the property that test checks.
func seenIsWhereTheLightComesFrom(test *rapid.T) {
	draws := pickedNumbers(test)
	black, position := drawnOutside(test, draws)
	frame := black.FrameOf(position, black.Normal(position))
	aim := drawnAim(test, draws)
	want := unit(aim)
	seen := black.Seen(frame, black.Arriving(position, frame, aim))
	photon := black.Seen(frame, black.Null(position, frame, aim))
	tolerance := draws.Frame.PropertyTolerance
	if !agree(seen, want, tolerance) {
		test.Fatalf("an arriving ray is seen at %v, want %v", seen, want)
	}
	if !agree(photon, reversed(want), tolerance) {
		test.Fatalf("a photon is seen at %v, want %v", photon,
			reversed(want))
	}
}

// agree is whether two directions are the same to a tolerance.
func agree(got, want hole.Vec3, tolerance float64) bool {
	return within(got.AlongX(), want.AlongX(), tolerance) &&
		within(got.AlongY(), want.AlongY(), tolerance) &&
		within(got.AlongZ(), want.AlongZ(), tolerance)
}

// redshiftRoot is the root of one less two M over the radius at a point:
// the rate of a still clock there, by the Schwarzschild metric.
func redshiftRoot(black *hole.Hole, at hole.Vec3) float64 {
	return math.Sqrt(schwarzschildOne - schwarzschildTwo*black.Mass()/
		black.Radius(at))
}

// The anti-test: a ray with no momentum across the observer's axes is seen from no
// direction: nought, and not the nought over nought that normalising it
// would give.
func TestSeenOfARayWithNoDirection(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	// on the disc the metric is flat to the last digit, so the ray's
	// momentum across the axes is nought exactly and not a rounding of it
	at := vec3(numbers.Places.Origin)
	frame := black.FrameOf(at, black.Static(at))
	ray := black.Null(at, frame, vec3(numbers.Places.Origin))
	seen := black.Seen(frame, ray)
	if seen != vec3(numbers.Places.Origin) {
		t.Errorf("a ray with no direction is seen from %v", seen)
	}
}

// The anti-test: where a ray is seen from is not how loud it is. A photon
// with three times the momentum, and an arriving ray with three times, are
// seen from the same direction.
func TestSeenDoesNotFollowTheEnergy(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Fast)
	at := vec3(numbers.Places.Eye)
	frame := black.FrameOf(at, black.Static(at))
	aim := vec3(numbers.Frame.Aim)
	for name, ray := range map[string]hole.Ray{
		"a photon":        black.Null(at, frame, aim),
		"an arriving ray": black.Arriving(at, frame, aim),
	} {
		quiet := black.Seen(frame, ray)
		loud := black.Seen(frame, louder(ray, numbers.Anti.MomentumScale))
		if !agree(loud, quiet, numbers.Frame.Tolerance) {
			t.Errorf("%s is seen at %v, and at %v with %v times the momentum",
				name, quiet, loud, numbers.Anti.MomentumScale)
		}
	}
}

// An arriving ray is the one the observer measures positive energy along,
// and a leaving ray, the photon itself, negative: the two have opposite
// momenta.
func TestArrivingMeasuresPositiveEnergy(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Fast)
	at := vec3(numbers.Places.Eye)
	observer := black.Static(at)
	frame := black.FrameOf(at, observer)
	aim := vec3(numbers.Frame.Aim)
	arriving := hole.Dot(black.Arriving(at, frame, aim).Momentum(), observer)
	leaving := hole.Dot(black.Null(at, frame, aim).Momentum(), observer)
	if !within(arriving, unitEnergy, numbers.Frame.Tolerance) {
		t.Errorf("an arriving ray's energy is %v, want %v", arriving,
			unitEnergy)
	}
	if !(leaving < none) {
		t.Errorf("a leaving ray's energy %v is not below nought", leaving)
	}
}

// Light from a still emitter at a radius reaches a still eye shifted by
// the ratio of the roots of one less two M over the radius: the same
// momentum serves at both ends, since a still observer's velocity has
// nothing but a time part.
func TestShiftBetweenStillObservers(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye, emitter := vec3(numbers.Places.Eye), vec3(numbers.Frame.Emitter)
	photon := vec4(numbers.Frame.Photon)
	got := hole.Shift(photon, still.Static(eye), photon,
		still.Static(emitter))
	want := redshiftRoot(still, emitter) / redshiftRoot(still, eye)
	if !within(got, want, numbers.Frame.Tolerance) {
		t.Errorf("light from %v to %v shifts by %v, want %v",
			numbers.Frame.Emitter, numbers.Places.Eye, got, want)
	}
	if !(got < unitEnergy) {
		t.Errorf("light climbing out of the well arrives %v of its energy",
			got)
	}
}
