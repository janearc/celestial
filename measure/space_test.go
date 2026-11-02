package measure_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/measure"
)

// three is three numbers, one on each axis.
type three struct {
	OnX float64 `json:"x"`
	OnY float64 `json:"y"`
	OnZ float64 `json:"z"`
}

// span is a range numbers are drawn from.
type span struct{ Least, Most float64 }

// spaced is the numbers space's tests use, from testdata/space.json.
type spaced struct {
	ThreeFour three   `json:"three four"`
	East      three   `json:"east"`
	Five      float64 `json:"five"`
	Along     float64 `json:"along"`
	Here      three   `json:"here"`
	There     three   `json:"there"`
	NoWays    []three `json:"no ways"`
	Beyond    []three `json:"beyond squaring"`
	Every     struct {
		By      three   `json:"by m"`
		Other   three   `json:"other m"`
		Dot     float64 `json:"dot m2"`
		Square  three   `json:"square m"`
		Moving  three   `json:"moving km/s"`
		North   three   `json:"north"`
		Squared float64 `json:"sweep squared"`
		Plus    three   `json:"plus m"`
	} `json:"every part"`
	TooFast     []three `json:"too fast km/s"`
	EarthOut    three   `json:"earth out m"`
	EarthMoving three   `json:"earth moving km/s"`
	EarthSpeed  float64 `json:"earth speed km/s"`
	Light       float64 `json:"light km/s"`
	Tolerance   float64 `json:"tolerance"`
	SweepTol    float64 `json:"sweep tolerance"`
	Lengths     span    `json:"lengths m"`
	Speeds      span    `json:"speeds km/s"`
}

// spaceNumbers reads the test numbers, or the test stops.
func spaceNumbers(test testing.TB) spaced {
	test.Helper()
	var numbers spaced
	raw, err := os.ReadFile("testdata/space.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// direction is the way three numbers point, or the test stops.
func direction(test testing.TB, parts three) measure.Direction {
	test.Helper()
	way, err := measure.NewDirection(parts.OnX, parts.OnY, parts.OnZ)
	if err != nil {
		test.Fatal(err)
	}
	return *way
}

// place is the place three positions in metres make.
func place(parts three) measure.Coord {
	return measure.At(measure.Metres(parts.OnX), measure.Metres(parts.OnY),
		measure.Metres(parts.OnZ))
}

// A direction is one long, and three and four go 0.6 of the way along
// the first.
func TestADirectionIsOneLong(test *testing.T) {
	numbers := spaceNumbers(test)
	way := direction(test, numbers.ThreeFour)
	one := pickedNumbers(test).One
	if self := way.Dot(way); !near(self, one, numbers.Tolerance) {
		test.Errorf("a direction is %v long, not one", self)
	}
	along := way.Dot(direction(test, numbers.East))
	if !near(along, numbers.Along, numbers.Tolerance) {
		test.Errorf("three and four is %v along the first", along)
	}
}

// Parts too large or too small to square still point somewhere: their
// direction is one long, and the way the same parts point at an everyday
// size, scaled down by their largest.
func TestPartsBeyondSquaringStillPoint(test *testing.T) {
	numbers := spaceNumbers(test)
	one := pickedNumbers(test).One
	for _, parts := range numbers.Beyond {
		way := direction(test, parts)
		if self := way.Dot(way); !near(self, one, numbers.Tolerance) {
			test.Errorf("a direction of %v is %v long", parts, self)
		}
		largest := math.Max(math.Abs(parts.OnX), math.Abs(parts.OnY))
		largest = math.Max(largest, math.Abs(parts.OnZ))
		everyday := direction(test, three{OnX: parts.OnX / largest,
			OnY: parts.OnY / largest, OnZ: parts.OnZ / largest})
		if along := way.Dot(everyday); !near(along, one,
			numbers.Tolerance) {
			test.Errorf("a direction of %v is %v along its own way",
				parts, along)
		}
	}
}

// The anti-test: a direction refuses no way at all, a part that is not
// a number and one that is endless, and gives no direction.
func TestADirectionRefusesWhatHasNoWay(test *testing.T) {
	numbers := spaceNumbers(test)
	given := append(numbers.NoWays, three{OnX: math.NaN()},
		three{OnX: math.Inf(upward)})
	for _, parts := range given {
		way, err := measure.NewDirection(parts.OnX, parts.OnY, parts.OnZ)
		if err == nil || way != nil {
			test.Errorf("a direction of %v was made", parts)
		}
	}
}

// Places subtract to displacements, and a place plus the way to another
// is the other.
func TestPlacesSubtractToDisplacements(test *testing.T) {
	numbers := spaceNumbers(test)
	here, there := place(numbers.Here), place(numbers.There)
	apart := there.Minus(here)
	if metres := apart.Length().InMetres(); !near(metres, numbers.Five,
		numbers.Tolerance) {
		test.Errorf("three and four apart is %v m, not five", metres)
	}
	back := here.Plus(apart).Minus(there).Length().InMetres()
	if back > numbers.Tolerance {
		test.Errorf("there and back misses by %v m", back)
	}
}

// The anti-test: a displacement that goes nowhere has no direction.
func TestADisplacementOfNothingHasNoDirection(test *testing.T) {
	still := measure.Origin().Minus(measure.Origin())
	if way, err := still.Direction(); err == nil || way != nil {
		test.Errorf("going nowhere points %v", way)
	}
}

// stopper is a test that can stop: a testing.T, or rapid's.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// velocity is a velocity in km/s, or the test stops.
func velocity(test stopper, parts three) measure.Velocity {
	test.Helper()
	moving, err := measure.KilometresPerSecond(parts.OnX, parts.OnY, parts.OnZ)
	if err != nil {
		test.Fatal(err)
	}
	return *moving
}

// Three and four km/s make five, whichever way.
func TestAVelocitysSpeedIsItsLength(test *testing.T) {
	numbers := spaceNumbers(test)
	speed := velocity(test, numbers.ThreeFour).Speed()
	if got := speed.InKilometresPerSecond(); !near(got, numbers.Five,
		numbers.Tolerance) {
		test.Errorf("three and four km/s is %v km/s, not five", got)
	}
	share := speed.InLightFraction()
	if !near(share, numbers.Five/numbers.Light, numbers.Tolerance) {
		test.Errorf("five km/s is %v of light", share)
	}
}

// The anti-test: a velocity refuses light's speed and past it, and a
// part that is not a number, and gives no velocity.
func TestAVelocityRefusesLightAndPast(test *testing.T) {
	numbers := spaceNumbers(test)
	given := append(numbers.TooFast, three{OnX: math.NaN()})
	for _, parts := range given {
		moving, err := measure.KilometresPerSecond(parts.OnX, parts.OnY, parts.OnZ)
		if err == nil || moving != nil {
			test.Errorf("a velocity of %v km/s was made", parts)
		}
	}
}

// The Earth, an AU out along x and moving toward minus z, goes round
// counterclockwise seen from above, so its sweep's north is the kit's up; and
// the sweep is the distance times the speed, as a share of light's.
func TestTheEarthSweepsRoundTheKitsUp(test *testing.T) {
	numbers := spaceNumbers(test)
	out := numbers.EarthOut
	apart := measure.Offset(measure.Metres(out.OnX), measure.Metres(out.OnY),
		measure.Metres(out.OnZ))
	sweep := apart.Cross(velocity(test, numbers.EarthMoving))
	north, err := sweep.North()
	if err != nil {
		test.Fatal(err)
	}
	one := pickedNumbers(test).One
	if up := north.Dot(measure.Up()); !near(up, one, numbers.Tolerance) {
		test.Errorf("the earth's north is %v of the way up", up)
	}
	size := sweep.Size().InMetres()
	want := out.OnX * numbers.EarthSpeed / numbers.Light
	if !near(size, want, numbers.SweepTol) {
		test.Errorf("the earth sweeps %v m, not %v", size, want)
	}
}

// The anti-test: what moves straight away sweeps no plane, so its sweep
// has no north.
func TestStraightAwayHasNoNorth(test *testing.T) {
	numbers := spaceNumbers(test)
	out := numbers.EarthOut
	apart := measure.Offset(measure.Metres(out.OnX), measure.Metres(out.OnY),
		measure.Metres(out.OnZ))
	away := velocity(test, three{OnX: numbers.EarthSpeed})
	if north, err := apart.Cross(away).North(); err == nil || north != nil {
		test.Errorf("moving straight away faces %v", north)
	}
}

// offset is the displacement of three lengths in metres.
func offset(parts three) measure.Displacement {
	return measure.Offset(measure.Metres(parts.OnX),
		measure.Metres(parts.OnY), measure.Metres(parts.OnZ))
}

// A displacement against another is each of its parts times the other's,
// summed: one, two and three metres against four, five and six is 32
// square metres, and with no part nought or one, every term shows.
func TestADotIsEveryPartTimesTheOthers(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	area := offset(every.By).Dot(offset(every.Other)).InSquareMetres()
	if !near(area, every.Dot, numbers.Tolerance) {
		test.Errorf("%v against %v is %v m^2, not %v", every.By,
			every.Other, area, every.Dot)
	}
}

// The anti-test: two displacements at a right angle have no area against
// each other, though neither is nought. An area that is no number fails
// it.
func TestARightAngleHasNoDot(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	area := offset(every.By).Dot(offset(every.Square)).InSquareMetres()
	if !(math.Abs(area) <= numbers.Tolerance) {
		test.Errorf("%v against %v is %v m^2", every.By, every.Square,
			area)
	}
}

// A displacement crossed with a velocity faces the way the right hand
// says and sweeps the size of their cross: one, two and three metres by
// four, five and six km/s faces minus one, two and minus one, and sweeps
// the root of 54 m km/s, as a share of light's.
func TestACrossIsEveryPartTimesTheOthers(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	sweep := offset(every.By).Cross(velocity(test, every.Moving))
	north, err := sweep.North()
	if err != nil {
		test.Fatal(err)
	}
	one := pickedNumbers(test).One
	along := north.Dot(direction(test, every.North))
	if !near(along, one, numbers.Tolerance) {
		test.Errorf("the sweep faces %v of the way north", along)
	}
	size := sweep.Size().InMetres()
	want := math.Sqrt(every.Squared) / numbers.Light
	if !near(size, want, numbers.SweepTol) {
		test.Errorf("the sweep is %v m, not %v", size, want)
	}
}

// The anti-test: crossed the other way round, four, five and six metres
// by one, two and three km/s, the sweep faces the other way.
func TestACrossTheOtherWayRoundFacesTheOtherWay(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	sweep := offset(every.Other).Cross(velocity(test, every.By))
	north, err := sweep.North()
	if err != nil {
		test.Fatal(err)
	}
	one := pickedNumbers(test).One
	along := north.Dot(direction(test, every.North))
	if !near(along, -one, numbers.Tolerance) {
		test.Errorf("the sweep faces %v of the way north", along)
	}
}

// A place plus a displacement moves along every part of it: one, two and
// three metres, and four, five and six more, is five, seven and nine.
func TestAPlacePlusADisplacementMovesAlongEveryPart(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	to := place(every.By).Plus(offset(every.Other)).Minus(measure.Origin())
	for _, part := range []struct{ got, want float64 }{
		{to.AlongX().InMetres(), every.Plus.OnX},
		{to.AlongY().InMetres(), every.Plus.OnY},
		{to.AlongZ().InMetres(), every.Plus.OnZ}} {
		if !near(part.got, part.want, numbers.Tolerance) {
			test.Errorf("%v plus %v is %v", every.By, every.Other, to)
		}
	}
}

// The anti-test: a place plus a displacement moves the whole of it, along
// it: not nowhere, and not against it. The way from where the place was
// to where it is, against the displacement, is the displacement's own
// square.
func TestAPlaceMovesAlongItsDisplacement(test *testing.T) {
	numbers := spaceNumbers(test)
	every := numbers.Every
	by := offset(every.Other)
	moved := place(every.By).Plus(by).Minus(place(every.By))
	along := moved.Dot(by).InSquareMetres()
	if want := by.Dot(by).InSquareMetres(); !near(along, want,
		numbers.Tolerance) {
		test.Errorf("a place moved %v m^2 along %v, not %v", along,
			every.Other, want)
	}
}

// Whatever the lengths and speeds, adding a displacement or a velocity
// and taking it away again leaves what was there, a direction times a
// length goes that length, and the parts are what went in.
func TestAddingAndTakingAwayUndoEachOther(test *testing.T) {
	rapid.Check(test, undoing{numbers: spaceNumbers(test)}.check)
}

// undoing draws displacements and velocities and undoes their sums.
type undoing struct{ numbers spaced }

// draw is a number from a span.
func draw(test *rapid.T, within span, name string) float64 {
	return rapid.Float64Range(within.Least, within.Most).Draw(test, name)
}

// check draws two of each and checks they undo.
func (undo undoing) check(test *rapid.T) {
	lengths, speeds := undo.numbers.Lengths, undo.numbers.Speeds
	tolerance := undo.numbers.SweepTol
	parts := three{OnX: draw(test, lengths, "x"),
		OnY: draw(test, lengths, "y"), OnZ: draw(test, lengths, "z")}
	by := measure.Offset(measure.Metres(parts.OnX), measure.Metres(parts.OnY),
		measure.Metres(parts.OnZ))
	got := three{OnX: by.AlongX().InMetres(), OnY: by.AlongY().InMetres(),
		OnZ: by.AlongZ().InMetres()}
	if !near(got.OnX, parts.OnX, tolerance) ||
		!near(got.OnY, parts.OnY, tolerance) ||
		!near(got.OnZ, parts.OnZ, tolerance) {
		test.Fatalf("an offset of %v has parts %v", parts, got)
	}
	scale := draw(test, span{Least: lengths.Least / lengths.Most,
		Most: lengths.Most / lengths.Most}, "share")
	back := by.Plus(by.Times(scale)).Minus(by.Times(scale))
	if !near(back.Length().InMetres(), by.Length().InMetres(), tolerance) {
		test.Fatalf("%v and back is %v", parts, back.Length().InMetres())
	}
	way, err := by.Direction()
	if err != nil {
		test.Fatal(err)
	}
	gone := way.Times(by.Length())
	if !near(gone.Dot(by).InSquareMetres(), by.Dot(by).InSquareMetres(),
		tolerance) {
		test.Fatalf("%v's own way goes %v", parts, gone.Length().InMetres())
	}
	moving := velocity(test, three{OnX: draw(test, speeds, "vx"),
		OnY: draw(test, speeds, "vy"), OnZ: draw(test, speeds, "vz")})
	speed := moving.Speed().InLightFraction()
	if !near(moving.Dot(moving), speed*speed, tolerance) {
		test.Fatalf("a velocity against itself is %v, its speed %v",
			moving.Dot(moving), speed)
	}
	other := moving.Times(draw(test, span{Least: speeds.Least / speeds.Most,
		Most: speeds.Most / speeds.Most}, "share"))
	again := moving.Plus(other).Minus(other)
	if !near(again.Speed().InLightFraction(),
		moving.Speed().InLightFraction(), tolerance) {
		test.Fatalf("%v and back is %v", moving.Speed(), again.Speed())
	}
	parted := moving.AlongX().InLightFraction() +
		moving.AlongY().InLightFraction() + moving.AlongZ().InLightFraction()
	if math.IsNaN(parted) {
		test.Fatalf("%v has parts that are not numbers", moving.Speed())
	}
}
