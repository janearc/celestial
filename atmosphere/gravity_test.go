//ninja:vow dojo the pull of gravity by latitude and height is held to list's formula as the code writes it, worked with bc, through gravityAt, which atmosphere does not export
package atmosphere

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/measure"
)

// pulled is gravity at a latitude, as the test data writes it.
type pulled struct {
	Degrees float64 `json:"degrees"`
	Pull    float64 `json:"cm s2"`
}

// madeGravity is what the gravity tests read from
// testdata/atmosphere.json.
type madeGravity struct {
	Gravity struct {
		Metres    float64  `json:"metres"`
		At        []pulled `json:"at"`
		Tolerance float64  `json:"tolerance"`
	} `json:"gravity"`
}

// readGravity reads the gravity tests' numbers, or the test stops.
func readGravity(test *testing.T) madeGravity {
	test.Helper()
	var numbers madeGravity
	raw, err := os.ReadFile("testdata/atmosphere.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// Gravity at 50 km, at 30 and at 60 degrees, is list's formula as it
// comes out worked with bc, to a part in a million million: every term,
// the latitude's in each, counts there. The coefficients are the code's
// own, from the research data, so this holds the code to its formula,
// not the paper. A gravity that is no number fails it.
func TestGravityIsListsFormula(test *testing.T) {
	gravity := readGravity(test).Gravity
	for _, at := range gravity.At {
		got := gravityAt(measure.Degrees(at.Degrees), gravity.Metres)
		if !(math.Abs(got-at.Pull) <= gravity.Tolerance*at.Pull) {
			test.Errorf("at %v degrees and %v m gravity is %v, worked %v",
				at.Degrees, gravity.Metres, got, at.Pull)
		}
	}
}

// The anti-test: height pulls gravity down. At either latitude gravity
// at 50 km is less than at the sea.
func TestGravityIsLessHigherUp(test *testing.T) {
	gravity := readGravity(test).Gravity
	for _, at := range gravity.At {
		latitude := measure.Degrees(at.Degrees)
		high := gravityAt(latitude, gravity.Metres)
		low := gravityAt(latitude, seaLevel)
		if !(high < low) {
			test.Errorf("at %v degrees gravity is %v at %v m and %v at "+
				"the sea", at.Degrees, high, gravity.Metres, low)
		}
	}
}
