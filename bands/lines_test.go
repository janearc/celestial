package bands_test

import (
	"maps"
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// lineOf is a single line of light at a wavelength.
func lineOf(nm, power float64) bands.Light {
	spectrum := map[float64]float64{}
	spectrum[nm] = power
	return bands.Lines(spectrum)
}

// wholeNanometres lists the whole nanometres in a band.
func wholeNanometres(band bands.Band) []int {
	lo, hi := band.Edges()
	var inside []int
	for nm := int(lo); float64(nm) < hi; nm += oneNM {
		if float64(nm) >= lo && bands.Of(float64(nm)) == band {
			inside = append(inside, nm)
		}
	}
	return inside
}

// Lines is Split for wavelengths that are not whole nanometres: given whole
// ones, it is Split exactly.
func TestLinesAgreesWithSplit(test *testing.T) {
	rapid.Check(test, linesAreSplit)
}

// linesAreSplit is one draw of the property above.
func linesAreSplit(test *rapid.T) {
	whole := spectrum(test, "spectrum")
	lines := map[float64]float64{}
	for nm, power := range whole {
		lines[float64(nm)] = power
	}
	fromSplit, fromLines := bands.Split(whole), bands.Lines(lines)
	splitFirst, splitLast := fromSplit.Span()
	linesFirst, linesLast := fromLines.Span()
	if splitFirst != linesFirst || splitLast != linesLast {
		test.Fatalf("split spans %d to %d, lines %d to %d",
			splitFirst, splitLast, linesFirst, linesLast)
	}
	if !sameLight(fromSplit, fromLines) {
		test.Fatalf("split and lines are different lights")
	}
}

// The anti-test: Lines keeps the wavelength it is given, not the whole
// nanometre under it. A line at a fraction of a nanometre is in a band of
// its own, which Split could not say.
func TestLinesKeepsAFractionOfANanometre(test *testing.T) {
	numbers := pickedNumbers(test)
	band := bands.Of(numbers.GammaNM)
	if band >= bands.Of(oneNM) {
		test.Fatal("the gamma line is expected to be below the first band")
	}
	light := lineOf(numbers.GammaNM, numbers.GammaPower)
	if light.Power(band) != numbers.GammaPower {
		test.Errorf("the line is %v in its band", light.Power(band))
	}
}

// Lines carries radiation far shorter than a nanometre: a gamma ray at a
// ten-thousandth of one is in its own band, with all its power.
func TestLinesCarriesGamma(test *testing.T) {
	numbers := pickedNumbers(test)
	light := bands.Lines(map[float64]float64{numbers.GammaNM: numbers.GammaPower})
	want := bands.Of(numbers.GammaNM)
	if first, last := light.Span(); first != want || last != want {
		test.Errorf("a gamma line spans %d to %d", first, last)
	}
	if light.Power(want) != numbers.GammaPower {
		test.Errorf("a gamma line: %v in its band", light.Power(want))
	}
}

// Power of exactly one is light and counts: it is in its band, and the
// band is the whole span.
func TestAPowerOfOneCounts(test *testing.T) {
	numbers := pickedNumbers(test)
	light := lineOf(numbers.LineNM, unitPower)
	band := bands.Of(numbers.LineNM)
	if first, last := light.Span(); first != band || last != band {
		test.Errorf("a line spans %d to %d", first, last)
	}
	if light.Power(band) != unitPower {
		test.Errorf("a power of one at %v nm is %v", numbers.LineNM, light.Power(band))
	}
}

// The anti-test: light at no wavelength is no light. It is left out, and
// the result is the same without it.
func TestLinesIgnoreAWavelengthOfNothing(test *testing.T) {
	numbers := pickedNumbers(test)
	spectrum := map[float64]float64{}
	spectrum[numbers.LineNM] = unitPower
	without := bands.Lines(spectrum)
	spectrum[none] = unitPower
	with := bands.Lines(spectrum)
	withFirst, withLast := with.Span()
	first, last := without.Span()
	if withFirst != first || withLast != last || !sameLight(with, without) {
		test.Errorf("a line at nothing changed the light: %d to %d", withFirst, withLast)
	}
}

// The anti-test: a wavelength shorter than nothing is ignored too, and
// lines of nothing but those are no light at all.
func TestLinesOfNothingAreNoLight(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, nm := range []float64{none, -unitPower, -numbers.LineNM} {
		light := bands.Lines(map[float64]float64{nm: numbers.LinePower})
		if first, last := light.Span(); first <= last {
			test.Errorf("a line at %v nm spans %d to %d", nm, first, last)
		}
	}
	if first, last := bands.Lines(nil).Span(); first <= last {
		test.Errorf("no lines span %d to %d", first, last)
	}
}

// The anti-test: light at an endless wavelength, or at none that is a
// number, is no light either, and the rest is the same without it: no
// band holds either.
func TestLinesIgnoreAnEndlessWavelength(test *testing.T) {
	numbers := pickedNumbers(test)
	spectrum := map[float64]float64{numbers.LineNM: unitPower}
	without := bands.Lines(spectrum)
	for _, nm := range []float64{math.Inf(endless), math.NaN()} {
		spectrum[nm] = unitPower
		with := bands.Lines(spectrum)
		withFirst, withLast := with.Span()
		first, last := without.Span()
		if withFirst != first || withLast != last ||
			!sameLight(with, without) {
			test.Errorf("a line at %v nm changed the light", nm)
		}
		delete(spectrum, nm)
	}
}

// endless asks math.Inf for the infinity above. Defined by math.Inf.
const endless = 1

// The anti-test: Lines reads the spectrum it is given and leaves it as it
// was. The lines it leaves out are still the caller's; a map is shared,
// and taking them out would change what the caller holds.
func TestLinesLeavesTheSpectrumAsItWas(test *testing.T) {
	numbers := pickedNumbers(test)
	given := map[float64]float64{numbers.LineNM: unitPower,
		none: unitPower, -numbers.LineNM: unitPower,
		math.Inf(endless): unitPower}
	before := maps.Clone(given)
	bands.Lines(given)
	if !maps.Equal(given, before) {
		test.Errorf("Lines changed the spectrum to %v", given)
	}
}

// The same spectrum is the same light, every time it is asked for. A map
// is read in no order, so a sum that depended on the order would show.
func TestTheSameSpectrumIsTheSameLight(test *testing.T) {
	numbers := pickedNumbers(test)
	want := bands.Split(numbers.Spectrum)
	for range numbers.Repeats {
		if got := bands.Split(numbers.Spectrum); !sameLight(got, want) {
			test.Fatal("the same spectrum came back as another light")
		}
	}
}

// The anti-test: a spectrum with every wavelength's power changed is
// another light.
func TestAnotherSpectrumIsAnotherLight(test *testing.T) {
	numbers := pickedNumbers(test)
	other := map[int]float64{}
	for nm, power := range numbers.Spectrum {
		other[nm] = power + unitPower
	}
	if sameLight(bands.Split(numbers.Spectrum), bands.Split(other)) {
		test.Error("a different spectrum is the same light")
	}
}

// A band takes the whole nanometres in it and no others: the next
// nanometre above the last in a band is the first of the next band.
func TestTheNextNanometreIsTheNextBand(test *testing.T) {
	band := bands.Band(pickedNumbers(test).Band)
	inside := wholeNanometres(band)
	if len(inside) == none {
		test.Fatalf("band %d has no whole nanometre", band)
	}
	next := float64(slices.Max(inside) + oneNM)
	if bands.Of(next) != band+neighbour {
		test.Errorf("%v nm is band %d, not %d", next, bands.Of(next), band+neighbour)
	}
	below := float64(slices.Min(inside) - oneNM)
	if bands.Of(below) != band-neighbour {
		test.Errorf("the nanometre below band %d is not in band %d", band, band-neighbour)
	}
}
