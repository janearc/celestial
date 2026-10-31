package hole_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// notANumber is a ray whose position is not a number.
func notANumber(momentum four) hole.Ray {
	nan := math.NaN()
	return hole.NewRay(hole.NewVec4(nan, nan, nan, nan), vec4(momentum))
}

// describe is a fate in words, for the failures that report one.
func describe(fate hole.Fate) string {
	switch fate {
	case hole.Escaped:
		return "escaped"
	case hole.Captured:
		return "captured"
	case hole.Stuck:
		return "stuck"
	}
	return "unknown"
}

// The test and the anti-test: a ray that is not a number is stuck at once, and handed back as it came.
func TestMarchOfARayThatIsNotANumberIsStuck(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	ray := notANumber(numbers.Geodesic.Momentum)
	end, fate := black.March(ray, numbers.March.Far, numbers.March.Steps)
	if fate != hole.Stuck {
		t.Errorf("a ray that is not a number is %v", describe(fate))
	}
	if end.Momentum() != ray.Momentum() {
		t.Errorf("it was stepped: %v", end.Momentum())
	}
}

// The test and the anti-test: with no steps to take a ray is stuck where it stands; with room to move,
// the same ray is not.
func TestMarchWithNoStepsIsStuck(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	ray := sent(black, numbers.Places.Eye, numbers.March.AwayFromHole)
	end, fate := black.March(ray, numbers.March.Far, none)
	if fate != hole.Stuck || end != ray {
		t.Errorf("with no steps the fate is %v and the ray moved to %v",
			fate, end.Position())
	}
	if _, fate := black.March(ray, numbers.March.Far,
		numbers.March.Steps); fate == hole.Stuck {
		t.Errorf("with steps to take the ray is stuck")
	}
}

// The test and the anti-test: a ray already past the far radius has escaped, and is not stepped.
func TestMarchOfARayAlreadyFarHasEscaped(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	ray := sent(black, numbers.Places.Far, numbers.March.AwayFromHole)
	end, fate := black.March(ray, numbers.March.Far, numbers.March.Steps)
	if fate != hole.Escaped || end != ray {
		t.Errorf("fate %v, and the ray moved to %v", fate, end.Position())
	}
}

// The test and the anti-test: light sent forward from inside the horizon is captured where it stands;
// light sent outward from just outside it is not, however near: it gets
// out.
func TestMarchFromTheEdgeOfTheHorizon(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	for name, from := range map[string]place{
		"inside": numbers.Frame.InsideHorizon, "outside": march.JustOutside,
	} {
		position := vec3(from)
		frame := still.FrameOf(position, still.Normal(position))
		ray := still.Null(position, frame, vec3(march.AwayFromHole))
		end, fate := still.March(ray, march.Far, march.Steps)
		if name == "inside" && (fate != hole.Captured || end != ray) {
			t.Errorf("light sent from inside: fate %v, moved to %v", fate,
				end.Position())
		}
		if name == "outside" && fate != hole.Escaped {
			t.Errorf("light sent outward from just outside: fate %v", fate)
		}
	}
}

// The test and the anti-test: an arriving ray aimed at the hole is found hovering at the horizon in a
// few hundred steps of PaceIn, with H still nought to a percent; a ray far
// from the hole is not hovering.
func TestAnArrivingRayLandsOnTheHorizon(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	eye := vec3(march.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	for _, aim := range []place{march.FallAim, march.FallGraze} {
		ray := still.Arriving(eye, frame, vec3(aim))
		if still.Hovering(ray) {
			t.Errorf("a ray hovers at the eye")
		}
		steps := none
		for !still.Hovering(ray) && steps < march.HoverSteps {
			ray = still.Step(ray, still.PaceIn(ray))
			steps++
		}
		radius := still.Radius(ray.Space())
		if !still.Hovering(ray) ||
			!(radius <= still.Horizon()*march.HoverShare) {
			t.Errorf("aim %v: at %v after %d steps", aim, radius, steps)
		}
		if got := math.Abs(still.Hamiltonian(ray)); !(got <= march.HoverH) {
			t.Errorf("aim %v: H is %v at the horizon", aim, got)
		}
	}
}

// The step of a ray is a part of its radius far from the hole, whatever its
// energy, and shrinks without end as an arriving ray nears the edge.
func TestPaceInShrinksAtTheEdge(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	far := sent(still, numbers.Places.Far, march.AwayFromHole)
	farRadius := still.Radius(far.Space())
	if got := still.PaceIn(far) / farRadius; !within(got, march.PaceShare,
		march.PaceTolerance) {
		t.Errorf("far from the hole a step is %v of the radius", got)
	}
	eye := vec3(march.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	ray := still.Arriving(eye, frame, vec3(march.FallAim))
	before := still.PaceIn(ray) / still.Radius(ray.Space())
	for range march.EdgeSteps {
		ray = still.Step(ray, still.PaceIn(ray))
	}
	after := still.PaceIn(ray) / still.Radius(ray.Space())
	if !(after < before*march.Shrinks) {
		t.Errorf("the step went from %v to %v of the radius", before, after)
	}
}

// The anti-test: light leaving the edge outward is not hovering: it gets out.
func TestLightLeavingTheEdgeIsNotHovering(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	position := vec3(march.JustOutside)
	frame := still.FrameOf(position, still.Normal(position))
	ray := still.Null(position, frame, vec3(march.AwayFromHole))
	if still.Hovering(ray) {
		t.Errorf("light leaving the edge is hovering")
	}
}

// The anti-test: a step is never nought or backward, for a leaving ray or
// an arriving one, whose time runs the other way.
func TestPaceInIsNeverBackward(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	leaving := sent(still, numbers.Places.Eye, march.AwayFromHole)
	eye := vec3(march.FallEye)
	frame := still.FrameOf(eye, still.Static(eye))
	arriving := still.Arriving(eye, frame, vec3(march.FallAim))
	for name, ray := range map[string]hole.Ray{
		"a leaving ray": leaving, "an arriving ray": arriving,
	} {
		if got := still.PaceIn(ray); !(got > none) {
			t.Errorf("%s has a step of %v", name, got)
		}
	}
}

// The anti-test: a ray is not escaped until it is past the far radius. One
// short of it, a march that has a step to take takes it and is not done.
func TestMarchDoesNotEscapeBeforeTheFarRadius(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	ray := sent(still, numbers.Places.Eye, numbers.March.AwayFromHole)
	_, fate := still.March(ray, numbers.Anti.NotYetFar, numbers.Anti.OneStep)
	if fate != hole.Stuck {
		t.Errorf("a ray short of the far radius has fate %v", describe(fate))
	}
}

// The anti-test: a march takes no more steps than it is given. With one
// step it is where one step of PaceIn's size puts the ray, and not done.
func TestMarchTakesNoMoreStepsThanItIsGiven(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	march := numbers.March
	ray := sent(still, numbers.Places.Eye, march.AwayFromHole)
	end, fate := still.March(ray, march.Far, numbers.Anti.OneStep)
	want := partsOf(still.Step(ray, still.PaceIn(ray)).Position())
	got := partsOf(end.Position())
	for place := range got {
		if !within(got[place], want[place], numbers.Anti.PositionsAgree) {
			t.Errorf("place %d is %v after one step, want %v", place,
				got[place], want[place])
		}
	}
	if fate != hole.Stuck {
		t.Errorf("with one step the fate is %v", describe(fate))
	}
}
