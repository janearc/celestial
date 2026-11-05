package radiation_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
)

// isoRanges is ISO 21348's named ranges by the name the test data uses.
func isoRanges() map[string]radiation.Range {
	return map[string]radiation.Range{
		"gamma ray":           radiation.Gamma(),
		"x-ray":               radiation.XRay(),
		"extreme ultraviolet": radiation.ExtremeUltraviolet(),
		"ultraviolet":         radiation.Ultraviolet(),
		"infrared":            radiation.Infrared(),
		"microwave":           radiation.Microwave(),
		"radio":               radiation.Radio(),
	}
}

// named is every named range by name, Fermi's with ISO's.
func named() map[string]radiation.Range {
	ranges := isoRanges()
	ranges["fermi"] = radiation.Fermi()
	return ranges
}

// The ranges are as the research ledger holds them, and every one the test
// data names is there. The ledger says where each end was read in ISO
// 21348:2007.
func TestTheNamedRanges(test *testing.T) {
	want := pickedNumbers(test).Named
	ranges := isoRanges()
	if len(ranges) != len(want) {
		test.Errorf("%d named ranges, not %d", len(ranges), len(want))
	}
	for name, edge := range want {
		short, long := ranges[name].Edges()
		if short != edge.Lo || long != edge.Hi {
			test.Errorf("%s runs %v, not %v to %v nm", name, ranges[name],
				edge.Lo, edge.Hi)
		}
	}
}

// The anti-test: every named range is a range, short end first, above
// nothing, and gamma rays are shorter than radio.
func TestEveryNamedRangeIsARange(test *testing.T) {
	for name, each := range named() {
		short, long := each.Edges()
		if made, err := radiation.NewRange(short, long); err != nil ||
			made == nil {
			test.Errorf("%s runs %v: %v", name, each, err)
		}
	}
	_, gammaLong := radiation.Gamma().Edges()
	radioShort, _ := radiation.Radio().Edges()
	if gammaLong >= radioShort {
		test.Error("gamma rays are not shorter than radio")
	}
}

// The test and the anti-test: Fermi's far end carries 10 keV and its near end
// 300 GeV, and the ends are not each other's energies.
func TestFermiEdgesCarryTheirEnergies(test *testing.T) {
	fermi := pickedNumbers(test).Fermi
	shortEnd, longEnd := radiation.Fermi().Edges()
	long, short := radiation.Energy(longEnd), radiation.Energy(shortEnd)
	if !(math.Abs(long-fermi.Lowest) <= fermi.Tolerance*fermi.Lowest) {
		test.Errorf("the long edge carries %v eV, not %v", long, fermi.Lowest)
	}
	if !(math.Abs(short-fermi.Highest) <= fermi.Tolerance*fermi.Highest) {
		test.Errorf("the short edge carries %v eV, not %v", short,
			fermi.Highest)
	}
	if near(short, long, fermi.Tolerance) {
		test.Errorf("both edges carry %v eV", short)
	}
}

// The anti-test: two wavelengths that make no range are refused, and give
// no range: a short end of nothing, below nothing or not a number, a long
// end not above the short, and an endless one.
func TestWhatIsNoRangeIsRefused(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	given := []edges{mapping.Flat.Render, mapping.BackRend.Render,
		mapping.ZeroSrc.Source, mapping.NegRend.Render,
		{Lo: math.NaN(), Hi: mapping.AboveNM},
		{Lo: mapping.InsideNM, Hi: math.NaN()},
		{Lo: mapping.InsideNM, Hi: math.Inf(upward)}}
	for _, edge := range given {
		made, err := radiation.NewRange(edge.Lo, edge.Hi)
		if err == nil || made != nil {
			test.Errorf("%v to %v nm made %v, %v", edge.Lo, edge.Hi, made,
				err)
		}
	}
}

// upward asks math.Inf for the endless above. Defined by math.Inf.
const upward = 1

// The anti-test: the zero Range, which is no range, holds nothing.
func TestTheZeroRangeHoldsNothing(test *testing.T) {
	var zero radiation.Range
	for _, nm := range []float64{none, pickedNumbers(test).Mapping.InsideNM} {
		if zero.Holds(nm) {
			test.Errorf("the zero range holds %v nm", nm)
		}
	}
}
