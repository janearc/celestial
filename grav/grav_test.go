package grav_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/measure"
	"pgregory.net/rapid"
)

// picked is the numbers these tests use, from testdata/grav.json.
type picked struct {
	NotSuns   []float64 `json:"not suns"`
	OneSun    float64   `json:"one sun"`
	Tolerance float64   `json:"tolerance"`
	Drawn     struct {
		Lowest  int `json:"lowest power"`
		Highest int `json:"highest power"`
	} `json:"drawn"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/grav.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// The anti-test: a point refuses what weighs nothing, less than nothing,
// no number or endlessly much, and gives no body.
func TestAPointRefusesWhatWeighsNothing(test *testing.T) {
	given := append(pickedNumbers(test).NotSuns, math.NaN(),
		math.Inf(eitherSign))
	for _, suns := range given {
		body, err := grav.Point(measure.SolarMasses(suns))
		if err == nil || body != nil {
			test.Errorf("a point of %v suns was made", suns)
		}
	}
}

// A point of a sun's mass has the sun's GM, and weighs a sun.
func TestAPointOfASunHasTheSunsGM(test *testing.T) {
	numbers := pickedNumbers(test)
	sun, err := grav.Point(measure.Sun().Mass())
	if err != nil {
		test.Fatal(err)
	}
	gm, want := sun.GM().InMetres(), measure.Sun().GM().InMetres()
	if !(math.Abs(gm-want) <= numbers.Tolerance*want) {
		test.Errorf("a sun's GM is %v m, not %v", gm, want)
	}
	suns := sun.Mass().InSolarMasses()
	if !(math.Abs(suns-numbers.OneSun) <= numbers.Tolerance) {
		test.Errorf("a sun weighs %v suns", suns)
	}
}

// The property and its anti-test: a point of any mass weighs that mass
// and has the sun's GM for each sun of it, GM being the mass times a
// constant; and a point of the same mass below nought is refused.
func TestAnyPointsGMIsItsMassInSuns(test *testing.T) {
	numbers := pickedNumbers(test)
	if numbers.Drawn.Highest <= numbers.Drawn.Lowest {
		test.Fatalf("no powers to draw from: %+v", numbers.Drawn)
	}
	rapid.Check(test, weighing{numbers: numbers,
		sunGM: measure.Sun().GM().InMetres()}.check)
}

// weighing draws masses and weighs points of them.
type weighing struct {
	numbers picked
	sunGM   float64
}

// The parts a drawn mass is made of, each defined.
const (
	leastMantissa = 1.0  // defined: a mantissa's least
	mostMantissa  = 10.0 // defined: a mantissa's most, one decade on
	below         = -1.0 // defined: the sign of a mass below nought
)

// check draws a mass in suns, weighs a point of it, and refuses one of
// it below nought.
func (weigh weighing) check(test *rapid.T) {
	numbers := weigh.numbers
	mantissa := rapid.Float64Range(leastMantissa, mostMantissa).Draw(test,
		"mantissa")
	power := rapid.IntRange(numbers.Drawn.Lowest,
		numbers.Drawn.Highest).Draw(test, "power")
	suns := mantissa * math.Pow10(power)
	body, err := grav.Point(measure.SolarMasses(suns))
	if err != nil {
		test.Fatalf("a point of %v suns: %v", suns, err)
	}
	if got := body.Mass().InSolarMasses(); !(math.Abs(got-suns) <=
		numbers.Tolerance*suns) {
		test.Fatalf("a point of %v suns weighs %v", suns, got)
	}
	perSun := body.GM().InMetres() / suns
	if !(math.Abs(perSun-weigh.sunGM) <= numbers.Tolerance*weigh.sunGM) {
		test.Fatalf("%v suns have %v m of GM a sun", suns, perSun)
	}
	if body, err := grav.Point(measure.SolarMasses(below * suns)); err ==
		nil || body != nil {
		test.Fatalf("a point of %v suns was made", below*suns)
	}
}
