package radiation_test

import (
	"math"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/radiation"
)

// mid is the middle of a range the way the ranges are spread: evenly in
// the logarithm of wavelength.
func mid(each radiation.Range) float64 {
	short, long := each.Edges()
	return math.Sqrt(short * long)
}

// along is the wavelength a fraction of the way up a range, evenly in the
// logarithm of wavelength.
func along(each radiation.Range, fraction float64) float64 {
	short, long := each.Edges()
	return short * math.Pow(long/short, fraction)
}

// profiles are the named ranges, each mapped onto the picked target.
func profiles(test *testing.T) map[string]built {
	test.Helper()
	target := span(test, pickedNumbers(test).Mapping.Target)
	out := map[string]built{}
	for name, each := range named() {
		out[name] = profileOf(test, name, each, target)
	}
	return out
}

// wide is the picked wide profile, wider than it is shown.
func wide(test *testing.T) built {
	test.Helper()
	mapping := pickedNumbers(test).Mapping
	return profileOf(test, "wide", span(test, mapping.Wide),
		span(test, mapping.Target))
}

// A profile holds its own range, from its short end up to but not
// including its long end, and nothing outside it.
func TestAProfileHoldsItsOwnRangeOnly(test *testing.T) {
	inside := pickedNumbers(test).Mapping.JustInside
	for name, profile := range profiles(test) {
		short, long := profile.source.Edges()
		for _, nm := range []float64{short, mid(profile.source),
			long * inside} {
			if !profile.Holds(nm) {
				test.Errorf("%s does not hold %g nm", name, nm)
			}
		}
		for _, nm := range []float64{short * inside, long, long * twice} {
			if profile.Holds(nm) {
				test.Errorf("%s holds %g nm, outside it", name, nm)
			}
		}
	}
}

// The anti-test: no profile holds no wavelength, a wavelength of nothing
// or a number that is not one, and the zero Profile holds nothing at all.
func TestAProfileHoldsNoSuchWavelength(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	for name, profile := range profiles(test) {
		for _, nm := range []float64{none, -mid(profile.source),
			math.NaN(), math.MaxFloat64} {
			if profile.Holds(nm) {
				test.Errorf("%s holds %v nm", name, nm)
			}
		}
	}
	var zero radiation.Profile
	if zero.Holds(mapping.InsideNM) {
		test.Errorf("the zero profile holds %v nm", mapping.InsideNM)
	}
}

// The bottom of a profile's range lands on the bottom of what it is
// mapped onto, the middle on the middle, and anything inside stays
// inside.
func TestOntoLandsInsideTheRender(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	for name, profile := range profiles(test) {
		short, long := profile.source.Edges()
		renderShort, renderLong := profile.render.Edges()
		bottom, ok := profile.Onto(short)
		if !ok || !near(bottom, renderShort, mapping.Tolerance) {
			test.Errorf("%s: the bottom lands at %v, ok %v", name, bottom,
				ok)
		}
		middle, ok := profile.Onto(mid(profile.source))
		if !ok || !near(middle, mid(profile.render), mapping.Tolerance) {
			test.Errorf("%s: the middle lands at %v, ok %v", name, middle,
				ok)
		}
		top, ok := profile.Onto(long * mapping.JustInside)
		if !ok || top >= renderLong || top <= middle {
			test.Errorf("%s: just under the top lands at %v, ok %v", name,
				top, ok)
		}
	}
}

// Onto and Back undo each other everywhere between, in every named range.
func TestOntoAndBack(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	for name, profile := range profiles(test) {
		for _, fraction := range mapping.Fractions {
			nm := along(profile.source, fraction)
			mapped, ok := profile.Onto(nm)
			back, backOK := profile.Back(mapped)
			if !ok || !backOK || !near(back, nm, mapping.Tolerance) {
				test.Errorf("%s: %g nm went to %g and came back %g",
					name, nm, mapped, back)
			}
		}
	}
}

// wideProfile is a profile wider than it is shown, and how near a round
// trip through it must come back.
type wideProfile struct {
	built
	tolerance float64
}

// The property: in a wide profile, a wavelength maps to a wavelength in
// the render range, a longer wavelength to a longer one, and back again.
func TestOntoKeepsOrderAndComesBack(test *testing.T) {
	tolerance := pickedNumbers(test).Mapping.Tolerance
	rapid.Check(test, wideProfile{wide(test), tolerance}.orderKept)
}

// orderKept is one draw of the property above. The wavelengths are drawn
// from the whole source range, its long end's last float included.
func (shown wideProfile) orderKept(draw *rapid.T) {
	short, long := shown.source.Edges()
	top := math.Nextafter(long, short)
	shorter := rapid.Float64Range(short, top).Draw(draw, "shorter")
	longer := rapid.Float64Range(shorter, top).Draw(draw, "longer")
	low, lowOK := shown.Onto(shorter)
	high, highOK := shown.Onto(longer)
	if !lowOK || !highOK || low > high {
		draw.Fatalf("%g and %g nm went to %g and %g", shorter, longer, low,
			high)
	}
	if back, ok := shown.Back(low); !ok ||
		!near(back, shorter, shown.tolerance) {
		draw.Fatalf("%g nm came back %g, ok %v", shorter, back, ok)
	}
}

// The source's top edge, which a draw of the property above seldom lands
// on: the last float below the source's long end maps to a wavelength
// the render holds, and comes back.
func TestTheTopOfTheSourceStaysInsideTheRender(test *testing.T) {
	shown := wide(test)
	short, long := shown.source.Edges()
	top := math.Nextafter(long, short)
	mapped, ok := shown.Onto(top)
	if !ok || !shown.render.Holds(mapped) {
		test.Fatalf("%g nm went to %g, ok %v", top, mapped, ok)
	}
	if back, ok := shown.Back(mapped); !ok ||
		!near(back, top, pickedNumbers(test).Mapping.Tolerance) {
		test.Errorf("%g nm came back %g, ok %v", top, back, ok)
	}
}

// The anti-test, as a property: a wavelength outside the source range is
// refused, whichever side it is off. It is reported, not mapped to a
// number that looks like an answer.
func TestAWavelengthOutsideTheProfileIsRefused(test *testing.T) {
	tolerance := pickedNumbers(test).Mapping.Tolerance
	rapid.Check(test, wideProfile{wide(test), tolerance}.refused)
}

// refused is one draw of the property above.
func (shown wideProfile) refused(draw *rapid.T) {
	short, long := shown.source.Edges()
	renderShort, renderLong := shown.render.Edges()
	below := rapid.Float64Range(math.SmallestNonzeroFloat64,
		math.Nextafter(short, none)).Draw(draw, "below")
	above := rapid.Float64Range(long, math.MaxFloat64).Draw(draw, "above")
	for _, nm := range []float64{below, above} {
		if mapped, ok := shown.Onto(nm); ok || mapped != none {
			draw.Fatalf("%g nm was mapped to %g", nm, mapped)
		}
	}
	belowRender := rapid.Float64Range(math.SmallestNonzeroFloat64,
		math.Nextafter(renderShort, none)).Draw(draw, "below render")
	aboveRender := rapid.Float64Range(renderLong, math.MaxFloat64).
		Draw(draw, "above render")
	for _, nm := range []float64{belowRender, aboveRender} {
		if source, ok := shown.Back(nm); ok || source != none {
			draw.Fatalf("%g nm was mapped back to %g", nm, source)
		}
	}
}

// The anti-test: a profile of a range that is no range is refused, and
// gives no profile. They are:
//   - the source or the render range running backwards
//   - the render range being one wavelength
//   - the source starting at no wavelength, or the render below it
func TestBrokenProfilesAreRefused(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	for _, broken := range []written{mapping.Flat, mapping.BackRend,
		mapping.BackSrc, mapping.ZeroSrc, mapping.NegRend} {
		source := radiation.Range{}
		if made, err := radiation.NewRange(broken.Source.Lo,
			broken.Source.Hi); err == nil {
			source = *made
		}
		render := radiation.Range{}
		if made, err := radiation.NewRange(broken.Render.Lo,
			broken.Render.Hi); err == nil {
			render = *made
		}
		profile, err := radiation.NewProfile(broken.Name, source, render)
		if err == nil || profile != nil {
			test.Errorf("%s made %v, %v", broken.Name, profile, err)
		}
	}
}

// The anti-test: the zero Profile maps nothing, either way.
func TestTheZeroProfileMapsNothing(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	var zero radiation.Profile
	if _, ok := zero.Onto(mapping.InsideNM); ok {
		test.Errorf("the zero profile maps %v nm", mapping.InsideNM)
	}
	if _, ok := zero.Back(mapping.RenderNM); ok {
		test.Errorf("the zero profile maps %v nm back", mapping.RenderNM)
	}
}

// Sound profiles map what they hold, and back.
func TestSoundProfilesMapWhatTheyHold(test *testing.T) {
	mapping := pickedNumbers(test).Mapping
	sound := profileOf(test, mapping.Sound.Name,
		span(test, mapping.Sound.Source), span(test, mapping.Sound.Render))
	if !sound.Holds(mapping.InsideNM) {
		test.Fatalf("%s does not hold %v nm", sound.name, mapping.InsideNM)
	}
	mapped, ok := sound.Onto(mapping.InsideNM)
	if !ok || !sound.render.Holds(mapped) {
		test.Errorf("%s maps %v nm to %v, ok %v", sound.name,
			mapping.InsideNM, mapped, ok)
	}
	if _, ok := sound.Back(mapping.RenderNM); !ok {
		test.Errorf("%s maps %v nm back to nothing", sound.name,
			mapping.RenderNM)
	}
}

// A profile says its name, its range and what it is mapped onto, for a
// person to read.
func TestAProfileSaysWhatItIs(test *testing.T) {
	heard := map[string]string{}
	for name, profile := range profiles(test) {
		said := profile.String()
		if other, again := heard[said]; again {
			test.Errorf("%s and %s say the same: %q", name, other, said)
		}
		heard[said] = name
		for _, part := range []string{profile.name, profile.source.String(),
			"shown as " + profile.render.String()} {
			if !strings.Contains(said, part) {
				test.Errorf("%q does not say %q", said, part)
			}
		}
	}
}
