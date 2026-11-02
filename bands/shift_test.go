package bands_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// The anti-test: a shifted light is another light, and the light it was
// shifted from is as it was. Shift copies, it does not slide the original.
func TestShiftingMakesAnotherLight(test *testing.T) {
	numbers := pickedNumbers(test)
	light := bands.Split(numbers.Spectrum)
	first, last := light.Span()
	reach := last + bands.Band(numbers.Shift)
	before := powers(light, first, reach)
	shifted := light.Shift(numbers.Shift)
	if sameLight(light, shifted) {
		test.Error("a light slid several bands is the same light")
	}
	after := powers(light, first, reach)
	for place := range before {
		if before[place] != after[place] {
			test.Fatalf("band %d of the light changed", first+bands.Band(place))
		}
	}
	if !sameLight(light, light.Shift(none)) {
		test.Error("a light slid no bands is another light")
	}
}

// The property: shifting by any steps moves the span and the power in
// every band by those steps, and shifting back is the light.
func TestShiftIsASlideAlongTheBands(test *testing.T) {
	rapid.Check(test, slidesAlong)
}

// slidesAlong is one draw of the property above.
func slidesAlong(test *rapid.T) {
	light := bands.Split(spectrum(test, "spectrum"))
	steps := bands.Band(rapid.IntRange(-slide, slide).Draw(test, "steps"))
	shifted := light.Shift(int(steps))
	first, last := light.Span()
	from, to := shifted.Span()
	if from != first+steps || to != last+steps {
		test.Fatalf("span %d to %d shifted %d is %d to %d", first, last, steps, from, to)
	}
	for band := first; band <= last; band++ {
		if shifted.Power(band+steps) != light.Power(band) {
			test.Fatalf("band %d did not travel", band)
		}
	}
	back := shifted.Shift(-int(steps))
	if !sameLight(back, light) {
		test.Fatalf("shifting back is another light")
	}
}

// Twice the light is twice as strong in every band.
func TestTwiceTheLight(test *testing.T) {
	light := bands.Split(pickedNumbers(test).Spectrum)
	first, last := light.Span()
	doubled := light.Scale(twice)
	for band := first; band <= last; band++ {
		if doubled.Power(band) != twice*light.Power(band) {
			test.Errorf("band %d: %v, once it was %v", band,
				doubled.Power(band), light.Power(band))
		}
	}
}

// The property: scaling by k scales every band by k and leaves the span
// where it was.
func TestScaleMultipliesEveryBand(test *testing.T) {
	rapid.Check(test, scalesEvery)
}

// scalesEvery is one draw of the property above.
func scalesEvery(test *rapid.T) {
	light := bands.Split(spectrum(test, "spectrum"))
	factor := rapid.Float64Range(weakest, strongest).Draw(test, "factor")
	scaled := light.Scale(factor)
	first, last := light.Span()
	if from, to := scaled.Span(); from != first || to != last {
		test.Fatalf("scaling moved the span to %d to %d", from, to)
	}
	for band := first; band <= last; band++ {
		if scaled.Power(band) != light.Power(band)*factor {
			test.Fatalf("band %d: %v, not %v times %v", band,
				scaled.Power(band), factor, light.Power(band))
		}
	}
}

// The anti-test: scaling by nothing is no power in any band, scaling does
// not change the light it was made from, and scaling no light is no light.
func TestScalingByNothingIsDarkAndLeavesTheOriginal(test *testing.T) {
	light := bands.Split(pickedNumbers(test).Spectrum)
	first, last := light.Span()
	before := powers(light, first, last)
	for _, power := range powers(light.Scale(none), first, last) {
		if power != none {
			test.Errorf("light scaled by nothing has power %v", power)
		}
	}
	for place, power := range powers(light, first, last) {
		if power != before[place] {
			test.Errorf("scaling changed band %d", first+bands.Band(place))
		}
	}
	if from, to := (bands.Light{}).Scale(strongest).Span(); from <= to {
		test.Errorf("scaled no light spans %d to %d", from, to)
	}
}
