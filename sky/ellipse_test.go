package sky_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// ellipse is the ellipse of two semi-axes in arcseconds and a position
// angle in degrees, or the test stops.
func ellipse(test *testing.T, major, minor, angle float64) sky.Ellipse {
	test.Helper()
	made, err := sky.NewEllipse(measure.Arcseconds(major),
		measure.Arcseconds(minor), measure.Degrees(angle))
	if err != nil {
		test.Fatal(err)
	}
	return *made
}

// outline is so many points round an ellipse, or the test stops.
func outline(test *testing.T, around sky.Ellipse, points int) []sky.Offset {
	test.Helper()
	offsets, err := around.Outline(points)
	if err != nil {
		test.Fatal(err)
	}
	return offsets
}

// lands is whether an offset is so far east and north, in arcseconds.
func lands(offset sky.Offset, east, north, tolerance float64) bool {
	return math.Abs(offset.East().InArcseconds()-east) <= tolerance &&
		math.Abs(offset.North().InArcseconds()-north) <= tolerance
}

// An ellipse at a position angle of nothing has its major axis to the
// north and its minor to the east; at a quarter turn, the major is east.
// The outline starts at the end of the major axis the angle points to.
func TestTheMajorAxisLiesAlongThePositionAngle(test *testing.T) {
	numbers := planeNumbers(test)
	shape := numbers.Ellipse
	quarter := shape.Points / quadrants
	north := outline(test, ellipse(test, shape.Major, shape.Minor, nought),
		shape.Points)
	if !lands(north[firstPoint], nought, shape.Major, numbers.Tolerance) ||
		!lands(north[quarter], shape.Minor, nought, numbers.Tolerance) {
		test.Errorf("at nothing, the outline starts %v and turns to %v",
			north[firstPoint], north[quarter])
	}
	east := outline(test, ellipse(test, shape.Major, shape.Minor,
		numbers.Quarter), shape.Points)
	if !lands(east[firstPoint], shape.Major, nought, numbers.Tolerance) {
		test.Errorf("at a quarter turn, the outline starts %v",
			east[firstPoint])
	}
}

// Ways round an outline, each defined: it starts at the first point, and
// a quarter of the way round is a quadrant of four.
const (
	firstPoint = 0 // defined by a slice's order
	quadrants  = 4 // defined by a turn: four quarters
)

// Every point of an outline is on its ellipse: its parts along the two
// axes, each over its axis, square and add to one.
func TestEveryPointIsOnTheEllipse(test *testing.T) {
	numbers := planeNumbers(test)
	shape := numbers.Ellipse
	angle := measure.Degrees(numbers.OneDegree).InRadians()
	sine, cosine := math.Sincos(angle)
	around := ellipse(test, shape.Major, shape.Minor, numbers.OneDegree)
	for _, point := range outline(test, around, shape.Points) {
		east, north := point.East().InArcseconds(),
			point.North().InArcseconds()
		alongMajor := (east*sine + north*cosine) / shape.Major
		alongMinor := (east*cosine - north*sine) / shape.Minor
		if sum := alongMajor*alongMajor + alongMinor*alongMinor; !(math.Abs(
			sum-numbers.OneDegree) <=
			numbers.RoundTrip) {
			test.Errorf("%v is %v of the way to the ellipse", point, sum)
		}
	}
}

// The anti-test: a minor axis longer than the major, an axis below
// nothing, past the far side of the sky or not a number, and an angle
// more than a turn from north or not a number, are refused and give no
// ellipse.
func TestWhatIsNoEllipseIsRefused(test *testing.T) {
	given := planeNumbers(test).NotEllipses
	major := planeNumbers(test).Ellipse.Major
	given = append(given, axes{Major: math.NaN()},
		axes{Major: math.Inf(upward)}, axes{Major: major, Angle: math.NaN()},
		axes{Major: major, Angle: math.Inf(upward)})
	for _, shape := range given {
		made, err := sky.NewEllipse(measure.Arcseconds(shape.Major),
			measure.Arcseconds(shape.Minor), measure.Degrees(shape.Angle))
		if err == nil || made != nil {
			test.Errorf("%+v made %v, %v", shape, made, err)
		}
	}
}

// upward asks math.Inf for the endless above. Defined by math.Inf.
const upward = 1

// The anti-test: an outline of fewer than three points closes round no
// area, and is refused.
func TestTooFewPointsAreRefused(test *testing.T) {
	shape := planeNumbers(test).Ellipse
	around := ellipse(test, shape.Major, shape.Minor, nought)
	for _, points := range planeNumbers(test).TooFew {
		offsets, err := around.Outline(points)
		if err == nil || offsets != nil {
			test.Errorf("%d points gave %v, %v", points, offsets, err)
		}
	}
}

// The anti-test: an outline has as many points as were asked for, and does
// not close on itself by repeating its first point at its end.
func TestAnOutlineDoesNotRepeatItsFirstPoint(test *testing.T) {
	numbers := planeNumbers(test)
	shape := numbers.Ellipse
	around := ellipse(test, shape.Major, shape.Minor, nought)
	for _, points := range []int{fewestToClose, shape.Points} {
		offsets := outline(test, around, points)
		last := offsets[len(offsets)-atEnd]
		if len(offsets) != points || lands(last,
			offsets[firstPoint].East().InArcseconds(),
			offsets[firstPoint].North().InArcseconds(), numbers.Tolerance) {
			test.Errorf("%d points gave %d, the last at %v", points,
				len(offsets), last)
		}
	}
}

// fewestToClose is the fewest points that close round an area. Defined by
// geometry: a triangle.
const fewestToClose = 3

// atEnd is how far back from a slice's length its last item is. Defined
// by a slice's order.
const atEnd = 1
