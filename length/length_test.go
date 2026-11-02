package length_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/si"
)

// picked is the numbers these tests check against, from
// testdata/length.json.
type picked struct {
	Exact           map[string]float64
	Parsec          float64
	PlanckLength    float64 `json:"planck length"`
	PlanckTolerance float64 `json:"planck tolerance"`
	FollowsTol      float64 `json:"follows tolerance"`
	Tolerance       float64
	Micron          float64
	KiloparsecIn    float64 `json:"kiloparsecs in parsecs"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/length.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// near is two numbers the same to a relative tolerance.
func near(got, want, tolerance float64) bool {
	larger := math.Max(math.Abs(got), math.Abs(want))
	return math.Abs(got-want) <= tolerance*larger
}

// exactly is every exact length by its name in the test data.
var exactly = map[string]float64{
	"metre": length.Metre, "angstrom": length.Angstrom,
	"au": length.AU, "light-year": length.LightYear,
}

// The lengths are the published figures: the exact ones to the last
// digit, the parsec to its tolerance, the Planck length to its own.
func TestTheLengthsAreThePublishedFigures(test *testing.T) {
	want := pickedNumbers(test)
	for name, published := range want.Exact {
		if got := exactly[name]; got != published {
			test.Errorf("the %s is %v m, not %v", name, got, published)
		}
	}
	if !near(length.Parsec, want.Parsec, want.Tolerance) {
		test.Errorf("the parsec is %v m", length.Parsec)
	}
	if !near(length.PlanckLength(), want.PlanckLength,
		want.PlanckTolerance) {
		test.Errorf("the Planck length is %v m", length.PlanckLength())
	}
}

// The anti-test: the light-year is not the parsec, and neither is the
// AU; a slip that set one to another shows.
func TestTheLengthsAreDistinct(test *testing.T) {
	au, year, parsec := length.AU, length.LightYear, length.Parsec
	if year == parsec || au == year || au == parsec {
		test.Error("two astronomical lengths are the same")
	}
	if au >= year || year >= parsec {
		test.Errorf("AU %v, light-year %v and parsec %v are out of order",
			au, year, parsec)
	}
}

// arcsecondsInHalfACircle is 180 degrees of 3600 arcseconds. Defined by
// the degree and the arcsecond.
const arcsecondsInHalfACircle = 180 * 3600

// A parsec is the distance at which an AU subtends one arcsecond: an AU
// over a parsec is that arcsecond, in radians.
func TestAParsecIsWhereAnAUIsAnArcsecond(test *testing.T) {
	arcsecond := math.Pi / arcsecondsInHalfACircle
	subtends := math.Atan(length.AU / length.Parsec)
	if !near(subtends, arcsecond, pickedNumbers(test).Tolerance) {
		test.Errorf("an AU at a parsec is %v rad", subtends)
	}
}

// degreesInHalfACircle is defined by the degree.
const degreesInHalfACircle = 180

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// The anti-test: the parsec is not an AU over an arcsecond taken in
// degrees, the slip of a unit that would make it far too short.
func TestTheParsecIsNotTakenInDegrees(test *testing.T) {
	arcsecondInDegrees := float64(degreesInHalfACircle) /
		arcsecondsInHalfACircle
	slipped := length.AU / math.Tan(arcsecondInDegrees)
	if math.IsInf(slipped, eitherSign) {
		test.Fatalf("the slip's parsec is %v, so the test shows nothing",
			slipped)
	}
	if near(length.Parsec, slipped, pickedNumbers(test).Tolerance) {
		test.Errorf("the parsec %v is the slip's", length.Parsec)
	}
}

// Multiples come from the SI's prefixes, not from names here: a micron is a
// millionth of a metre and a kiloparsec a thousand parsecs.
func TestMultiplesComeFromThePrefixes(test *testing.T) {
	want := pickedNumbers(test)
	if got := si.Micro(length.Metre); got != want.Micron {
		test.Errorf("a micron is %v m", got)
	}
	kiloparsec := si.Kilo(length.Parsec)
	if !near(kiloparsec, want.KiloparsecIn*length.Parsec, want.Tolerance) {
		test.Errorf("a kiloparsec is %v m", kiloparsec)
	}
}

// The anti-test: the Planck length follows G. Squared, times c cubed over hbar,
// it is G again, to a few roundings, which a figure typed in its place would
// miss: hbar here is SI's exact Planck constant over a turn, not length's own.
func TestThePlanckLengthFollowsG(test *testing.T) {
	tolerance := pickedNumbers(test).FollowsTol
	if !(tolerance > nothing) {
		test.Fatalf("no tolerance to hold to: %v", tolerance)
	}
	planck := length.PlanckLength()
	hbar := si.Planck / fullTurn
	back := planck * planck * math.Pow(si.Speed, lightPower) / hbar
	if !near(back, si.Gravitation(), tolerance) {
		test.Errorf("the Planck length %v m gives back G of %v, not %v",
			planck, back, si.Gravitation())
	}
}

// The numbers the Planck length is undone by, each defined: a turn in
// radians, the power of c in it, and no tolerance.
const (
	fullTurn   = 2 * math.Pi // defined: a turn is two pi radians
	lightPower = 3           // defined by the formula: c cubed
	nothing    = 0.0         // defined by the scale: no tolerance
)
