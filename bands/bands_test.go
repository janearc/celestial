package bands_test

import (
	"testing"

	"github.com/janearc/celestial/bands"
)

// Every PerOctave bands the wavelength doubles, and a band's end is the
// next one's start.
func TestAnOctaveIsPerOctaveBands(test *testing.T) {
	for _, nm := range pickedNumbers(test).OctaveNM {
		doubled := twice * nm
		if bands.Of(doubled) != bands.Of(nm)+bands.PerOctave() {
			test.Errorf("%v nm is band %d and %v nm is band %d",
				nm, bands.Of(nm), doubled, bands.Of(doubled))
		}
		band := bands.Of(nm)
		lo, hi := band.Edges()
		if nm < lo || nm >= hi {
			test.Errorf("%v nm is in band %d, which is %v to %v",
				nm, band, lo, hi)
		}
		if next, _ := (band + neighbour).Edges(); next != hi {
			test.Errorf("band %d ends at %v, band %d starts at %v",
				band, hi, band+neighbour, next)
		}
	}
}

// The anti-test: a wavelength is in one band and no other. The band below
// ends at or before it, and the band above starts after it.
func TestAWavelengthIsInOneBandOnly(test *testing.T) {
	for _, nm := range pickedNumbers(test).OctaveNM {
		band := bands.Of(nm)
		if _, below := (band - neighbour).Edges(); nm < below {
			test.Errorf("%v nm is inside band %d too", nm, band-neighbour)
		}
		if above, _ := (band + neighbour).Edges(); nm >= above {
			test.Errorf("%v nm is inside band %d too", nm, band+neighbour)
		}
		if bands.Of(nm) == bands.Of(twice*nm) {
			test.Errorf("%v and %v nm are in one band", nm, twice*nm)
		}
	}
}

// Visible light is the bands the Band comment names at 24 to an octave.
func TestVisibleLightIsTheBandsTheCommentNames(test *testing.T) {
	visible := pickedNumbers(test).Visible
	short, long := bands.Of(visible.Short), bands.Of(visible.Long)
	if int(short) != visible.First || int(long) != visible.Last {
		test.Errorf("visible light is bands %d to %d, not %d to %d",
			short, long, visible.First, visible.Last)
	}
}

// The test and the anti-test: visible light is twenty-six bands wide, as
// the libtheme measurements say, and the bands beside the visible ones do
// not reach into it.
func TestVisibleLightIsTwentySixBandsWide(test *testing.T) {
	visible := pickedNumbers(test).Visible
	short, long := bands.Of(visible.Short), bands.Of(visible.Long)
	if width := int(long-short) + int(neighbour); width != visible.Width {
		test.Errorf("visible light is %d bands wide, not %d", width, visible.Width)
	}
	_, belowHi := (short - neighbour).Edges()
	aboveLo, _ := (long + neighbour).Edges()
	if belowHi > visible.Short || aboveLo <= visible.Long {
		test.Errorf("the bands beside the visible ones reach into it")
	}
}

// Redshift by one step moves every band's light exactly one band redder.
func TestShiftByOneMovesEveryBandOne(test *testing.T) {
	light := bands.Split(pickedNumbers(test).Spectrum)
	shifted := light.Shift(int(neighbour))
	first, last := light.Span()
	if from, to := shifted.Span(); from != first+neighbour || to != last+neighbour {
		test.Errorf("span %d to %d shifted to %d to %d", first, last, from, to)
	}
	for band := first; band <= last; band++ {
		if light.Power(band) != shifted.Power(band+neighbour) {
			test.Errorf("band %d had %v, band %d has %v", band,
				light.Power(band), band+neighbour, shifted.Power(band+neighbour))
		}
	}
}
