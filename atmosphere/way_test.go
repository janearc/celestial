//ninja:vow dojo a way's own parts are held to being one long and along what made it, which atmosphere does not export
package atmosphere

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// partsOf is a way's three parts as the test data writes them.
type partsOf struct {
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// madeWays is what the way tests read from testdata/atmosphere.json.
type madeWays struct {
	Ways struct {
		Parts     partsOf   `json:"parts"`
		Length    float64   `json:"length"`
		Scales    []float64 `json:"scales"`
		Edge      partsOf   `json:"edge"`
		Tolerance float64   `json:"tolerance"`
	} `json:"ways"`
}

// readWays reads the way tests' numbers, or the test stops.
func readWays(test *testing.T) madeWays {
	test.Helper()
	var numbers madeWays
	raw, err := os.ReadFile("testdata/atmosphere.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// alike is whether a value is within a tolerance of what is wanted.
func alike(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// A way is one long, and each of its parts is its maker's part over the
// maker's length, at whatever scale the parts are given: three, minus
// four and twelve make thirteen, as tens of three hundred or as tenths
// of three hundred.
func TestAWayIsOneLongAlongItsParts(test *testing.T) {
	ways := readWays(test).Ways
	parts, tolerance := ways.Parts, ways.Tolerance
	for _, scale := range ways.Scales {
		way, err := NewWay(parts.AlongX*scale, parts.AlongY*scale,
			parts.AlongZ*scale)
		if err != nil {
			test.Fatalf("at a scale of %v: %v", scale, err)
		}
		if self := way.dot(*way); !alike(self, whole, tolerance) {
			test.Errorf("at a scale of %v the way is %v long", scale, self)
		}
		for _, part := range []struct{ got, want float64 }{
			{way.x, parts.AlongX / ways.Length},
			{way.y, parts.AlongY / ways.Length},
			{way.z, parts.AlongZ / ways.Length}} {
			if !alike(part.got, part.want, tolerance) {
				test.Errorf("at a scale of %v the way is %v", scale, *way)
			}
		}
	}
}

// A part as large as a float holds is still a way, and points along that
// part: the edge of what NewWay takes, beside the endless part
// TestAWayThatPointsNowhereIsRefused holds it to refusing.
func TestAWayAtTheFloatsTopIsAWay(test *testing.T) {
	ways := readWays(test).Ways
	edge := ways.Edge
	way, err := NewWay(edge.AlongX, edge.AlongY, edge.AlongZ)
	if err != nil {
		test.Fatalf("a way of %v was refused: %v", edge, err)
	}
	if !alike(way.x, whole, ways.Tolerance) || way.y != none ||
		way.z != none {
		test.Errorf("a way of %v is %v", edge, *way)
	}
}
