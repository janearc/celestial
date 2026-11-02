package bands_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// Light at every wavelength counts, those an eye never sees too: it has
// power in its bands.
func TestInfraredIsLight(test *testing.T) {
	spectrum := map[int]float64{}
	for _, nm := range pickedNumbers(test).InfraredNM {
		spectrum[nm] = unitPower
	}
	light := bands.Split(spectrum)
	for nm := range spectrum {
		band := bands.Of(float64(nm))
		if got := light.Power(band); got != unitPower {
			test.Errorf("%d nm carries %v", nm, got)
		}
	}
}

// The anti-test: power stays in the band it falls in, and is not spread to
// the bands beside it.
func TestPowerStaysInItsBand(test *testing.T) {
	numbers := pickedNumbers(test)
	light := bands.Split(map[int]float64{int(numbers.LineNM): numbers.LinePower})
	band := bands.Of(numbers.LineNM)
	if light.Power(band-neighbour) != none || light.Power(band+neighbour) != none {
		test.Errorf("power leaked out of band %d", band)
	}
	if light.Power(band) != numbers.LinePower {
		test.Errorf("band %d has %v, not %v", band, light.Power(band), numbers.LinePower)
	}
}

// Wavelengths in one band add up in it: light adds.
func TestSplitAddsWavelengthsInABand(test *testing.T) {
	numbers := pickedNumbers(test)
	same := numbers.SameBand
	if bands.Of(float64(same.Low)) != bands.Of(float64(same.High)) {
		test.Fatalf("%d and %d nm are expected to share a band", same.Low, same.High)
	}
	light := bands.Split(map[int]float64{same.Low: same.LowP, same.High: same.HighP})
	want := same.LowP + same.HighP
	band := bands.Of(float64(same.Low))
	got := light.Power(band)
	if !(math.Abs(got-want) <= numbers.Tolerance) {
		test.Errorf("band has %v, want the sum %v", got, want)
	}
}

// The anti-test: wavelengths in different bands do not add. Each band has
// its own line's power and none of the other's.
func TestSplitDoesNotAddAcrossBands(test *testing.T) {
	numbers := pickedNumbers(test)
	run := numbers.Run
	inside, outside := bands.Of(run.Inside), bands.Of(run.Outside)
	if inside == outside {
		test.Fatal("the two wavelengths are expected to be in different bands")
	}
	light := bands.Split(map[int]float64{
		int(run.Inside): numbers.LinePower, int(run.Outside): unitPower})
	if got := light.Power(inside); got != numbers.LinePower {
		test.Errorf("band %d has %v, want its own %v", inside, got,
			numbers.LinePower)
	}
	if got := light.Power(outside); got != unitPower {
		test.Errorf("band %d has %v, want its own %v", outside, got,
			unitPower)
	}
}

// The property: splitting a spectrum keeps all its power. The powers of
// every band add up to the powers of every wavelength.
func TestSplitKeepsAllThePower(test *testing.T) {
	rapid.Check(test, keepsPower)
}

// The anti-test: splitting puts no power where no wavelength was. A band
// inside the span with no wavelength in it is dark. The power is kept,
// as the property says, and it is kept in place.
func TestSplitLeavesEmptyBandsDark(test *testing.T) {
	spectrum := pickedNumbers(test).Spectrum
	given := map[bands.Band]bool{}
	for nm := range spectrum {
		given[bands.Of(float64(nm))] = true
	}
	light := bands.Split(spectrum)
	first, last := light.Span()
	dark := none
	for band := first; band <= last; band++ {
		if given[band] {
			continue
		}
		dark++
		if power := light.Power(band); power != none {
			test.Errorf("band %d, with no wavelength, has %v", band, power)
		}
	}
	if dark == none {
		test.Fatal("the picked spectrum is expected to leave bands dark")
	}
}

// keepsPower is one draw of the property above.
func keepsPower(test *rapid.T) {
	given := spectrum(test, "spectrum")
	var want, got float64
	for _, power := range given {
		want += power
	}
	light := bands.Split(given)
	first, last := light.Span()
	for band := first; band <= last; band++ {
		got += light.Power(band)
	}
	if !(math.Abs(got-want) <= want*relativeTolerance) {
		test.Fatalf("the bands hold %v, the spectrum %v", got, want)
	}
}
