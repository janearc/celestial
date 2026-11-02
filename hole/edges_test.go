package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// Numbers the edge tests use that the arithmetic defines.
const (
	// twoSteps is a march of two steps: one to leave a place and one to
	// find where it went. Defined by the test.
	twoSteps = 2
	// aboveAll is the sign math.Inf takes for plus infinity, which a hair
	// past a number goes toward. Defined by package math.
	aboveAll = 1
)

// bodyAt is a body of unit mass at a point at time nought, moving as the
// observer normal to the chart's slices there: it exists inside the
// horizon as well as outside.
func bodyAt(black *hole.Hole, at hole.Vec3) hole.Ray {
	where := hole.NewVec4(none, at.AlongX(), at.AlongY(), at.AlongZ())
	normal := black.Lower(at, black.Normal(at))
	return hole.NewRay(where, normal)
}

// onX is the point on the x axis at a distance.
func onX(distance float64) hole.Vec3 { return hole.NewVec3(distance, none, none) }

// A body exactly at the singularity margin, in M, from a still hole of mass
// two is not within it, and is advanced.
func TestAtTheMarginABodyIsAdvanced(t *testing.T) {
	numbers := pickedNumbers(t)
	edges := numbers.Edges
	double := build(t, edges.Double)
	margin := edges.SingularityMargin * double.Mass()
	body := bodyAt(double, onX(margin))
	moved, _ := double.Advance(body, numbers.Fall.Second)
	if moved.Position() == body.Position() || !isNumber(moved.Position()) {
		t.Errorf("a body at the margin, %v, was not advanced",
			body.Position())
	}
}

// The anti-test: a hair inside the margin it is handed back where it is,
// inside.
func TestAHairInsideTheMarginABodyStops(t *testing.T) {
	numbers := pickedNumbers(t)
	edges := numbers.Edges
	double := build(t, edges.Double)
	margin := edges.SingularityMargin * double.Mass()
	aHairIn := math.Nextafter(margin, none)
	body := bodyAt(double, onX(aHairIn))
	moved, inside := double.Advance(body, numbers.Fall.Second)
	if moved.Position() != body.Position() || !inside {
		t.Errorf("a body a hair inside the margin %v moved to %v, inside %v",
			margin, moved.Position(), inside)
	}
}

// A body exactly on the inner horizon of a spinning hole is not inside it,
// and is advanced.
func TestOnTheInnerHorizonABodyIsAdvanced(t *testing.T) {
	numbers := pickedNumbers(t)
	edges := numbers.Edges
	black := build(t, edges.TwoHorizons)
	at := vec3(edges.OnTheInnerHorizon)
	if got := black.Radius(at); got != edges.InnerHorizon {
		t.Fatalf("the point is at %v, not %v", got, edges.InnerHorizon)
	}
	body := bodyAt(black, at)
	moved, _ := black.Advance(body, numbers.Fall.Second)
	if moved.Position() == body.Position() || !isNumber(moved.Position()) {
		t.Errorf("a body on the inner horizon was not advanced: %v",
			moved.Position())
	}
}

// The anti-test: just inside the inner horizon it is handed back where it
// is, inside.
func TestJustInsideTheInnerHorizonABodyStops(t *testing.T) {
	numbers := pickedNumbers(t)
	edges := numbers.Edges
	black := build(t, edges.TwoHorizons)
	at := vec3(edges.JustInsideInner)
	if got := black.Radius(at); !(got < edges.InnerHorizon) {
		t.Fatalf("the point is at %v, not inside %v", got, edges.InnerHorizon)
	}
	body := bodyAt(black, at)
	moved, inside := black.Advance(body, numbers.Fall.Second)
	if moved.Position() != body.Position() || !inside {
		t.Errorf("a body just inside the inner horizon moved to %v, "+
			"inside %v", moved.Position(), inside)
	}
}

// A body exactly on the horizon, given no proper time, is not inside it:
// inside is inside the horizon itself, to the last digit.
func TestOnTheHorizonABodyIsNotInside(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	at := vec3(numbers.Edges.OnTheHorizon)
	if got := still.Radius(at); got != still.Horizon() {
		t.Fatalf("the point is at %v, not %v", got, still.Horizon())
	}
	if _, inside := still.Advance(bodyAt(still, at), none); inside {
		t.Errorf("a body on the horizon is inside it")
	}
}

// The anti-test: a hair inside the horizon it is inside.
func TestAHairInsideTheHorizonABodyIsInside(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	on := numbers.Edges.OnTheHorizon
	at := onX(math.Nextafter(on.AlongX, none))
	if _, inside := still.Advance(bodyAt(still, at), none); !inside {
		t.Errorf("a body a hair inside the horizon, at %v, is not inside",
			still.Radius(at))
	}
}

// A ray exactly at the far radius is not yet past it: a march steps it on,
// and finds it past.
func TestAtTheFarRadiusARayHasNotEscaped(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	far := numbers.March.Far
	ray := sent(still, place{AlongX: far}, numbers.March.AwayFromHole)
	end, fate := still.March(ray, far, twoSteps)
	if got := still.Radius(end.Space()); fate != hole.Escaped ||
		!(got > far) {
		t.Errorf("a ray from the far radius %v is %v at %v", far,
			describe(fate), got)
	}
}

// The anti-test: a hair past the far radius a ray has escaped where it
// stands.
func TestAHairPastTheFarRadiusARayHasEscaped(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	far := numbers.March.Far
	past := place{AlongX: math.Nextafter(far, math.Inf(aboveAll))}
	ray := sent(still, past, numbers.March.AwayFromHole)
	end, fate := still.March(ray, far, twoSteps)
	if fate != hole.Escaped || end != ray {
		t.Errorf("a ray a hair past the far radius is %v, moved to %v",
			describe(fate), end.Position())
	}
}

// Light running along the horizon of a still hole, an arriving ray with no
// energy whose time runs back, is on the horizon and not inside it: the
// march does not find it hovering, and carries it on.
func TestLightOnTheHorizonIsCarriedOn(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	light := numbers.Edges.HorizonLight
	ray := rayOf(light.Position, light.Momentum)
	end, fate := still.March(ray, numbers.March.Far, numbers.Anti.OneStep)
	if fate != hole.Stuck || end == ray {
		t.Errorf("light on the horizon is %v, moved to %v", describe(fate),
			end.Position())
	}
}

// The anti-test: a hair inside the horizon the same light is found
// hovering, and given up where it stands.
func TestLightAHairInsideTheHorizonIsGivenUp(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	light := numbers.Edges.HorizonLight
	inside := light.Position
	inside.AlongX = math.Nextafter(inside.AlongX, none)
	ray := rayOf(inside, light.Momentum)
	end, fate := still.March(ray, numbers.March.Far, numbers.Anti.OneStep)
	if fate != hole.Captured || end != ray {
		t.Errorf("light a hair inside the horizon is %v, moved to %v",
			describe(fate), end.Position())
	}
}

// A hair off the disc inside the ring, where the radius rounds to nought
// though the point is not on the disc, the metric is flat, as it is on the
// disc, and a step there goes straight.
func TestAHairOffTheDiscIsFlat(t *testing.T) {
	numbers := pickedNumbers(t)
	spinning := build(t, numbers.Holes.Spinning)
	hair := numbers.Edges.Hair
	at := vec4(hair).Space()
	if got := spinning.Radius(at); got != none {
		t.Fatalf("a hair off the disc the radius is %v", got)
	}
	for name, matrix := range map[string]hole.Mat4{
		"metric": spinning.Metric(at), "inverse": spinning.Inverse(at),
	} {
		for row := range dimensions {
			for column := range dimensions {
				if matrix[row][column] != flatEntry(row, column) {
					t.Errorf("the %s is %v at %d, %d", name,
						matrix[row][column], row, column)
				}
			}
		}
	}
	goesStraight(t, spinning, rayOf(hair, numbers.Approach.Disc.Momentum),
		numbers)
}

// goesStraight is whether a step of the disc's size carries a ray along its
// flat velocity, minus the time momentum and the space momentum, and leaves
// its momentum as it was.
func goesStraight(t *testing.T, black *hole.Hole, ray hole.Ray,
	numbers *picked) {
	t.Helper()
	disc := numbers.Approach.Disc
	stepped := black.Step(ray, disc.Step)
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
		t.Errorf("the momentum changed: %v", stepped.Momentum())
	}
}

// The anti-test: further off the disc the radius is not nought, and the
// metric is not flat.
func TestFurtherOffTheDiscIsNotFlat(t *testing.T) {
	numbers := pickedNumbers(t)
	spinning := build(t, numbers.Holes.Spinning)
	at := vec3(numbers.Edges.OffTheHair)
	if got := spinning.Radius(at); !(got > none) {
		t.Errorf("further off the disc the radius is %v", got)
	}
	// the time entry is minus one plus the kernel's scalar, which is above
	// nought wherever the radius is
	metric := spinning.Metric(at)
	if !(metric[timeRow][timeRow] > flatEntry(timeRow, timeRow)) {
		t.Errorf("further off the disc the metric's time entry is %v",
			metric[timeRow][timeRow])
	}
}
