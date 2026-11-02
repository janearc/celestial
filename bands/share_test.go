package bands_test

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// even is a share that is part everywhere, or the test stops.
func even(test *testing.T, part float64) *bands.Share {
	test.Helper()
	share, err := bands.Even(part)
	if err != nil {
		test.Fatal(err)
	}
	return share
}

// pickedLight is the picked spectrum in bands.
func pickedLight(test *testing.T) bands.Light {
	test.Helper()
	return bands.Split(pickedNumbers(test).Spectrum)
}

func TestASurfaceThatPassesEverythingLeavesTheLightAsItIs(test *testing.T) {
	light := pickedLight(test)
	lit := bands.Lit(light, even(test, unitPower))
	first, last := light.Span()
	for band := first; band <= last; band++ {
		if lit.Power(band) != light.Power(band) {
			test.Fatalf("band %d: %v lit, %v before", band,
				lit.Power(band), light.Power(band))
		}
	}
}

func TestHalfTheShareIsHalfTheLight(test *testing.T) {
	light := pickedLight(test)
	half := pickedNumbers(test).Half
	lit := bands.Lit(light, even(test, half))
	first, last := light.Span()
	for band := first; band <= last; band++ {
		if lit.Power(band) != half*light.Power(band) {
			test.Fatalf("band %d: %v, want half of %v", band,
				lit.Power(band), light.Power(band))
		}
	}
}

// The property: light through an even share is, in every band, the light's
// power times the part, and the span is the light's.
func TestAnEvenShareScalesEveryBand(test *testing.T) {
	rapid.Check(test, scalesByPart)
}

// scalesByPart is one draw of the property above.
func scalesByPart(test *rapid.T) {
	given := lamp(test, "light")
	part := rapid.Float64Range(none, unitPower).Draw(test, "part")
	share, err := bands.Even(part)
	if err != nil {
		test.Fatal(err)
	}
	lit := bands.Lit(given, share)
	first, last := given.Span()
	if litFirst, litLast := lit.Span(); litFirst != first || litLast != last {
		test.Fatalf("lit spans %d to %d, not %d to %d", litFirst, litLast, first, last)
	}
	for band := first; band <= last; band++ {
		if lit.Power(band) != given.Power(band)*part {
			test.Fatalf("band %d: %v, not %v times %v", band,
				lit.Power(band), part, given.Power(band))
		}
	}
}

// The anti-test: a share's part in any band of its run never leaves the
// parts it was given. A mean stays between the least and the most given,
// and so does a straight line between two of them; a sum would not.
func TestASharesPartStaysWithinThePartsGiven(test *testing.T) {
	rapid.Check(test, staysWithin)
}

// staysWithin is one draw of the property above.
func staysWithin(test *rapid.T) {
	curve := rapid.MapOfN(
		rapid.IntRange(shortestDrawn, longestDrawn),
		rapid.Float64Range(none, unitPower),
		fewestDrawn, mostDrawn).Draw(test, "curve")
	share, err := bands.Shares(curve)
	if err != nil {
		test.Fatal(err)
	}
	least, most := math.Inf(endless), math.Inf(-endless)
	for _, part := range curve {
		least, most = min(least, part), max(most, part)
	}
	order := slices.Collect(maps.Keys(curve))
	shortest, longest := slices.Min(order), slices.Max(order)
	first, last := bands.Of(float64(shortest)), bands.Of(float64(longest))
	for band := first; band <= last; band++ {
		if part := share.Part(band); !(part >= least && part <= most) {
			test.Fatalf("band %d passes %v, outside %v to %v", band, part,
				least, most)
		}
	}
}

// The anti-test: a transmission fraction cannot increase incoming power in any
// band.
func TestAShareNeverMakesALamp(test *testing.T) {
	rapid.Check(test, neverBrighter)
}

// neverBrighter is one draw of the property above.
func neverBrighter(test *rapid.T) {
	given := lamp(test, "light")
	curve := rapid.MapOfN(
		rapid.IntRange(shortestDrawn, longestDrawn),
		rapid.Float64Range(none, unitPower),
		fewestDrawn, mostDrawn).Draw(test, "curve")
	share, err := bands.Shares(curve)
	if err != nil {
		test.Fatal(err)
	}
	lit := bands.Lit(given, share)
	first, last := given.Span()
	for band := first; band <= last; band++ {
		if lit.Power(band) > given.Power(band) {
			test.Fatalf("band %d: %v after, %v before", band,
				lit.Power(band), given.Power(band))
		}
	}
}

func TestAShareIsAveragedInABandNotSummed(test *testing.T) {
	// two wavelengths in one band; light would add them, a share
	// is the fraction passed there, so it is their mean
	numbers := pickedNumbers(test)
	same := numbers.SameBand
	band := bands.Of(float64(same.Low))
	if bands.Of(float64(same.High)) != band {
		test.Fatal("the two wavelengths are expected to share a band")
	}
	share, err := bands.Shares(map[int]float64{same.Low: same.LowP, same.High: same.HighP})
	if err != nil {
		test.Fatal(err)
	}
	if got := share.Part(band); !(math.Abs(got-same.Mean) <= numbers.Tolerance) {
		test.Errorf("band %d: part %v, want the mean %v", band, got, same.Mean)
	}
}

func TestABandBetweenSamplesIsInterpolated(test *testing.T) {
	// ends other than 0 and 1, so a sum in place of a difference shows
	numbers := pickedNumbers(test)
	between := numbers.Between
	share, err := bands.Shares(map[int]float64{
		between.Low: between.LowP, between.High: between.HighP})
	if err != nil {
		test.Fatal(err)
	}
	middle := bands.Of(between.Middle)
	lo, hi := middle.Edges()
	centre := math.Sqrt(lo * hi)
	along := (centre - float64(between.Low)) / float64(between.High-between.Low)
	want := between.LowP + (between.HighP-between.LowP)*along
	// written so a NaN fails it, not passes
	if got := share.Part(middle); !(math.Abs(got-want) <= numbers.Tolerance) {
		test.Errorf("band %d: part %v, want %v between the two",
			middle, got, want)
	}
}

// The anti-test: measured samples take precedence over interpolation. A band
// containing a supplied wavelength uses its measured fraction, not a line drawn
// between neighbouring bands.
func TestAGivenPartIsNotDrawnOver(test *testing.T) {
	numbers := pickedNumbers(test)
	between := numbers.Between
	lo, hi := bands.Of(between.Middle).Edges()
	centre := math.Sqrt(lo * hi)
	along := (centre - float64(between.Low)) /
		float64(between.High-between.Low)
	drawn := between.LowP + (between.HighP-between.LowP)*along
	if math.Abs(drawn-between.Sample) <= numbers.Tolerance {
		test.Fatal("the given part is expected to differ from the line")
	}
	share, err := bands.Shares(map[int]float64{
		between.Low: between.LowP, between.High: between.HighP,
		int(between.Middle): between.Sample})
	if err != nil {
		test.Fatal(err)
	}
	middle := bands.Of(between.Middle)
	got := share.Part(middle)
	if !(math.Abs(got-between.Sample) <= numbers.Tolerance) {
		test.Errorf("band %d: part %v, want the given %v, not the drawn %v",
			middle, got, between.Sample, drawn)
	}
}

// The anti-test: light through a share spans what the light does, not
// what the share does. A share wider than the light adds no bands to it,
// and one narrower takes none away: outside its run it darkens them.
func TestLitSpansTheLightNotTheShare(test *testing.T) {
	numbers := pickedNumbers(test)
	run := numbers.Run
	share, err := bands.Shares(map[int]float64{
		run.Low: unitPower, run.High: unitPower})
	if err != nil {
		test.Fatal(err)
	}
	low, high := bands.Of(float64(run.Low)), bands.Of(float64(run.High))
	narrow := bands.Split(map[int]float64{int(run.Inside): unitPower})
	if first, last := narrow.Span(); first <= low || last >= high {
		test.Fatal("the inside line is expected to be well inside the run")
	}
	wide := pickedLight(test)
	if first, last := wide.Span(); first >= low || last <= high {
		test.Fatal("the picked light is expected to reach past the run")
	}
	for name, light := range map[string]bands.Light{
		"narrower": narrow, "wider": wide} {
		wantFirst, wantLast := light.Span()
		first, last := bands.Lit(light, share).Span()
		if first != wantFirst || last != wantLast {
			test.Errorf("a light %s than the share, lit, spans %d to %d, "+
				"not %d to %d", name, first, last, wantFirst, wantLast)
		}
	}
}

// The anti-test: finite measured curves pass no light beyond either endpoint.
// Do not extend the endpoint value into unmeasured bands.
func TestAShareHoldsNothingPastItsRun(test *testing.T) {
	run := pickedNumbers(test).Run
	share, err := bands.Shares(map[int]float64{
		run.Low: unitPower, run.High: unitPower})
	if err != nil {
		test.Fatal(err)
	}
	below := bands.Of(float64(run.Low)) - neighbour
	above := bands.Of(float64(run.High)) + neighbour
	for _, band := range []bands.Band{below, above} {
		if part := share.Part(band); part != none {
			test.Errorf("band %d, past the run, passes %v", band, part)
		}
	}
}

// The test and the anti-test: inside a share's run light passes, and
// outside it nothing does.
func TestOutsideAShareNothingPasses(test *testing.T) {
	run := pickedNumbers(test).Run
	share, err := bands.Shares(map[int]float64{run.Low: unitPower, run.High: unitPower})
	if err != nil {
		test.Fatal(err)
	}
	spectrum := map[int]float64{int(run.Inside): unitPower, int(run.Outside): unitPower}
	lit := bands.Lit(bands.Split(spectrum), share)
	if got := lit.Power(bands.Of(run.Outside)); got != none {
		test.Errorf("%v nm, outside the share: %v passed, want 0", run.Outside, got)
	}
	if got := lit.Power(bands.Of(run.Inside)); got == none {
		test.Errorf("%v nm, inside the share: nothing passed", run.Inside)
	}
}

func TestNoLightOrNoShareIsNoLight(test *testing.T) {
	var nothing bands.Light
	if first, last := bands.Lit(nothing, even(test, unitPower)).Span(); first <= last {
		test.Errorf("no light, lit: span %d to %d, want empty", first, last)
	}
	var zero bands.Share
	given := pickedLight(test)
	first, last := given.Span()
	for _, lit := range []bands.Light{bands.Lit(given, &zero),
		bands.Lit(given, nil)} {
		for band := first; band <= last; band++ {
			if lit.Power(band) != none {
				test.Errorf("band %d: light through no share is %v, want 0",
					band, lit.Power(band))
			}
		}
	}
}

// The anti-test: a part past nothing or past all is not a fraction, and is
// refused, as is a share of no wavelengths. The refusal says what a
// fraction is.
func TestASharePastZeroOrOneIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, part := range append(numbers.Refused, math.NaN()) {
		if _, err := bands.Even(part); err == nil ||
			!strings.Contains(err.Error(), "0 and 1") {
			test.Errorf("Even(%v): %v, want a refusal naming 0 and 1",
				part, err)
		}
		if _, err := bands.Shares(map[int]float64{numbers.Run.Low: part}); err == nil {
			test.Errorf("Shares with %v: want a refusal", part)
		}
	}
	if _, err := bands.Shares(nil); err == nil {
		test.Errorf("Shares of nothing: want a refusal")
	}
}

// The anti-test: a refused share is no share. Beside the error there is
// nothing, so a caller who ignores the error has nothing to use.
func TestARefusedShareIsNoShare(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, part := range numbers.Refused {
		if share, _ := bands.Even(part); share != nil {
			test.Errorf("Even(%v) gave a share beside its refusal", part)
		}
		curve := map[int]float64{numbers.Run.Low: part}
		if share, _ := bands.Shares(curve); share != nil {
			test.Errorf("Shares with %v gave a share beside its refusal",
				part)
		}
	}
	for _, nm := range numbers.BadNM {
		curve := map[int]float64{nm: numbers.Half}
		if share, _ := bands.Shares(curve); share != nil {
			test.Errorf("Shares at %d nm gave a share beside its refusal",
				nm)
		}
	}
	if share, _ := bands.Shares(nil); share != nil {
		test.Error("Shares of nothing gave a share beside its refusal")
	}
}

// The anti-test: fraction errors identify the wavelength and allowed range so
// the bad sample can be found directly.
func TestAShareRefusalSaysWhere(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, part := range numbers.Refused {
		_, err := bands.Shares(map[int]float64{numbers.Run.Low: part})
		where := fmt.Sprintf("at %d nm", numbers.Run.Low)
		if err == nil || !strings.Contains(err.Error(), where) ||
			!strings.Contains(err.Error(), "0 and 1") {
			test.Errorf("Shares with %v at %d nm: %v, want a refusal "+
				"saying %q and what a fraction is", part, numbers.Run.Low,
				err, where)
		}
	}
	for _, nm := range numbers.BadNM {
		_, err := bands.Shares(map[int]float64{nm: numbers.Half})
		where := fmt.Sprintf("%d nm", nm)
		if err == nil || !strings.Contains(err.Error(), where) {
			test.Errorf("Shares at %d nm: %v, want a refusal saying %q",
				nm, err, where)
		}
	}
}

// The test and the anti-test: a share at no wavelength, or at less, is
// refused, and one at a nanometre, the shortest band 0 carries, is taken.
func TestAShareAtNoWavelengthIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, nm := range numbers.BadNM {
		share := map[int]float64{nm: numbers.Half, numbers.Run.Low: numbers.Half}
		if _, err := bands.Shares(share); err == nil {
			test.Errorf("a share at %d nm: want a refusal", nm)
		}
	}
	// The shortest wavelength bands carries is a nanometre, band 0
	share := map[int]float64{numbers.ShortestNM: numbers.Half, numbers.Run.Low: numbers.Half}
	if _, err := bands.Shares(share); err != nil {
		test.Errorf("a share at %d nm: %v, want it allowed", numbers.ShortestNM, err)
	}
}

func TestAShareOfNothingIsAllowed(test *testing.T) {
	// A surface that sends nothing back is black, and still a surface
	given := pickedLight(test)
	lit := bands.Lit(given, even(test, none))
	first, last := given.Span()
	for band := first; band <= last; band++ {
		if lit.Power(band) != none {
			test.Errorf("band %d: a share of 0 passed %v", band, lit.Power(band))
		}
	}
	if _, err := bands.Shares(map[int]float64{pickedNumbers(test).Run.Low: none}); err != nil {
		test.Errorf("a share of 0: %v, want it allowed", err)
	}
}
