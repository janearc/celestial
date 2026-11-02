package hole_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/hole"
)

// axesOf are a frame's four vectors in order, the observer first.
func axesOf(frame hole.Frame) [dimensions]hole.Vec4 {
	return [dimensions]hole.Vec4{
		frame.Observer(), frame.XAxis(), frame.YAxis(), frame.ZAxis(),
	}
}

// Whatever the hole and the point, the frame of the normal observer there
// is orthonormal: the observer has length squared minus one, each axis has
// one, and every pair is square.
func TestFrameOfIsOrthonormal(t *testing.T) {
	rapid.Check(t, frameOfIsOrthonormal)
}

// frameOfIsOrthonormal is the property that test checks.
func frameOfIsOrthonormal(test *rapid.T) {
	draws := pickedNumbers(test)
	black, position := drawnOutside(test, draws)
	frame := black.FrameOf(position, black.Normal(position))
	axes := axesOf(frame)
	for row := range dimensions {
		for column := range dimensions {
			want := flatEntry(row, column)
			got := hole.Dot(black.Lower(position, axes[row]), axes[column])
			if !within(got, want, draws.Frame.PropertyTolerance) {
				test.Fatalf("axes %d and %d have inner product %v, want %v",
					row, column, got, want)
			}
		}
	}
}

// A frame is the observer it was made for, handed back as it came.
func TestFrameOfKeepsItsObserver(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	at := vec3(numbers.Places.Eye)
	observer := black.Static(at)
	if got := black.FrameOf(at, observer).Observer(); got != observer {
		t.Errorf("the frame's observer is %v, not %v", got, observer)
	}
}

// The anti-test: near a hole the axes are not the coordinate axes: gram
// and schmidt carry time into them. A frame that only copied the
// coordinate axes would be caught here.
func TestFrameOfAxesAreNotTheCoordinateAxesNearTheHole(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	at := vec3(numbers.Places.Near)
	frame := black.FrameOf(at, black.Normal(at))
	if frame.XAxis().Time() == none {
		t.Errorf("the x axis has no time part: %v", frame.XAxis())
	}
}

// A ray of light is null whatever it is aimed at, and the observer whose
// frame aimed it measures its energy to be one.
func TestNullIsNullWithUnitEnergy(t *testing.T) {
	rapid.Check(t, nullIsNullWithUnitEnergy)
}

// nullIsNullWithUnitEnergy is the property that test checks.
func nullIsNullWithUnitEnergy(test *rapid.T) {
	draws := pickedNumbers(test)
	black, position := drawnOutside(test, draws)
	observer := black.Normal(position)
	frame := black.FrameOf(position, observer)
	aim := drawnAim(test, draws)
	ray := black.Null(position, frame, aim)
	if got := black.Hamiltonian(ray); !within(got, none,
		draws.Frame.PropertyTolerance) {
		test.Fatalf("the ray is not null: H is %v", got)
	}
	energy := -hole.Dot(ray.Momentum(), observer)
	if !within(energy, unitEnergy, draws.Frame.PropertyTolerance) {
		test.Fatalf("the observer measures energy %v, not %v", energy,
			unitEnergy)
	}
}

// The anti-test: with no direction to aim at a ray has the observer's
// own momentum, which is timelike and not null.
func TestNullWithNoDirectionIsNotNull(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	at := vec3(numbers.Places.Eye)
	frame := black.FrameOf(at, black.Static(at))
	ray := black.Null(at, frame, vec3(numbers.Places.Origin))
	if got := black.Hamiltonian(ray); within(got, none,
		numbers.Frame.Tolerance) {
		t.Errorf("a ray with no direction is null: H is %v", got)
	}
}

// A ray starts at time zero, at the point it was sent from.
func TestNullStartsAtTimeNought(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	at := vec3(numbers.Places.Eye)
	frame := black.FrameOf(at, black.Static(at))
	ray := black.Null(at, frame, vec3(numbers.Frame.Aim))
	if ray.Position().Time() != none || ray.Space() != at {
		t.Errorf("the ray starts at %v, not time nought at %v",
			ray.Position(), at)
	}
}

// The anti-test: only the way a direction points matters, not its length.
func TestNullTakesNoNoticeOfTheLengthOfADirection(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	at := vec3(numbers.Places.Eye)
	frame := black.FrameOf(at, black.Static(at))
	aim := numbers.Frame.Aim
	longer := place{
		aim.AlongX * numbers.Frame.Stretch, aim.AlongY * numbers.Frame.Stretch,
		aim.AlongZ * numbers.Frame.Stretch,
	}
	short := partsOf(black.Null(at, frame, vec3(aim)).Momentum())
	long := partsOf(black.Null(at, frame, vec3(longer)).Momentum())
	for part := range short {
		if !within(short[part], long[part], numbers.Frame.Tolerance) {
			t.Errorf("place %d: %v for the aim, %v for a longer one",
				part, short[part], long[part])
		}
	}
}
