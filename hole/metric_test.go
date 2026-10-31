package hole_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/hole"
)

// four is the four parts of a Vec4 as the data gives them.
type four struct {
	Time   float64
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// vec4 is four numbers as a hole's Vec4.
func vec4(parts four) hole.Vec4 {
	return hole.NewVec4(
		parts.Time, parts.AlongX, parts.AlongY, parts.AlongZ,
	)
}

// drawnVec4 draws a Vec4 from the range the data picks for coordinates.
func drawnVec4(test *rapid.T, draws *picked, name string) hole.Vec4 {
	part := rapid.Float64Range(
		draws.Draws.CoordinateLow, draws.Draws.CoordinateHigh,
	)
	return hole.NewVec4(
		part.Draw(test, name+" time"), part.Draw(test, name+" x"),
		part.Draw(test, name+" y"), part.Draw(test, name+" z"),
	)
}

// partsOf are a Vec4's four parts in order.
func partsOf(vector hole.Vec4) [dimensions]float64 {
	return [dimensions]float64{
		vector.Time(), vector.AlongX(), vector.AlongY(), vector.AlongZ(),
	}
}

// The metric times its inverse is the identity, with and without spin.
func TestTheMetricTimesItsInverseIsTheIdentity(t *testing.T) {
	numbers := pickedNumbers(t)
	for _, from := range []spec{
		numbers.Holes.Still, numbers.Holes.Spinning, numbers.Holes.Heavy,
	} {
		black := build(t, from)
		for _, at := range []place{
			numbers.Places.Skew, numbers.Places.Slant, numbers.Places.Tilt,
		} {
			position := vec3(at)
			metric, inverse := black.Metric(position), black.Inverse(position)
			for row := range dimensions {
				for column := range dimensions {
					sum := float64(none)
					for between := range dimensions {
						sum += metric[row][between] * inverse[between][column]
					}
					want := float64(none)
					if row == column {
						want = identity
					}
					if !within(sum, want, numbers.Metric.IdentityTolerance) {
						t.Fatalf("at %v the product has %v at %d, %d",
							at, sum, row, column)
					}
				}
			}
		}
	}
}

// The anti-test: a metric that was not its inverse's inverse would be caught by this: the
// metric is not its own inverse near the hole, so the test above could not
// pass by a metric and an inverse that are the same matrix.
func TestTheMetricIsNotItsOwnInverseNearTheHole(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	position := vec3(numbers.Places.Skew)
	metric, inverse := black.Metric(position), black.Inverse(position)
	apart := float64(none)
	for row := range dimensions {
		for column := range dimensions {
			apart = math.Max(apart, math.Abs(metric[row][column]-
				inverse[row][column]))
		}
	}
	if !(apart > numbers.Metric.IdentityTolerance) {
		t.Errorf("the metric and its inverse differ by only %v", apart)
	}
}

// The metric is the flat one far from the hole: minus one along time, one
// along each axis of space, and nothing between.
func TestTheMetricIsFlatFarAway(t *testing.T) {
	numbers := pickedNumbers(t)
	far := vec3(numbers.Places.Far)
	black := build(t, numbers.Holes.Spinning)
	for name, matrix := range map[string]hole.Mat4{
		"metric": black.Metric(far), "inverse": black.Inverse(far),
	} {
		for row := range dimensions {
			for column := range dimensions {
				if !within(matrix[row][column], flatEntry(row, column),
					numbers.Metric.FarTolerance) {
					t.Errorf("far away the %s is %v at %d, %d", name,
						matrix[row][column], row, column)
				}
			}
		}
	}
}

// flatEntry is the flat metric's entry, which is also the flat inverse's:
// minus one on the diagonal along time, one on the others, nought off it.
func flatEntry(row, column int) float64 {
	switch {
	case row != column:
		return none
	case row == timeRow:
		return timeSign
	default:
		return spaceSign
	}
}

// The anti-test: near the hole the metric is not the flat one, and the difference is the
// hole's pull.
func TestTheMetricIsNotFlatNearTheHole(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	metric := black.Metric(vec3(numbers.Places.Near))
	if within(metric[timeRow][timeRow], flatEntry(timeRow, timeRow),
		numbers.Metric.FarTolerance) {
		t.Errorf("at %v the time entry is flat: %v", numbers.Places.Near,
			metric[timeRow][timeRow])
	}
}

// The test and the anti-test: on the disc inside the ring the radius is nought and the kernel has no
// value: the metric is flat there, as Metric says, and a step from it into
// the open is not.
func TestTheMetricOnTheDiscIsFlat(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Spinning)
	for name, matrix := range map[string]hole.Mat4{
		"metric":  black.Metric(vec3(numbers.Places.Origin)),
		"inverse": black.Inverse(vec3(numbers.Places.Origin)),
	} {
		for row := range dimensions {
			for column := range dimensions {
				if matrix[row][column] != flatEntry(row, column) {
					t.Errorf("on the disc the %s is %v at %d, %d", name,
						matrix[row][column], row, column)
				}
			}
		}
	}
	away := black.Metric(vec3(numbers.Places.Equator))
	if away[timeRow][timeRow] == flatEntry(timeRow, timeRow) {
		t.Errorf("off the disc the metric is flat")
	}
}

// Lower is the metric on a vector and Raise is the inverse on a covector,
// whatever the hole, the point and the vector.
func TestLowerAndRaiseAreTheMetricAndItsInverse(t *testing.T) {
	rapid.Check(t, lowerAndRaiseAreTheMetric)
}

// lowerAndRaiseAreTheMetric is the property that test checks.
func lowerAndRaiseAreTheMetric(test *rapid.T) {
	draws := pickedNumbers(test)
	black, position := drawnOutside(test, draws)
	vector := drawnVec4(test, draws, "vector")
	lowered := partsOf(black.Lower(position, vector))
	raised := partsOf(black.Raise(position, vector))
	metric, inverse := black.Metric(position), black.Inverse(position)
	given := partsOf(vector)
	for row := range dimensions {
		byMetric, byInverse := float64(none), float64(none)
		for column := range dimensions {
			byMetric += metric[row][column] * given[column]
			byInverse += inverse[row][column] * given[column]
		}
		if !within(lowered[row], byMetric, draws.Metric.RoundTrip) ||
			!within(raised[row], byInverse, draws.Metric.RoundTrip) {
			test.Fatalf("row %d: lowered %v want %v, raised %v want %v",
				row, lowered[row], byMetric, raised[row], byInverse)
		}
	}
}

// drawnOutside draws a hole and a point outside its horizon, where the
// metric is asked for.
func drawnOutside(test *rapid.T, draws *picked) (*hole.Hole, hole.Vec3) {
	black := drawnHole(test, draws)
	position := drawnPlace(test, draws, "position")
	if black.Radius(position) < black.Horizon() {
		test.Skip("inside the horizon the metric is not asked for")
	}
	return black, position
}

// Raising what was lowered brings a vector home.
func TestRaiseUndoesLower(t *testing.T) {
	rapid.Check(t, raiseUndoesLower)
}

// raiseUndoesLower is the property that test checks.
func raiseUndoesLower(test *rapid.T) {
	draws := pickedNumbers(test)
	black, position := drawnOutside(test, draws)
	vector := drawnVec4(test, draws, "vector")
	lowered := black.Lower(position, vector)
	home := partsOf(black.Raise(position, lowered))
	for place, part := range partsOf(vector) {
		if !within(home[place], part, draws.Metric.RoundTrip) {
			test.Fatalf("place %d came home as %v, not %v",
				place, home[place], part)
		}
	}
}

// The test and the anti-test: near a hole lowering is not the flat metric's minus-plus-plus-plus: a
// Lower that only flipped the sign of time would be caught here.
func TestLowerIsNotFlatNearTheHole(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	position := vec3(numbers.Places.Near)
	vector := vec4(numbers.Dot.Vector)
	lowered := black.Lower(position, vector)
	flat := -vector.Time()
	if within(lowered.Time(), flat, numbers.Metric.FarTolerance) {
		t.Errorf("lowering near the hole only turned time about: %v",
			lowered.Time())
	}
	far := black.Lower(vec3(numbers.Places.Far), vector)
	allowed := numbers.Metric.FarTolerance * math.Abs(vector.Time())
	if !within(far.Time(), flat, allowed) {
		t.Errorf("far away lowering is %v, not the flat %v", far.Time(), flat)
	}
}

// The radius is the distance from the centre round a hole with no spin, and
// round a spinning one it is the oblate radius: on the axis the same
// distance, and in the equator less than it by the spin squared.
func TestRadiusIsTheKerrSchildRadius(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	spinning := build(t, numbers.Holes.Spinning)
	equator, pole := numbers.Places.Equator, numbers.Places.Pole
	if got := still.Radius(vec3(equator)); !within(got, equator.AlongX,
		numbers.Metric.IdentityTolerance) {
		t.Errorf("a still hole's radius at %v is %v", equator, got)
	}
	if got := spinning.Radius(vec3(pole)); !within(got, pole.AlongZ,
		numbers.Metric.IdentityTolerance) {
		t.Errorf("on the axis the radius is %v, not %v", got, pole.AlongZ)
	}
	spin := spinning.Spin()
	want := math.Sqrt(equator.AlongX*equator.AlongX - spin*spin)
	if got := spinning.Radius(vec3(equator)); !within(got, want,
		numbers.Metric.IdentityTolerance) {
		t.Errorf("in the equator the radius is %v, not %v", got, want)
	}
	if spinning.Radius(vec3(equator)) == still.Radius(vec3(equator)) {
		t.Errorf("spin did not change the radius in the equator")
	}
}

// The test and the anti-test: a point within a hair of the disc has radius nought, not the root of a
// negative that rounding left behind: rapid found one, and the radius
// there was not a number. A point off the disc has a radius that is.
func TestRadiusNearTheDiscRoundsToNought(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Hair.Hole)
	if got := black.Radius(vec3(numbers.Hair.Place)); got != none {
		t.Errorf("a hair from the disc the radius is %v", got)
	}
	if got := black.Radius(vec3(numbers.Places.Eye)); !(got > none) {
		t.Errorf("off the disc the radius is %v", got)
	}
}

// Dot is the sum of the parts multiplied, with no sign on time: the metric
// is the hole's to apply, through Lower.
func TestDotIsTheSumOfTheProducts(t *testing.T) {
	dot := pickedNumbers(t).Dot
	got := hole.Dot(vec4(dot.Covector), vec4(dot.Vector))
	if got != dot.Want {
		t.Errorf("the dot of the vectors is %v, want %v", got, dot.Want)
	}
}

// A covector with only a time part is square to a vector with none.
func TestDotOfSquareVectorsIsNought(t *testing.T) {
	dot := pickedNumbers(t).Dot
	got := hole.Dot(vec4(dot.TimeOnly), vec4(dot.SpaceOnly))
	if got != none {
		t.Errorf("the dot of square vectors is %v", got)
	}
}
