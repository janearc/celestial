//ninja:vow dojo the air's refractive index and king factor are held to bodhaine and the others' formulas as the code writes them, worked with bc, through refraction and king, which atmosphere does not export
package atmosphere

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// madeComposition is what the composition tests read from
// testdata/atmosphere.json.
type madeComposition struct {
	Composition struct {
		Microns    float64 `json:"microns"`
		CO2        float64 `json:"co2 ppm"`
		MoreCO2    float64 `json:"more co2 ppm"`
		Refraction float64 `json:"refraction"`
		King       float64 `json:"king"`
		Tolerance  float64 `json:"tolerance"`
	} `json:"composition"`
}

// readComposition reads the composition tests' numbers, or the test
// stops.
func readComposition(test *testing.T) madeComposition {
	test.Helper()
	var numbers madeComposition
	raw, err := os.ReadFile("testdata/atmosphere.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// airOf is dry air of so many parts a million of co2, or the test stops.
func airOf(test *testing.T, ppm float64) Composition {
	test.Helper()
	air, err := DryAir(ppm)
	if err != nil {
		test.Fatal(err)
	}
	return *air
}

// sameTo is whether a value is within a share of what is wanted.
func sameTo(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// At half a micron, air of 360 ppm of co2 has peck and reeder's index,
// scaled from their 300 ppm, as their formula comes out worked with bc,
// to a part in a million million: the scaling's own 300 ppm shows at a
// part in a million. The coefficients are the code's own, from the
// research data, so this holds the code to its formula, not the paper.
func TestTheIndexIsPeckAndReedersScaledForCO2(test *testing.T) {
	made := readComposition(test).Composition
	got := airOf(test, made.CO2).refraction(made.Microns)
	if !sameTo(got, made.Refraction, made.Tolerance) {
		test.Errorf("at %v um the index less one is %v, by hand %v",
			made.Microns, got, made.Refraction)
	}
}

// The anti-test: co2 bends light more than the air it stands in for, so
// air of more co2 has the larger index.
func TestMoreCO2BendsLightMore(test *testing.T) {
	made := readComposition(test).Composition
	less := airOf(test, made.CO2).refraction(made.Microns)
	more := airOf(test, made.MoreCO2).refraction(made.Microns)
	if !(more > less) {
		test.Errorf("at %v ppm the index less one is %v, at %v ppm %v",
			made.MoreCO2, more, made.CO2, less)
	}
}

// At half a micron, the king factor of air of 360 ppm of co2 is
// bodhaine and the others' 23 as it comes out worked with bc: co2's
// share, in percent, counted beside nitrogen's, oxygen's and argon's.
// As above, this holds the code to its own formula.
func TestTheKingFactorWeighsEachGas(test *testing.T) {
	made := readComposition(test).Composition
	got := airOf(test, made.CO2).king(made.Microns)
	if !sameTo(got, made.King, made.Tolerance) {
		test.Errorf("at %v um the king factor is %v, worked %v",
			made.Microns, got, made.King)
	}
}

// The anti-test: co2's molecules are less round than the rest, so air of
// more co2 has the larger king factor.
func TestCO2RaisesTheKingFactor(test *testing.T) {
	made := readComposition(test).Composition
	more := airOf(test, made.MoreCO2).king(made.Microns)
	less := airOf(test, made.CO2).king(made.Microns)
	if !(more > less) {
		test.Errorf("at %v ppm the king factor is %v, at %v ppm %v",
			made.MoreCO2, more, made.CO2, less)
	}
}
