package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// The proper time of a fall from rest at a radius to the singularity of a
// hole with no spin is pi / 2 times the root of the radius cubed over 2 M:
// the cycloid of radial infall. It is worked, not read: falling from rest
// at r0, (dr/dtau)^2 = 2M/r - 2M/r0, and r = r0 sin^2 phi turns the fall
// into 2 r0^(3/2) / sqrt(2M) times the integral of sin^2 phi to pi / 2.
//
// That is (pi / 2) sqrt(r0^3 / 2M); a numeric integral of the same fall
// from 10 M, run on 2026-10-31, agrees to ten places. Misner, Thorne and
// Wheeler give it too, but their book is not open and was not read.
const (
	// cycloidQuarter is the pi / 2 of that proper time. It is defined by the
	// cycloid.
	cycloidQuarter = math.Pi / 2
	// cycloidTwo is the 2 of 2 M in it. It is defined by the cycloid.
	cycloidTwo = 2
	// cycloidCube is the exponent of the radius in it. It is defined by the
	// cycloid.
	cycloidCube = 3
)

// Falling drops its body at the radius it is given, on the x axis, at
// rest, as a body of unit mass; and it is Dropped there.
func TestFallingStartsAtTheGivenRadius(t *testing.T) {
	numbers := pickedNumbers(t)
	fall := numbers.Fall
	for _, from := range []spec{
		numbers.Holes.Still, numbers.Holes.Kerr, numbers.Holes.Fast,
	} {
		black := build(t, from)
		body := black.Falling(fall.Radius)
		if got := black.Radius(body.Space()); !within(got, fall.Radius,
			fall.Tolerance) {
			t.Errorf("spin %v: Falling starts at radius %v", from.Spin, got)
		}
		position := body.Position()
		if position.AlongY() != none || position.AlongZ() != none ||
			!(position.AlongX() > none) {
			t.Errorf("spin %v: starts at %v, off the x axis", from.Spin,
				position)
		}
		if body != black.Dropped(body.Space()) {
			t.Errorf("spin %v: Falling is not Dropped on the x axis",
				from.Spin)
		}
		if got := black.Hamiltonian(body); !within(got, fall.BodyH,
			fall.Tolerance) {
			t.Errorf("spin %v: a body has H %v, want %v", from.Spin, got,
				fall.BodyH)
		}
	}
}

// Dropped from rest anywhere has no angular momentum, the dragging apart.
func TestDroppedFromOffTheAxisIsAtRest(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	body := still.Dropped(vec3(numbers.Fall.Off))
	if got := still.Lz(body); !within(got, none, numbers.Fall.Tolerance) {
		t.Errorf("a body dropped from rest has angular momentum %v", got)
	}
	if body.Space() != vec3(numbers.Fall.Off) {
		t.Errorf("it starts at %v", body.Space())
	}
}

// The anti-test: dropping from rest means zero initial spatial velocity, on and
// off the x axis, with and without spin. Zero angular momentum alone is
// insufficient: the normal observer also has it but is already falling.
func TestDroppedFromRestHasNoVelocityInSpace(t *testing.T) {
	numbers := pickedNumbers(t)
	fall := numbers.Fall
	for _, from := range []spec{numbers.Holes.Still, numbers.Holes.Kerr} {
		black := build(t, from)
		for name, body := range map[string]hole.Ray{
			"dropped": black.Dropped(vec3(fall.Off)),
			"falling": black.Falling(fall.Radius),
		} {
			space := black.Velocity(body).Space()
			if !within(space.AlongX(), none, fall.Tolerance) ||
				!within(space.AlongY(), none, fall.Tolerance) ||
				!within(space.AlongZ(), none, fall.Tolerance) {
				t.Errorf("spin %v: a body %s from rest moves at %v", from.Spin,
					name, space)
			}
		}
	}
}

// The test and the anti-test: a camera dropped from ten M reaches the
// horizon of a hole with no spin in finite proper time, and not before.
func TestACameraFallsIn(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	fall := numbers.Fall
	camera := still.Falling(fall.Radius)
	inside := false
	for range fall.Seconds {
		if inside {
			break
		}
		camera, inside = still.Advance(camera, fall.Second)
	}
	if !inside {
		t.Errorf("the camera did not reach the horizon in %d seconds",
			fall.Seconds)
	}
	early, reached := still.Advance(still.Falling(fall.Radius), fall.Early)
	reach := still.Radius(early.Space())
	if reached || !(reach >= fall.EarlyRadius) {
		t.Errorf("after %v of proper time: inside %v at radius %v",
			fall.Early, reached, reach)
	}
}

// From rest at ten M the singularity is pi over two times the root of five
// hundred away in proper time: a little short of it the body is close in,
// and a little past it it has stopped at the margin.
func TestTheFallTakesTheTimeItShould(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	fall := numbers.Fall
	cube := math.Pow(fall.Radius, cycloidCube)
	arrival := cycloidQuarter * math.Sqrt(cube/(cycloidTwo*still.Mass()))
	before, _ := still.Advance(still.Falling(fall.Radius), arrival-fall.Near)
	after, _ := still.Advance(still.Falling(fall.Radius), arrival+fall.Near)
	if got := still.Radius(before.Space()); !(got > fall.BeforeLow &&
		got < fall.BeforeHigh) {
		t.Errorf("a little short of the singularity the radius is %v", got)
	}
	if got := still.Radius(after.Space()); !(got <= fall.StopRadius) {
		t.Errorf("a little past the singularity the radius is %v", got)
	}
}

// A body must stop within the supported chart even if more proper time is
// requested, remaining inside the horizon and unit timelike.
func TestAdvanceStopsWhereTheChartEnds(t *testing.T) {
	numbers := pickedNumbers(t)
	fall := numbers.Fall
	for _, from := range []spec{numbers.Holes.Still, numbers.Holes.Kerr} {
		black := build(t, from)
		for _, start := range fall.Starts {
			for _, properTime := range fall.Times {
				dropped := black.Dropped(vec3(start))
				body, inside := black.Advance(dropped, properTime)
				radius := black.Radius(body.Space())
				if !inside || !(radius < black.Horizon()) {
					t.Errorf("spin %v from %v after %v: inside %v at %v",
						from.Spin, start, properTime, inside, radius)
				}
				drift := math.Abs(black.Hamiltonian(body) - fall.BodyH)
				if !(drift < fall.Drift) {
					t.Errorf("spin %v from %v: H is %v from %v", from.Spin,
						start, drift, fall.BodyH)
				}
				if from.Spin == none && !(radius <= fall.StopRadius) {
					t.Errorf("a still hole's body stopped at %v", radius)
				}
			}
		}
	}
}

// The anti-test: with no proper time, or less, a body stays where it is;
// a body already at the margin is handed back as it is, inside.
func TestAdvanceWithNoTimeDoesNotMove(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	body := still.Falling(numbers.Fall.Radius)
	for _, properTime := range []float64{none, -numbers.Fall.Second} {
		moved, inside := still.Advance(body, properTime)
		if moved != body || inside {
			t.Errorf("proper time %v moved the body to %v, inside %v",
				properTime, moved.Position(), inside)
		}
	}
	at := still.Dropped(vec3(numbers.Fall.AtTheMargin))
	moved, inside := still.Advance(at, numbers.Fall.Second)
	// The momentum there is not a number, since nothing is still inside the
	// horizon, so the places are compared and not the rays
	if moved.Position() != at.Position() || !inside {
		t.Errorf("a body at the margin moved to %v, inside %v",
			moved.Position(), inside)
	}
}

// The anti-test: a body is not given more proper time than it is asked
// for. A proper time shorter than the body's own pace is one step of
// that length, and not one of the pace.
func TestAdvanceDoesNotSpendMoreTimeThanItIsGiven(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	body := still.Falling(numbers.Fall.Radius)
	short := numbers.Anti.ShortProperTime
	moved, inside := still.Advance(body, short)
	want := partsOf(still.Step(body, short).Position())
	got := partsOf(moved.Position())
	for place := range got {
		if !within(got[place], want[place], numbers.Anti.PositionsAgree) {
			t.Errorf("place %d is %v after %v of proper time, want %v", place,
				got[place], short, want[place])
		}
	}
	if inside {
		t.Errorf("a body from ten M is inside the horizon after %v", short)
	}
}

// Light from a hundred M aimed at the hole plunges, aimed away escapes, and
// aimed to miss escapes; matter going round on a circle circulates, and a
// ray that is not a number is called circulating too.
func TestClassifyRay(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	class := numbers.Class
	for _, want := range []struct {
		aim  place
		fate hole.RayClassification
	}{
		{class.AtTheHole, hole.RayPlunges},
		{class.Away, hole.RayEscapes},
		{class.Missing, hole.RayEscapes},
	} {
		ray := sent(still, class.From, want.aim)
		if got := still.ClassifyRay(ray); got != want.fate {
			t.Errorf("light aimed %v is %v, want %v", want.aim, got,
				want.fate)
		}
	}
	at := onEquator(still, class.CircleRadius)
	where := hole.NewVec4(none, at.AlongX(), at.AlongY(), at.AlongZ())
	going := still.Circular(at)
	round := hole.NewRay(where, still.Lower(at, going))
	if got := still.ClassifyRay(round); got != hole.RayCirculates {
		t.Errorf("matter on a circle is %v", got)
	}
	nan := notANumber(numbers.Geodesic.Momentum)
	if got := still.ClassifyRay(nan); got != hole.RayCirculates {
		t.Errorf("a ray that is not a number is %v", got)
	}
}
