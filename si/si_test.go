package si_test

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
	"pgregory.net/rapid"
)

// picked is the numbers these tests check against, from testdata/si.json.
type picked struct {
	Powers    map[string]int
	Constants struct {
		Planck, Speed, Charge, Boltzmann, Avogadro float64
	}
	Scale struct {
		Units        float64
		KiloOfUnits  float64 `json:"kilo of units"`
		ThereAndBack float64 `json:"there and back"`
		Tolerance    float64
	}
	Drawn  drawn
	Undoes map[string]string
}

// drawn is the magnitudes the property draws from, as powers of ten,
// and how near a prefix must come to its power.
type drawn struct {
	Lowest    int     `json:"lowest power"`
	Highest   int     `json:"highest power"`
	Tolerance float64 `json:"relative tolerance"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/si.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// prefixes is every prefix function by its name in the brochure.
var prefixes = map[string]func(float64) float64{
	"nano": si.Nano, "micro": si.Micro, "milli": si.Milli,
	"centi": si.Centi, "kilo": si.Kilo, "mega": si.Mega,
	"giga": si.Giga, "tera": si.Tera,
}

// unit is one of anything. Defined: a prefix of one unit is its factor.
const unit = 1

// Each prefix of one unit is the power of ten the brochure gives it, and
// every prefix si holds is in the test data.
func TestPrefixesAreTheirPowersOfTen(test *testing.T) {
	powers := pickedNumbers(test).Powers
	if len(powers) != len(prefixes) {
		test.Errorf("the test data has %d prefixes and si %d",
			len(powers), len(prefixes))
	}
	for name, power := range powers {
		apply, held := prefixes[name]
		if !held {
			test.Errorf("si has no %s", name)
			continue
		}
		if got, want := apply(unit), math.Pow10(power); got != want {
			test.Errorf("%s of one unit is %v, not %v", name, got, want)
		}
	}
}

// The anti-test: no two prefixes are the same, so a slip that gave two
// names one power shows.
func TestNoTwoPrefixesAreTheSame(test *testing.T) {
	seen := map[float64]string{}
	for name, apply := range prefixes {
		if other, held := seen[apply(unit)]; held {
			test.Errorf("%s and %s are the same", name, other)
		}
		seen[apply(unit)] = name
	}
}

// A prefix scales the unit it is given, not one: three kilo-somethings
// are three thousand somethings, and a milli undoes a kilo.
func TestAPrefixScalesItsUnit(test *testing.T) {
	scale := pickedNumbers(test).Scale
	if got := si.Kilo(scale.Units); got != scale.KiloOfUnits {
		test.Errorf("kilo of %v is %v", scale.Units, got)
	}
	kilo := si.Kilo(scale.ThereAndBack)
	if back := si.Milli(kilo); !(math.Abs(back-scale.ThereAndBack) <=
		scale.Tolerance) {
		test.Errorf("a milli of a kilo of %v is %v", scale.ThereAndBack,
			back)
	}
}

// The anti-test: a prefix of no number is no number. None turns a NaN
// into a value a caller could take for a measurement.
func TestAPrefixKeepsNoNumberAsNone(test *testing.T) {
	for name, apply := range prefixes {
		if got := apply(math.NaN()); !math.IsNaN(got) {
			test.Errorf("%s of no number is %v", name, got)
		}
	}
}

// The property: a prefix of any number is that number times its power of
// ten, and the prefix of the opposite power undoes it.
func TestAnyNumberTakesItsPrefixesPower(test *testing.T) {
	numbers := pickedNumbers(test)
	if numbers.Drawn.Highest <= numbers.Drawn.Lowest ||
		len(numbers.Undoes) == noPairs {
		test.Fatalf("nothing to draw from: %+v", numbers.Drawn)
	}
	rapid.Check(test, scaling{numbers: numbers}.check)
}

// scaling draws a number and a prefix and scales one by the other.
type scaling struct{ numbers picked }

// The parts a drawn number is made of, each defined: a mantissa from one
// to ten, and a sign.
const (
	leastMantissa = 1.0  // defined: a mantissa's least
	mostMantissa  = 10.0 // defined: a mantissa's most, one decade on
	negative      = -1.0 // defined: the sign of a number below nought
	positive      = 1.0  // defined: the sign of one above
	noPairs       = 0    // defined: no prefix paired with its undoing
)

// check draws a magnitude, a sign and a prefix, and holds the prefix to
// its power and to the prefix that undoes it.
func (scale scaling) check(test *rapid.T) {
	numbers := scale.numbers
	mantissa := rapid.Float64Range(leastMantissa, mostMantissa).Draw(test,
		"mantissa")
	power := rapid.IntRange(numbers.Drawn.Lowest,
		numbers.Drawn.Highest).Draw(test, "power")
	sign := rapid.SampledFrom([]float64{negative, positive}).Draw(test,
		"sign")
	number := sign * mantissa * math.Pow10(power)
	names := slices.Sorted(maps.Keys(numbers.Powers))
	name := rapid.SampledFrom(names).Draw(test, "prefix")
	want := number * math.Pow10(numbers.Powers[name])
	if got := prefixes[name](number); !near(got, want,
		numbers.Drawn.Tolerance) {
		test.Fatalf("%s of %v is %v, not %v", name, number, got, want)
	}
	inverse, paired := numbers.Undoes[name]
	if !paired {
		return
	}
	if back := prefixes[inverse](prefixes[name](number)); !near(back,
		number, numbers.Drawn.Tolerance) {
		test.Fatalf("%s of %s of %v is %v", inverse, name, number, back)
	}
}

// near is whether two numbers agree to a share of the larger.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Max(math.Abs(got),
		math.Abs(want))
}

// fixed is a constant as si holds it and as the brochure fixes it.
type fixed struct {
	name          string
	held, written float64
}

// The defining constants are the brochure's, to the last digit.
func TestTheDefiningConstantsAreTheBrochures(test *testing.T) {
	want := pickedNumbers(test).Constants
	for _, constant := range []fixed{
		{name: "planck", held: si.Planck, written: want.Planck},
		{name: "speed", held: si.Speed, written: want.Speed},
		{name: "charge", held: si.Charge, written: want.Charge},
		{name: "boltzmann", held: si.Boltzmann, written: want.Boltzmann},
		{name: "avogadro", held: si.Avogadro, written: want.Avogadro},
	} {
		if constant.held != constant.written {
			test.Errorf("%s is %v, and the brochure fixes %v",
				constant.name, constant.held, constant.written)
		}
	}
}

// The anti-test: G is not one of the defined constants. It is measured,
// and comes from the research data, so a better measurement changes the
// data and no code.
func TestGravitationIsMeasuredNotDefined(test *testing.T) {
	read, err := research.Read()
	if err != nil {
		test.Fatal(err)
	}
	number, err := read.Get("gravitation")
	if err != nil {
		test.Fatal(err)
	}
	if si.Gravitation() != number.Value {
		test.Errorf("G is %v, and the research data holds %v",
			si.Gravitation(), number.Value)
	}
	if number.From == nil || number.From.Key == "" {
		test.Error("G names no source")
	}
}
