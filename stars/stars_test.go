package stars_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/stars"
)

// birth is a star's mass in suns and age in years, as the test data
// gives them.
type birth struct {
	Why   string  `json:"why"`
	Suns  float64 `json:"suns"`
	Years float64 `json:"years"`
	Life  string  `json:"life"`
}

// picked is the numbers these tests use, from testdata/stars.json.
type picked struct {
	Refused []birth `json:"refused"`
	Sun     struct {
		birth
		Kelvin    float64 `json:"kelvin"`
		KelvinTol float64 `json:"kelvin tolerance"`
		LightTol  float64 `json:"light tolerance"`
	} `json:"sun"`
	SunBurning birth `json:"sun burning"`
	SunEnded   birth `json:"sun ended"`
	RedDwarf   struct {
		birth
		Hottest   float64 `json:"hottest kelvin"`
		Brightest float64 `json:"brightest suns"`
	} `json:"red dwarf"`
	YoungGiant struct {
		birth
		Coolest float64 `json:"coolest kelvin"`
	} `json:"young giant"`
	OldGiant birth   `json:"old giant"`
	Ends     []birth `json:"ends"`
	Dark     birth   `json:"dark"`
	Masses   struct {
		Least float64 `json:"least"`
		Most  float64 `json:"most"`
	} `json:"masses"`
	Sequence struct {
		Lightest  float64 `json:"lightest"`
		Heaviest  float64 `json:"heaviest"`
		LeastPast float64 `json:"least share past"`
		MostPast  float64 `json:"most share past"`
	} `json:"ends of the sequence"`
	Between struct {
		Lighter float64 `json:"lighter"`
		Heavier float64 `json:"heavier"`
		Between float64 `json:"between"`
	} `json:"between rows"`
	OutOfOrder struct {
		Lightest float64 `json:"lightest"`
		Heaviest float64 `json:"heaviest"`
	} `json:"out of order"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/stars.json")
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

// born is the star of a birth, or the test stops.
func born(test stopper, given birth) stars.Star {
	test.Helper()
	made, err := stars.NewStar(measure.SolarMasses(given.Suns),
		measure.Years(given.Years))
	if err != nil {
		test.Fatal(err)
	}
	return *made
}

// near is whether a value lands within a share of what is published for
// it.
func near(got, published, share float64) bool {
	return math.Abs(got-published) <= share*math.Abs(published)
}

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// The anti-test: a star refuses what is no star's mass or no star's
// age, a NaN or an endless one too, and then gives no star.
func TestAStarRefusesWhatIsNotAStar(test *testing.T) {
	refused := pickedNumbers(test).Refused
	for _, none := range []float64{math.NaN(), math.Inf(eitherSign)} {
		refused = append(refused, birth{Why: "no number", Suns: none,
			Years: pickedNumbers(test).Sun.Years},
			birth{Why: "no number of years", Suns: pickedNumbers(test).
				Sun.Suns, Years: none})
	}
	for _, given := range refused {
		star, err := stars.NewStar(measure.SolarMasses(given.Suns),
			measure.Years(given.Years))
		if err == nil || star != nil {
			test.Errorf("%s: a star of %v suns at %v years was made",
				given.Why, given.Suns, given.Years)
		}
	}
}

// The sun at its age is burning, at its temperature, and shines a sun.
func TestTheSunIsTheSun(test *testing.T) {
	want := pickedNumbers(test).Sun
	sun := born(test, want.birth)
	if sun.Life() != stars.Burning() {
		test.Fatalf("the sun is %v, not burning", sun.Life())
	}
	if kelvin := sun.Temperature().InKelvin(); !near(kelvin, want.Kelvin,
		want.KelvinTol) {
		test.Errorf("the sun is %v kelvin, not %v", kelvin, want.Kelvin)
	}
	light := sun.Luminosity().InSolarLuminosities()
	if !near(light, want.Suns, want.LightTol) {
		test.Errorf("the sun shines %v suns, not one", light)
	}
	radius := sun.Radius().InSolarRadii()
	if !near(radius, want.Suns, want.LightTol) {
		test.Errorf("the sun is %v suns across, not one", radius)
	}
}

// The sun burns for about ten billion years: burning at nine, ended at
// eleven.
func TestTheSunBurnsForAboutTenBillionYears(test *testing.T) {
	numbers := pickedNumbers(test)
	if life := born(test, numbers.SunBurning).Life(); life !=
		stars.Burning() {
		test.Errorf("the sun at %v years is %v", numbers.SunBurning.Years,
			life)
	}
	if life := born(test, numbers.SunEnded).Life(); life !=
		stars.Ended() {
		test.Errorf("the sun at %v years is %v", numbers.SunEnded.Years,
			life)
	}
}

// The anti-test: a red dwarf is not the sun. It is red and dim, cooler
// and far fainter.
func TestARedDwarfIsRedAndDim(test *testing.T) {
	want := pickedNumbers(test).RedDwarf
	dwarf := born(test, want.birth)
	kelvin := dwarf.Temperature().InKelvin()
	light := dwarf.Luminosity().InSolarLuminosities()
	if kelvin > want.Hottest || light > want.Brightest {
		test.Errorf("a red dwarf is %v kelvin and shines %v suns", kelvin,
			light)
	}
}

// A big star is hot, and dies young.
func TestABigStarIsHotAndDiesYoung(test *testing.T) {
	numbers := pickedNumbers(test)
	young := born(test, numbers.YoungGiant.birth)
	kelvin := young.Temperature().InKelvin()
	if young.Life() != stars.Burning() ||
		!(kelvin >= numbers.YoungGiant.Coolest) {
		test.Errorf("ten suns at ten million years is %v at %v kelvin",
			young.Life(), kelvin)
	}
	if old := born(test, numbers.OldGiant); old.Life() != stars.Ended() {
		test.Errorf("ten suns at a hundred million years is %v",
			old.Life())
	}
}

// A star ends when its mass says: the heavier, the sooner.
func TestAStarEndsWhenItsMassSays(test *testing.T) {
	for _, given := range pickedNumbers(test).Ends {
		made := born(test, given)
		if made.Life().String() != given.Life {
			test.Errorf("%v suns at %v years is %v, not %v", given.Suns,
				given.Years, made.Life(), given.Life)
		}
	}
}

// The anti-test: an ended star gives no light, has no size and no heat
// here.
func TestTheEndedGiveNoLight(test *testing.T) {
	ended := born(test, pickedNumbers(test).Dark)
	if ended.Life() != stars.Ended() {
		test.Fatalf("twelve suns at a hundred million years is %v",
			ended.Life())
	}
	if light := ended.Luminosity().InWatts(); light != nothing {
		test.Errorf("an ended star shines %v W", light)
	}
	if radius := ended.Radius().InMetres(); radius != nothing {
		test.Errorf("an ended star is %v m across", radius)
	}
	if kelvin := ended.Temperature().InKelvin(); kelvin != nothing {
		test.Errorf("an ended star is %v kelvin", kelvin)
	}
}

// nothing is no amount at all. Defined by the scale.
const nothing = 0.0

// The property and its anti-test: any mass the main sequence reaches
// makes a star that, newly born, is burning, and of two such masses the
// heavier is the brighter; and any mass just past either end of the
// table, which the fixed refusals stand well clear of, is refused and
// gives no star.
//
// Heavier is brighter is not held when both masses fall between A9V's
// 1.75 suns and A2V's 1.98. Mamajek's published masses rise twice down
// the sequence there, so in mass order his table dims twice. The rows
// are kept as published, and this test gives way instead: a decision of
// 2026-10-31, to be reviewed. The reasoning is in stars.json, "out of
// order about".
func TestEveryMassOnTheSequenceBurnsAndHeavierShinesMore(test *testing.T) {
	numbers := pickedNumbers(test)
	ends, odd := numbers.Sequence, numbers.OutOfOrder
	if !(ends.Lightest > nothing && ends.Heaviest > ends.Lightest &&
		ends.LeastPast > nothing && ends.MostPast > ends.LeastPast) {
		test.Fatalf("no ends to draw past: %+v", ends)
	}
	if !(odd.Lightest > ends.Lightest && odd.Heaviest > odd.Lightest &&
		ends.Heaviest > odd.Heaviest) {
		test.Fatalf("no stretch out of order inside the table: %+v", odd)
	}
	rapid.Check(test, sequence{numbers: numbers}.check)
}

// sequence draws two masses along the main sequence and compares them.
type sequence struct{ numbers picked }

// check draws two masses, newly born, and compares their light.
func (along sequence) check(test *rapid.T) {
	masses := along.numbers.Masses
	light := rapid.Float64Range(masses.Least, masses.Most).Draw(test, "suns")
	heavy := rapid.Float64Range(light, masses.Most).Draw(test, "heavier")
	lighter := born(test, birth{Suns: light})
	heavier := born(test, birth{Suns: heavy})
	if lighter.Life() != stars.Burning() ||
		heavier.Life() != stars.Burning() {
		test.Fatalf("newly born, %v suns is %v and %v is %v", light,
			lighter.Life(), heavy, heavier.Life())
	}
	if !along.bothOutOfOrder(light, heavy) &&
		!(heavier.Luminosity().Over(lighter.Luminosity()) >= whole) {
		test.Fatalf("%v suns shine less than %v", heavy, light)
	}
	ends := along.numbers.Sequence
	past := rapid.Float64Range(ends.LeastPast, ends.MostPast).Draw(test,
		"past")
	for _, suns := range []float64{ends.Lightest * (whole - past),
		ends.Heaviest * (whole + past)} {
		star, err := stars.NewStar(measure.SolarMasses(suns),
			measure.Years(nothing))
		if err == nil || star != nil {
			test.Fatalf("a star of %v suns, past the table, was made", suns)
		}
	}
}

// bothOutOfOrder says whether two masses are both in the stretch of the
// table where the published masses are not in the order of the types.
func (along sequence) bothOutOfOrder(one, other float64) bool {
	odd := along.numbers.OutOfOrder
	return one >= odd.Lightest && one <= odd.Heaviest &&
		other >= odd.Lightest && other <= odd.Heaviest
}

// The anti-test: a star between two rows of the table is not either
// row's star. Newly born, it is brighter and larger than the lighter
// row's and dimmer and smaller than the heavier's, so a main sequence
// that took the row below, or above, is caught; the property above
// allows a heavier star to shine only as much, and cannot.
func TestAStarBetweenTwoRowsIsBetweenThem(test *testing.T) {
	rows := pickedNumbers(test).Between
	if !(rows.Lighter < rows.Between && rows.Between < rows.Heavier) {
		test.Fatalf("no mass between the rows: %+v", rows)
	}
	lighter := born(test, birth{Suns: rows.Lighter})
	between := born(test, birth{Suns: rows.Between})
	heavier := born(test, birth{Suns: rows.Heavier})
	if !(lighter.Luminosity().Over(between.Luminosity()) < whole &&
		between.Luminosity().Over(heavier.Luminosity()) < whole) {
		test.Errorf("%v suns shine %v; the rows %v and %v",
			rows.Between, between.Luminosity(), lighter.Luminosity(),
			heavier.Luminosity())
	}
	if !(lighter.Radius().Over(between.Radius()) < whole &&
		between.Radius().Over(heavier.Radius()) < whole) {
		test.Errorf("%v suns are %v across; the rows %v and %v",
			rows.Between, between.Radius(), lighter.Radius(),
			heavier.Radius())
	}
}

// The anti-test: age moves a star only from one part of its life to the
// next. While it burns, its light, size and heat are its birth's: the
// sun newly born and the sun at its own age shine alike here, so age
// cannot have crept into how a star shines.
func TestAgeMovesOnlyThePartOfLife(test *testing.T) {
	numbers := pickedNumbers(test)
	newborn := born(test, birth{Suns: numbers.Sun.Suns})
	grown := born(test, numbers.Sun.birth)
	if newborn.Life() != grown.Life() ||
		newborn.Luminosity() != grown.Luminosity() ||
		newborn.Radius() != grown.Radius() ||
		newborn.Temperature() != grown.Temperature() {
		test.Errorf("the sun at %v years shines unlike the sun newborn",
			numbers.Sun.Years)
	}
}

// whole is a ratio of a thing to itself. Defined by division.
const whole = 1.0
