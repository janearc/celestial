package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// movingObserver is a body dropped from the fall's off point and carried
// the data's proper time: where it is, and its four-velocity there, scaled
// to unit length so that the march's drift is not held against a frame.
func movingObserver(t *testing.T, numbers *picked) (*hole.Hole, hole.Vec3,
	hole.Vec4) {
	t.Helper()
	black := build(t, numbers.Holes.Kerr)
	dropped := black.Dropped(vec3(numbers.Fall.Off))
	body, _ := black.Advance(dropped, numbers.Moving.ProperTime)
	at := body.Space()
	velocity := black.Velocity(body)
	scale := math.Sqrt(timelikeNorm / lengthSquared(black, at, velocity))
	return black, at, hole.NewVec4(velocity.Time()*scale,
		velocity.AlongX()*scale, velocity.AlongY()*scale,
		velocity.AlongZ()*scale)
}

// The frame of an observer moving along every axis at once is orthonormal:
// the observer has length squared minus one, each axis one, and every pair
// is square.
func TestFrameOfAMovingObserverIsOrthonormal(t *testing.T) {
	numbers := pickedNumbers(t)
	black, at, observer := movingObserver(t, numbers)
	axes := axesOf(black.FrameOf(at, observer))
	for row := range dimensions {
		for column := range dimensions {
			want := flatEntry(row, column)
			got := hole.Dot(black.Lower(at, axes[row]), axes[column])
			if !within(got, want, numbers.Frame.Tolerance) {
				t.Errorf("axes %d and %d have inner product %v, want %v",
					row, column, got, want)
			}
		}
	}
}

// The anti-test: a moving observer's x axis has a part along time, along y
// and along z, which a still observer's lacks along y and z.
func TestAMovingObserversXAxisLeansEveryWay(t *testing.T) {
	numbers := pickedNumbers(t)
	black, at, observer := movingObserver(t, numbers)
	moving := black.FrameOf(at, observer).XAxis()
	still := black.FrameOf(at, black.Static(at)).XAxis()
	if !isNumber(moving) || moving.Time() == none ||
		moving.AlongY() == none || moving.AlongZ() == none {
		t.Errorf("a moving observer's x axis is %v", moving)
	}
	if still.AlongY() != none || still.AlongZ() != none {
		t.Errorf("a still observer's x axis is %v", still)
	}
}

// An observer moving along every axis sees a ray it made arriving from a
// direction come from that direction, and the photon it sent toward one
// come from the opposite.
func TestAMovingObserverSeesWhereItAimed(t *testing.T) {
	numbers := pickedNumbers(t)
	black, at, observer := movingObserver(t, numbers)
	frame := black.FrameOf(at, observer)
	aim := vec3(numbers.Moving.Aim)
	want := unit(aim)
	seen := black.Seen(frame, black.Arriving(at, frame, aim))
	if !agree(seen, want, numbers.Frame.Tolerance) {
		t.Errorf("an arriving ray is seen at %v, want %v", seen, want)
	}
	photon := black.Seen(frame, black.Null(at, frame, aim))
	if !agree(photon, reversed(want), numbers.Frame.Tolerance) {
		t.Errorf("a photon is seen at %v, want %v", photon, reversed(want))
	}
}

// The anti-test: a still observer at the same point sees that arriving ray
// elsewhere: the moving one's sky is turned by its motion.
func TestAStillObserverSeesTheMovingOnesRayElsewhere(t *testing.T) {
	numbers := pickedNumbers(t)
	black, at, observer := movingObserver(t, numbers)
	moving := black.FrameOf(at, observer)
	still := black.FrameOf(at, black.Static(at))
	aim := vec3(numbers.Moving.Aim)
	seen := black.Seen(still, black.Arriving(at, moving, aim))
	if math.IsNaN(seen.AlongX()+seen.AlongY()+seen.AlongZ()) ||
		agree(seen, unit(aim), numbers.Frame.Tolerance) {
		t.Errorf("a still observer sees the moving one's ray at %v too",
			seen)
	}
}

// Matter set on a circle round a heavy spinning hole stays on it: carried
// about half way round, its radius has not moved.
func TestCircularMatterRoundAHeavyHoleStaysOnItsCircle(t *testing.T) {
	numbers := pickedNumbers(t)
	heavy := build(t, numbers.Holes.Heavy)
	moving := numbers.Moving
	at := onEquator(heavy, moving.CircleRadius)
	where := hole.NewVec4(none, at.AlongX(), at.AlongY(), at.AlongZ())
	circling := heavy.Lower(at, heavy.Circular(at))
	body := hole.NewRay(where, circling)
	after, _ := heavy.Advance(body, moving.CircleTime)
	got := heavy.Radius(after.Space())
	if !within(got, moving.CircleRadius,
		moving.CircleShare*moving.CircleRadius) {
		t.Errorf("matter on the circle at %v is at %v", moving.CircleRadius,
			got)
	}
	turned := math.Atan2(after.Space().AlongY(), after.Space().AlongX())
	if !(math.Abs(turned) > moving.LeastTurn) {
		t.Errorf("matter on the circle turned %v rad", turned)
	}
}

// The anti-test: matter held still at that point and let go does not stay
// on the circle in the same time: it falls inside it.
func TestStillMatterRoundAHeavyHoleLeavesTheCircle(t *testing.T) {
	numbers := pickedNumbers(t)
	heavy := build(t, numbers.Holes.Heavy)
	moving := numbers.Moving
	at := onEquator(heavy, moving.CircleRadius)
	after, _ := heavy.Advance(heavy.Dropped(at), moving.CircleTime)
	got := heavy.Radius(after.Space())
	if !(got < moving.CircleRadius-moving.CircleShare*moving.CircleRadius) {
		t.Errorf("matter let go at %v is at %v", moving.CircleRadius, got)
	}
}

// Matter on a circle goes round the axis the same way wherever it is on
// the circle: its velocity on the y axis is its velocity on the x axis
// turned a quarter of the way round, as the point is.
func TestCircularTurnsWithThePoint(t *testing.T) {
	numbers := pickedNumbers(t)
	fast := build(t, numbers.Holes.Fast)
	onXAxis := onEquator(fast, numbers.Frame.CircleRadius)
	onYAxis := hole.NewVec3(none, onXAxis.AlongX(), none)
	there, here := fast.Circular(onXAxis), fast.Circular(onYAxis)
	want := hole.NewVec4(there.Time(), -there.AlongY(), there.AlongX(),
		there.AlongZ())
	got, wanted := partsOf(here), partsOf(want)
	for place := range got {
		if !within(got[place], wanted[place], numbers.Frame.Tolerance) {
			t.Errorf("place %d on the y axis is %v, want %v", place,
				got[place], wanted[place])
		}
	}
}

// The anti-test: matter on the circle is not still: on the y axis its
// velocity has a part across the radius, and none along it.
func TestCircularIsNotStillOnTheYAxis(t *testing.T) {
	numbers := pickedNumbers(t)
	fast := build(t, numbers.Holes.Fast)
	onXAxis := onEquator(fast, numbers.Frame.CircleRadius)
	onYAxis := hole.NewVec3(none, onXAxis.AlongX(), none)
	here := fast.Circular(onYAxis)
	if here.AlongX() == none || here.AlongY() != none {
		t.Errorf("matter on the circle at the y axis moves %v", here)
	}
}
