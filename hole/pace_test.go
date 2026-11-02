package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// overOne is the one of one over a loudness, which makes a ray that much
// quieter. Defined by the arithmetic.
const overOne = 1

// offTheEquator are rays sent from the skew and tilt places round the
// hole of spin 0.9, toward the pace's aim, which crosses the equator.
func offTheEquator(t *testing.T, numbers *picked) (*hole.Hole, []hole.Ray) {
	t.Helper()
	fast := build(t, numbers.Holes.Fast)
	var rays []hole.Ray
	for _, from := range []place{numbers.Places.Skew, numbers.Places.Tilt} {
		rays = append(rays, sent(fast, from, numbers.Pace.Aim))
	}
	return fast, rays
}

// PaceIn is the step in which a ray's time moves by the time share of its
// radius, off the equator and moving across it as on it: the step times
// the time part of the ray's velocity.
func TestPaceInMovesTimeByTheShareOffTheEquator(t *testing.T) {
	numbers := pickedNumbers(t)
	fast, rays := offTheEquator(t, numbers)
	for _, ray := range rays {
		want := numbers.March.PaceShare * fast.Radius(ray.Space())
		got := fast.PaceIn(ray) * math.Abs(fast.Velocity(ray).Time())
		if !within(got, want, numbers.Pace.Relative*want) {
			t.Errorf("from %v a step moves time by %v, want %v",
				ray.Space(), got, want)
		}
	}
}

// The anti-test: near the hole that step is not the time share of the
// radius in the affine parameter: time there does not run at one.
func TestPaceInIsNotTheShareInTheParameterNearTheHole(t *testing.T) {
	numbers := pickedNumbers(t)
	fast, rays := offTheEquator(t, numbers)
	for _, ray := range rays {
		share := numbers.March.PaceShare * fast.Radius(ray.Space())
		if got := fast.PaceIn(ray); math.IsNaN(got) || within(got, share,
			numbers.Pace.Relative*share) {
			t.Errorf("from %v the step is %v, the share itself",
				ray.Space(), got)
		}
	}
}

// arrivingAtTheEdge is the arriving ray aimed at the hole from the fall's
// eye, stepped until it hovers, or the test stops.
func arrivingAtTheEdge(t *testing.T, still *hole.Hole,
	numbers *picked) hole.Ray {
	t.Helper()
	eye := vec3(numbers.March.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	ray := still.Arriving(eye, frame, vec3(numbers.March.FallAim))
	for range numbers.March.HoverSteps {
		if still.Hovering(ray) {
			return ray
		}
		ray = still.Step(ray, still.PaceIn(ray))
	}
	t.Fatalf("the ray did not come to hover in %d steps",
		numbers.March.HoverSteps)
	return ray
}

// Whether a ray is hovering is not how loud it is: at the eye it is not,
// at the edge it is, a hundred times louder or quieter alike.
func TestHoveringDoesNotFollowTheLoudness(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye := vec3(numbers.March.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	far := still.Arriving(eye, frame, vec3(numbers.March.FallAim))
	edge := arrivingAtTheEdge(t, still, numbers)
	loud := numbers.Pace.Loud
	for _, scale := range []float64{loud, overOne / loud} {
		if still.Hovering(louder(far, scale)) {
			t.Errorf("at the eye, %v times as loud, the ray hovers", scale)
		}
		if !still.Hovering(louder(edge, scale)) {
			t.Errorf("at the edge, %v times as loud, the ray does not "+
				"hover", scale)
		}
	}
}

// The anti-test: where a ray is, and not its loudness, says whether it
// hovers: the ray at the eye does not, and the same ray at the edge does.
func TestHoveringFollowsWhereTheRayIs(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	eye := vec3(numbers.March.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	far := still.Arriving(eye, frame, vec3(numbers.March.FallAim))
	if still.Hovering(far) {
		t.Errorf("the ray hovers at the eye")
	}
	if edge := arrivingAtTheEdge(t, still, numbers); !still.Hovering(edge) {
		t.Errorf("the ray does not hover at the edge")
	}
}
