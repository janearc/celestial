package sky_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/sky"
)

// Gps time is tai less 19 s at every instant: the same wall reading on
// gps is 19 s after it on tai. IS-GPS-200 (revision M, section 3.3.4)
// makes gps time continuous, with no leap seconds, but never names tai;
// the 19 s is the BIPM's, read on 2026-10-31 in its Circular T 22 (1989
// december 1): "TAI-GPS TIME = 19 seconds + Co", Co a small measured
// correction.
func TestGPSIsTAILessNineteenSeconds(test *testing.T) {
	numbers := clockNumbers(test)
	wall := at(test, numbers.GPS.Wall, sky.GPS())
	atomic := on(test, wall, sky.TAI())
	same := at(test, numbers.GPS.Wall, sky.TAI())
	if got := apart(atomic, same); !(math.Abs(got-numbers.GPS.BehindTAI) <=
		numbers.SecondsTolerance) {
		test.Errorf("gps is %.6f s behind tai, not %v", -got,
			numbers.GPS.BehindTAI)
	}
	back := on(test, atomic, sky.GPS())
	if got := apart(back, wall); !(math.Abs(got) <= numbers.SecondsTolerance) {
		test.Errorf("tai back to gps is %.6f s off", got)
	}
}

// Gps time nought is the start of 1980 january 6 on utc, by the
// definition of the gps epoch (IS-GPS-200).
func TestGPSNoughtIsTheEpoch(test *testing.T) {
	numbers := clockNumbers(test)
	epoch, err := sky.FromGPS(nought)
	if err != nil {
		test.Fatal(err)
	}
	civil := on(test, *epoch, sky.UTC())
	want := at(test, numbers.GPS.Epoch, sky.UTC())
	if got := apart(civil, want); !(math.Abs(got) <= numbers.SecondsTolerance) {
		test.Errorf("gps nought is %.6f s from 1980-01-06 0h utc", got)
	}
}

// GW150914 is at gps 1126259462.4, as the open science centre's event
// record gives it. LIGO's discovery paper (Abbott et al. 2016, PRL 116,
// 061102) gives 2015 september 14, 09:50:45 utc. Gps less utc was 17 s
// then (tai less utc 36 s from 2015 july 1, less the 19 s gps is behind
// tai), so the tenth of a second is 09:50:45.4.
func TestGW150914IsAtItsPublishedTime(test *testing.T) {
	numbers := clockNumbers(test)
	event, err := sky.FromGPS(numbers.GPS.GW150914)
	if err != nil {
		test.Fatal(err)
	}
	civil := on(test, *event, sky.UTC())
	want := at(test, numbers.GPS.GW150914UTC, sky.UTC())
	if got := apart(civil, want); !(math.Abs(got) <= numbers.GPS.GW150914Tol) {
		test.Errorf("GW150914 is %.6f s from 09:50:45.4 utc", got)
	}
}

// The property: gps seconds to an instant and back is the same number,
// to well inside a microsecond, from 1950 to 2075.
func TestGPSSecondsComeBack(test *testing.T) {
	rapid.Check(test, gpsTrip{numbers: clockNumbers(test)}.check)
}

// gpsTrip draws gps times and goes there and back.
type gpsTrip struct{ numbers clocked }

// check is one draw of the property above.
func (trip gpsTrip) check(test *rapid.T) {
	drawn := trip.numbers.GPS.Drawn
	seconds := rapid.Float64Range(drawn.Least, drawn.Most).
		Draw(test, "seconds")
	moment, err := sky.FromGPS(seconds)
	if err != nil {
		test.Fatal(err)
	}
	back, err := moment.GPSSeconds()
	if err != nil {
		test.Fatalf("%v came back with %v", seconds, err)
	}
	if !(math.Abs(*back-seconds) <= trip.numbers.SecondsTolerance) {
		test.Fatalf("%v came back %v", seconds, *back)
	}
}

// The anti-test: a gps time or julian date that is no number, or so far
// off its days cannot be counted, is refused and gives no instant; and
// an instant that cannot reach gps, one on ut1, gives no gps seconds.
func TestNoGPSTimeIsRefused(test *testing.T) {
	numbers := clockNumbers(test)
	nonsense := []float64{math.NaN(), math.Inf(upward), math.Inf(-upward)}
	for _, seconds := range append(numbers.GPS.NotSeconds, nonsense...) {
		if moment, err := sky.FromGPS(seconds); err == nil ||
			moment != nil {
			test.Errorf("gps time %v gave %v", seconds, err)
		}
	}
	for _, julian := range append(numbers.NotJulian, nonsense...) {
		if moment, err := sky.FromJulian(julian, nought, sky.TT()); err == nil || moment != nil {
			test.Errorf("julian date %v gave %v", julian, err)
		}
		if moment, err := sky.FromJulian(nought, julian, sky.TT()); err == nil || moment != nil {
			test.Errorf("a julian fraction of %v gave %v", julian, err)
		}
	}
	turning := at(test, numbers.UT1Day, sky.UT1())
	if seconds, err := turning.GPSSeconds(); err == nil || seconds != nil {
		test.Errorf("an instant on ut1 gave gps seconds: %v", err)
	}
}
