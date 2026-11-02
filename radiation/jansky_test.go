package radiation_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/radiation"
)

// one is a single jansky. It is defined by counting.
const one = 1.0

// A jansky is 1e-26 watts a square metre a hertz, and the AB zero point of
// 3631 janskys is 3.631e-23, both worked by hand in testdata/radiation.json.
func TestTheJanskyInSI(test *testing.T) {
	jansky := pickedNumbers(test).Jansky
	if got := radiation.JanskyToSI(one); !near(got, jansky.One, jansky.Tolerance) {
		test.Errorf("a jansky is %v W m^-2 Hz^-1", got)
	}
	if got := radiation.JanskyToSI(jansky.ABZeroJy); !near(got, jansky.ABZeroSI, jansky.Tolerance) {
		test.Errorf("3631 Jy is %v W m^-2 Hz^-1", got)
	}
}

// The anti-test: the CGS value of a jansky, 1e-23, is not its SI value, which
// is the slip the units invite. Zero stays zero and a negative density keeps
// its sign, so nothing is clamped on the way.
func TestAJanskyIsNotItsCGSValue(test *testing.T) {
	jansky := pickedNumbers(test).Jansky
	if radiation.JanskyToSI(one) == jansky.CGSOne {
		test.Error("a jansky was taken in cgs")
	}
	if radiation.JanskyToSI(none) != none {
		test.Error("no flux became some")
	}
	if radiation.JanskyToSI(-one) >= none {
		test.Error("a negative density lost its sign")
	}
}

// The property: the conversion is one factor for every flux density, so
// the ratio of two densities is the same in janskys and in SI.
func TestJanskysKeepTheirRatios(test *testing.T) {
	numbers := pickedNumbers(test).Jansky
	spread := jyRatios{numbers.Lowest, numbers.Highest, numbers.Tolerance}
	rapid.Check(test, spread.check)
}

// jyRatios is the numbers of the property above: the janskys drawn from
// and the relative tolerance.
type jyRatios struct{ lowest, highest, tolerance float64 }

// check is one draw of the property above.
func (spread jyRatios) check(draw *rapid.T) {
	first := rapid.Float64Range(spread.lowest, spread.highest).Draw(draw, "first")
	second := rapid.Float64Range(spread.lowest, spread.highest).Draw(draw, "second")
	got := radiation.JanskyToSI(first) / radiation.JanskyToSI(second)
	if !near(got, first/second, spread.tolerance) {
		draw.Fatalf("%v Jy over %v Jy became %v", first, second, got)
	}
}
