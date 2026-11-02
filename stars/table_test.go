//ninja:vow dojo the published table is held to itself row by row, through readDwarf and shining, which stars does not export
package stars

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

// tableNumbers is what these tests check the table against, from
// testdata/stars.json.
type tableNumbers struct {
	Rows       int     `json:"rows"`
	Yield      float64 `json:"fusion yield"`
	YieldTol   float64 `json:"fusion tolerance"`
	BetweenTol float64 `json:"between tolerance"`
	Younger    float64 `json:"a moment younger"`
}

// readTableNumbers reads the test numbers, or the test stops.
func readTableNumbers(test *testing.T) tableNumbers {
	test.Helper()
	var numbers tableNumbers
	raw, err := os.ReadFile("testdata/stars.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// The table is read whole, every row of it, lightest first.
func TestTheTableIsReadWholeAndInOrder(test *testing.T) {
	table, err := mainSequence()
	if err != nil {
		test.Fatal(err)
	}
	if want := readTableNumbers(test).Rows; len(table) != want {
		test.Errorf("%d rows of the main sequence, not %d", len(table), want)
	}
	for place, row := range table[oneRow:] {
		if row.mass.Over(table[place].mass) < whole {
			test.Errorf("%s comes after %s", row.kind, table[place].kind)
		}
	}
}

// The anti-test: a row with too few fields, or a field that is not a
// number, is refused, and gives no row.
func TestARowThatIsNotOneIsRefused(test *testing.T) {
	for _, line := range []string{"G2V 1 5772", "G2V one 5772 0 1"} {
		row, err := readDwarf(strings.Fields(line))
		if err == nil || row != nil {
			test.Errorf("%q was read as a row", line)
		}
	}
	if rows, err := readDwarfs("G2V 1 5772"); err == nil || rows != nil {
		test.Errorf("a table of a bad row was read as %v", rows)
	}
}

// disagreeing are the rows of the published table whose temperature is
// further from what their luminosity and radius make than their rounding
// allows: the table disagrees with itself there.
var disagreeing = map[string]bool{
	"B5V": true, "B6V": true, "B8V": true, "M7V": true, "M9.5V": true,
}

// How a black body's light goes with its temperature, to the fourth, and
// its area with its radius, squared.
const (
	stefanPower = 4 // defined by stefan's law
	radiusPower = 2 // defined by a sphere's area
)

// The test and the anti-test: the table publishes temperatures beside
// luminosities and radii; ours are derived from those two, and agree as
// far as the digits the table gives allow, but for the rows where the
// table disagrees with itself. Each way is held: a row that disagrees
// and is not listed fails, and so does a listed row that agrees, so the
// list cannot hide a row.
func TestDerivedTemperaturesAgreeWithTheTable(test *testing.T) {
	for _, line := range strings.Split(mainSequenceText, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		row, err := readDwarf(fields)
		if err != nil {
			test.Fatal(err)
		}
		published := row.heat.InKelvin()
		derived := shining(*row).Temperature().InKelvin()
		off := math.Abs(derived/published - whole)
		light := halfPlace(fields[column("logL")]) * math.Ln10
		size := halfPlace(fields[column("radius")]) /
			row.radius.InSolarRadii()
		heat := halfPlace(fields[column("teff")]) / published
		allowed := (light+radiusPower*size)/stefanPower + heat
		if !(off <= allowed) != disagreeing[row.kind] {
			test.Errorf("%s: published %v kelvin, derived %.0f, off by "+
				"%.4f where its digits allow %.4f", row.kind, published,
				derived, off, allowed)
		}
	}
}

// writtenBase is the base the table's numbers are written in. Defined by
// the table.
const writtenBase = 10

// half is the half of a unit in the last place a number is good to.
// Defined by rounding.
const half = 0.5

// halfPlace is half a unit in the last place a published number is
// written to: 2.57 is good to 0.005, and 14500 to 50.
func halfPlace(field string) float64 {
	if point := strings.IndexByte(field, '.'); point >= firstRow {
		places := len(field) - point - oneRow
		return math.Pow(writtenBase, -float64(places)) * half
	}
	zeros := len(field) - len(strings.TrimRight(field, "0"))
	return math.Pow(writtenBase, float64(zeros)) * half
}

// Burning hydrogen to helium frees about seven thousandths of its mass:
// four hydrogens to a helium frees 26.73 MeV of their 3755.1 MeV.
func TestBurningHydrogenFreesSevenThousandthsOfItsMass(test *testing.T) {
	numbers := readTableNumbers(test)
	if share := fusionYield(); !(math.Abs(share-numbers.Yield) <=
		numbers.YieldTol*numbers.Yield) {
		test.Errorf("burning hydrogen frees %v of its mass", share)
	}
}
