package bands_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// The property: two lights on one place are, in every band, the sum of the
// two; it does not matter which is named first; and light added to no light
// is the light, from either side.
func TestAddingLightSumsEveryBand(test *testing.T) {
	rapid.Check(test, addsEveryBand)
}

// addsEveryBand is one draw of the property above.
func addsEveryBand(test *rapid.T) {
	left, right := lamp(test, "left"), lamp(test, "right")
	sum := bands.Add(left, right)
	if !sameLight(sum, bands.Add(right, left)) {
		test.Fatalf("adding in the other order gave another light")
	}
	leftFirst, leftLast := left.Span()
	rightFirst, rightLast := right.Span()
	for band := min(leftFirst, rightFirst); band <= max(leftLast, rightLast); band++ {
		if sum.Power(band) != left.Power(band)+right.Power(band) {
			test.Fatalf("band %d: %v, not %v and %v", band, sum.Power(band),
				left.Power(band), right.Power(band))
		}
	}
	nothing := bands.Light{}
	if !sameLight(bands.Add(nothing, left), left) || !sameLight(bands.Add(left, nothing), left) {
		test.Fatalf("no light changed a light")
	}
}

// Light adds: two lights on one place are, in every band, the sum of the
// two.
func TestLightAdds(test *testing.T) {
	numbers := pickedNumbers(test)
	one := bands.Split(numbers.Spectrum)
	other := bands.Split(numbers.OtherSpectrum)
	both := bands.Add(one, other)
	for _, spectrum := range []map[int]float64{numbers.Spectrum, numbers.OtherSpectrum} {
		for nm := range spectrum {
			band := bands.Of(float64(nm))
			if both.Power(band) != one.Power(band)+other.Power(band) {
				test.Errorf("band %d: %v is not %v plus %v", band,
					both.Power(band), one.Power(band), other.Power(band))
			}
		}
	}
}

// The anti-test: two lights are not one of them. The sum has power the
// first does not, where only the second shines.
func TestTwoLightsAreMoreThanOne(test *testing.T) {
	numbers := pickedNumbers(test)
	one := bands.Split(numbers.Spectrum)
	other := bands.Split(numbers.OtherSpectrum)
	both := bands.Add(one, other)
	if sameLight(both, one) || sameLight(both, other) {
		test.Error("two different lights together are one of them")
	}
}

// The anti-test: a light added to itself is not the light. It is twice
// the light in every band; the sum takes no shortcut for two lights that
// span the same bands.
func TestALightAddedToItselfIsTwiceTheLight(test *testing.T) {
	light := pickedLight(test)
	both := bands.Add(light, light)
	if sameLight(both, light) {
		test.Error("a light added to itself is the light")
	}
	if !sameLight(both, light.Scale(twice)) {
		test.Error("a light added to itself is not twice the light")
	}
}

// The anti-test: no light and no light is no light. It has no span.
func TestNoLightAndNoLightIsNoLight(test *testing.T) {
	both := bands.Add(bands.Light{}, bands.Light{})
	if first, last := both.Span(); first <= last {
		test.Errorf("no light and no light spans %d to %d", first, last)
	}
}

// No light is no light, whichever way it is asked for, and added to a
// light it leaves the light.
func TestNoLight(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, light := range []bands.Light{{}, bands.Split(nil), bands.Split(map[int]float64{})} {
		if first, last := light.Span(); first <= last {
			test.Errorf("no light spans %d to %d", first, last)
		}
		if power := light.Power(bands.Of(numbers.LineNM)); power != none {
			test.Errorf("no light has power %v", power)
		}
	}
	light := bands.Split(numbers.Spectrum)
	if !sameLight(bands.Add(bands.Light{}, light), light) {
		test.Error("no light plus a light is another light")
	}
}

// Adding keeps the span of the lights. It runs from the earlier first
// band to the later last band and no wider.
func TestAddSpansBothLightsExactly(test *testing.T) {
	numbers := pickedNumbers(test)
	left := bands.Split(numbers.Spectrum).Shift(numbers.Shift)
	right := bands.Split(numbers.OtherSpectrum).Shift(-numbers.Shift)
	leftFirst, leftLast := left.Span()
	rightFirst, rightLast := right.Span()
	first, last := bands.Add(left, right).Span()
	if first != min(leftFirst, rightFirst) || last != max(leftLast, rightLast) {
		test.Errorf("the sum spans %d to %d", first, last)
	}
}

// The anti-test: light added to no light spans what the light does and no
// wider, from either side. No light has no span to add.
func TestAddingNoLightDoesNotWidenTheSpan(test *testing.T) {
	numbers := pickedNumbers(test)
	light := bands.Split(numbers.Spectrum).Shift(numbers.Shift)
	var nothing bands.Light
	wantFirst, wantLast := light.Span()
	for name, sum := range map[string]bands.Light{
		"no light first": bands.Add(nothing, light),
		"no light last":  bands.Add(light, nothing),
	} {
		first, last := sum.Span()
		if first != wantFirst || last != wantLast {
			test.Errorf("%s spans %d to %d, not %d to %d", name, first, last, wantFirst, wantLast)
		}
	}
}

// The anti-test: adding light to another does not change the first, whose
// powers are its own: Add copies, it does not share.
func TestAddingDoesNotChangeEitherLight(test *testing.T) {
	rapid.Check(test, leavesBothAlone)
}

// leavesBothAlone is one draw of the property above.
func leavesBothAlone(test *rapid.T) {
	left, right := lamp(test, "left"), lamp(test, "right")
	first, last := left.Span()
	before := powers(left, first, last)
	bands.Add(left, right)
	bands.Add(bands.Light{}, left).Scale(strongest)
	bands.Add(left, bands.Light{}).Scale(strongest)
	after := powers(left, first, last)
	for place := range before {
		if before[place] != after[place] {
			test.Fatalf("band %d of the light changed", first+bands.Band(place))
		}
	}
}
