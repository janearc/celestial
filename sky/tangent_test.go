package sky_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// span is a range numbers are drawn from.
type span struct {
	Least float64 `json:"least"`
	Most  float64 `json:"most"`
}

// axes is an ellipse's two semi-axes and position angle, in arcseconds
// and degrees.
type axes struct {
	Major float64 `json:"major"`
	Minor float64 `json:"minor"`
	Angle float64 `json:"angle"`
}

// planed is the numbers the tangent plane's tests use, from
// testdata/tangent.json.
type planed struct {
	TanOneDegree float64   `json:"tan one degree"`
	OneDegree    float64   `json:"one degree"`
	Quarter      float64   `json:"quarter turn degrees"`
	Behind       []float64 `json:"behind degrees"`
	Ascensions   span      `json:"ascensions"`
	Declinations span      `json:"declinations"`
	Field        span      `json:"field degrees"`
	RoundTrip    float64   `json:"round trip arcseconds"`
	Shortcut     struct {
		Declination float64 `json:"declination degrees"`
		Arcminute   float64 `json:"one arcminute degrees"`
		Within      float64 `json:"within arcseconds"`
		Beyond      float64 `json:"beyond arcseconds"`
	} `json:"shortcut"`
	Ellipse struct {
		Major  float64 `json:"major arcseconds"`
		Minor  float64 `json:"minor arcseconds"`
		Points int     `json:"points"`
	} `json:"ellipse"`
	NotEllipses []axes  `json:"not ellipses arcseconds"`
	TooFew      []int   `json:"too few points"`
	Tolerance   float64 `json:"tolerance"`
}

// planeNumbers reads the test numbers, or the test stops.
func planeNumbers(test *testing.T) planed {
	test.Helper()
	var numbers planed
	raw, err := os.ReadFile("testdata/tangent.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// stopper is a test that can stop: a testing.T, or rapid's.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// toward is the direction at an ascension and declination in degrees,
// or the test stops.
func toward(test stopper, east, north float64) sky.Equatorial {
	test.Helper()
	direction, err := sky.NewEquatorial(measure.Degrees(east),
		measure.Degrees(north))
	if err != nil {
		test.Fatal(err)
	}
	return *direction
}

// landed is where a direction lands on a plane, or the test stops.
func landed(test stopper, plane sky.Tangent, at sky.Equatorial) sky.Offset {
	test.Helper()
	offset, err := plane.Onto(at)
	if err != nil {
		test.Fatal(err)
	}
	return *offset
}

// zero is no angle at all. Defined by the scale.
const nought = 0.0

// A plane's centre lands on it at nothing east and nothing north.
func TestTheCentreLandsAtNothing(test *testing.T) {
	centre := toward(test, planeNumbers(test).Quarter, planeNumbers(test).
		OneDegree)
	offset := landed(test, sky.NewTangent(centre), centre)
	if offset.East().InRadians() != nought ||
		offset.North().InRadians() != nought {
		test.Errorf("the centre lands %v east, %v north",
			offset.East().InArcseconds(), offset.North().InArcseconds())
	}
}

// A degree along the equator, or up from it, from a centre on it lands
// the tangent of a degree away, as the projection's definition has it.
func TestADegreeAwayLandsItsTangentAway(test *testing.T) {
	numbers := planeNumbers(test)
	plane := sky.NewTangent(toward(test, nought, nought))
	east := landed(test, plane, toward(test, numbers.OneDegree, nought))
	north := landed(test, plane, toward(test, nought, numbers.OneDegree))
	if got := east.East().InRadians(); !(math.Abs(got-numbers.TanOneDegree) <=
		numbers.Tolerance) ||
		east.North().InRadians() != nought {
		test.Errorf("a degree east lands %v east, %v north", got,
			east.North().InRadians())
	}
	if got := north.North().InRadians(); !(math.Abs(got-
		numbers.TanOneDegree) <=
		numbers.Tolerance) {
		test.Errorf("a degree north lands %v north", got)
	}
}

// The anti-test: a direction a quarter turn or more from the centre is
// behind the plane, refused, and gives no offset.
func TestWhatIsBehindThePlaneIsRefused(test *testing.T) {
	numbers := planeNumbers(test)
	plane := sky.NewTangent(toward(test, nought, nought))
	for _, away := range numbers.Behind {
		at := toward(test, away, nought)
		offset, err := plane.Onto(at)
		if err == nil || offset != nil {
			test.Errorf("%v landed at %v, %v", at, offset, err)
		}
	}
}

// Wherever the centre, a place in a field a degree across goes back to
// a direction that lands where it was, and that direction is the one
// the place came from: Back undoes Onto, and Onto undoes Back.
func TestBackUndoesOnto(test *testing.T) {
	rapid.Check(test, trip{numbers: planeNumbers(test)}.check)
}

// trip draws a centre and a place near it.
type trip struct{ numbers planed }

// draw is a number from a span.
func draw(test *rapid.T, within span, name string) float64 {
	return rapid.Float64Range(within.Least, within.Most).Draw(test, name)
}

// check draws a centre and an offset, and goes there and back.
func (there trip) check(test *rapid.T) {
	numbers := there.numbers
	centre := toward(test, draw(test, numbers.Ascensions, "ascension"),
		draw(test, numbers.Declinations, "declination"))
	plane := sky.NewTangent(centre)
	east := draw(test, numbers.Field, "east")
	north := draw(test, numbers.Field, "north")
	offset := sky.NewOffset(measure.Degrees(east), measure.Degrees(north))
	direction := plane.Back(offset)
	again := landed(test, plane, direction)
	missed := math.Hypot(again.East().InArcseconds()-
		offset.East().InArcseconds(), again.North().InArcseconds()-
		offset.North().InArcseconds())
	if missed > numbers.RoundTrip {
		test.Fatalf("%v there and back misses by %v arcseconds", offset,
			missed)
	}
	if north := direction.Declination().InDegrees(); !(math.Abs(north) <=
		numbers.Quarter) {
		test.Fatalf("%v went back to declination %v", offset, north)
	}
}

// shortcutMiss is how far the small-angle shortcut puts a place a field
// east of a centre at a declination, from where the gnomonic plane does,
// in arcseconds: the shortcut takes east as the ascension apart times the
// cosine of the centre's declination, and north as the declination apart.
func shortcutMiss(test *testing.T, declination, east float64) float64 {
	test.Helper()
	centre := toward(test, nought, declination)
	plane := sky.NewTangent(centre)
	out := sky.NewOffset(measure.Degrees(east), measure.Radians(nought))
	at := plane.Back(out)
	cosine := math.Cos(measure.Degrees(declination).InRadians())
	shortEast := at.Ascension().InArcseconds() * cosine
	shortNorth := at.Declination().InArcseconds() -
		centre.Declination().InArcseconds()
	exact := landed(test, plane, at)
	return math.Hypot(shortEast-exact.East().InArcseconds(),
		shortNorth-exact.North().InArcseconds())
}

// The small-angle shortcut, which a quick drawing of the sky reaches
// for, is within a tenth of an arcsecond of the gnomonic plane an
// arcminute from a centre at sixty degrees north.
func TestTheShortcutHoldsInASmallField(test *testing.T) {
	shortcut := planeNumbers(test).Shortcut
	miss := shortcutMiss(test, shortcut.Declination, shortcut.Arcminute)
	test.Logf("an arcminute out, the shortcut misses by %.3g arcseconds",
		miss)
	if miss > shortcut.Within {
		test.Errorf("an arcminute out, the shortcut misses by %v "+
			"arcseconds", miss)
	}
}

// The anti-test: a degree from the same centre the shortcut misses by
// more than ten arcseconds, so a field that wide needs the plane.
func TestTheShortcutFailsInAWideField(test *testing.T) {
	numbers := planeNumbers(test)
	miss := shortcutMiss(test, numbers.Shortcut.Declination,
		numbers.OneDegree)
	test.Logf("a degree out, the shortcut misses by %.3g arcseconds", miss)
	if miss < numbers.Shortcut.Beyond {
		test.Errorf("a degree out, the shortcut misses by only %v "+
			"arcseconds", miss)
	}
}
