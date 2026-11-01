package sky_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// Greenwich mean sidereal time on 1987 april 10 at 19h21m ut is
// 8h34m57.0896s: meeus, astronomical algorithms, example 12.b, p. 89,
// read on 2026-10-31 as sonia keys transcribes it in her go
// implementation of the book (github.com/soniakeys/meeus, mit licence,
// commit bfbd9ac). Cited, not copied: none of her code is here.
func TestMeanSiderealTimeIsMeeussExample(test *testing.T) {
	numbers := clockNumbers(test)
	turned, err := at(test, numbers.Sidereal.At, sky.UT1()).MeanSidereal()
	if err != nil {
		test.Fatal(err)
	}
	want := degreesOf(numbers, numbers.Sidereal.Time)
	off := math.Abs(turned.InDegrees()-want) *
		numbers.Sidereal.SecondsADegree
	if off > numbers.Sidereal.ToleranceSeconds {
		test.Errorf("sidereal time is %.7f deg; meeus says %.7f",
			turned.InDegrees(), want)
	}
}

// The anti-test: sidereal time is the earth's turning, so it is asked of
// ut1 alone; ut1 is utc and a dut1 within 0.9 s, and no further.
func TestSiderealTimeIsAskedOfUT1(test *testing.T) {
	numbers := clockNumbers(test)
	civil := at(test, numbers.Sidereal.At, sky.UTC())
	if angle, err := civil.MeanSidereal(); err == nil || angle != nil {
		test.Errorf("utc gives a sidereal time: %v, %v", angle, err)
	}
	turning, err := civil.UT1(measure.Seconds(numbers.DUT1.Seconds))
	if err != nil {
		test.Fatal(err)
	}
	wall := at(test, numbers.Sidereal.At, sky.UT1())
	if got := apart(*turning, wall); !(math.Abs(got-numbers.DUT1.Seconds) <=
		numbers.DUT1.Tolerance) {
		test.Errorf("ut1 with dut1 %v s is %.6f s from utc",
			numbers.DUT1.Seconds, got)
	}
	if moment, err := civil.UT1(measure.Seconds(numbers.DUT1.TooMuch)); err == nil || moment != nil {
		test.Errorf("a dut1 of a second is taken: %v", err)
	}
	if moment, err := turning.UT1(measure.Seconds(numbers.DUT1.Seconds)); err == nil || moment != nil {
		test.Errorf("an instant already on ut1 is taken: %v", err)
	}
}

// washington is the us naval observatory, by meeus's example 13.b.
func washington(test *testing.T) sky.Place {
	test.Helper()
	numbers := clockNumbers(test)
	north := angleOf(numbers, numbers.Washington.Latitude)
	east := angleOf(numbers, numbers.Washington.Longitude)
	up := measure.Metres(numbers.WashingtonUp)
	there, err := sky.OnEarth(north, east, up)
	if err != nil {
		test.Fatal(err)
	}
	if there.Latitude() != north || there.Longitude() != east ||
		there.Height() != up {
		test.Fatalf("washington is %v, %v, %v", there.Latitude(),
			there.Longitude(), there.Height())
	}
	return *there
}

// Venus over washington on 1987 april 10 at 19h21m ut stands at azimuth
// 248.034 from the north and altitude 15.125: meeus's example 13.b, p.
// 95, read on 2026-10-31 as sonia keys transcribes it (soniakeys/meeus,
// mit licence, commit bfbd9ac; cited, not copied), +68.034 from the south
// and +15.125, to three decimals and with apparent sidereal time.
//
// Apparent is 0.23 s off the mean; that moves these by under 0.001 deg,
// and nutation is not held yet, so they are held to 0.002 deg.
func TestVenusOverWashingtonIsMeeussExample(test *testing.T) {
	numbers := clockNumbers(test)
	turned, err := at(test, numbers.Sidereal.At, sky.UT1()).MeanSidereal()
	if err != nil {
		test.Fatal(err)
	}
	venus := toward(test, degreesOf(numbers, numbers.Venus.Ascension),
		degreesOf(numbers, numbers.Venus.Declination))
	seen := washington(test).Local(venus, *turned)
	want := numbers.Venus
	if !(math.Abs(seen.Azimuth().InDegrees()-want.Azimuth) <= want.Tolerance) ||
		!(math.Abs(seen.Altitude().InDegrees()-want.Altitude) <=
			want.Tolerance) {
		test.Errorf("venus is at azimuth %.4f, altitude %.4f; meeus says "+
			"%v and %v", seen.Azimuth().InDegrees(),
			seen.Altitude().InDegrees(), want.Azimuth, want.Altitude)
	}
}

// The celestial pole stands due north at the place's own latitude,
// whatever the time: the geometry of a sphere turning on its axis.
func TestThePoleStandsAtTheLatitude(test *testing.T) {
	numbers := clockNumbers(test)
	pole := toward(test, nought, numbers.PoleDeclination)
	here := washington(test)
	latitude := here.Latitude().InDegrees()
	for _, hours := range numbers.PoleHours {
		turned := measure.Degrees(hours * numbers.Sexagesimal.DegreesAnHour)
		seen := here.Local(pole, turned)
		if !(math.Abs(seen.Altitude().InDegrees()-latitude) <=
			numbers.PoleTolerance) {
			test.Errorf("at %g h the pole is %.9f deg up", hours,
				seen.Altitude().InDegrees())
		}
	}
}

// The anti-test: a place off the earth, a height that is no number,
// below the earth's centre or above the highest a place stands, and a
// direction off the sky are refused, and give nothing.
func TestWhatIsNoPlaceIsRefused(test *testing.T) {
	numbers := clockNumbers(test)
	for _, place := range numbers.NotPlaces {
		there, err := sky.OnEarth(measure.Degrees(place.Latitude),
			measure.Degrees(place.Longitude), measure.Metres(nought))
		if err == nil || there != nil {
			test.Errorf("%+v is a place: %v", place, err)
		}
	}
	for _, metres := range []float64{math.NaN(), numbers.BelowTheCentre,
		numbers.AboveTheHighest} {
		there, err := sky.OnEarth(measure.Degrees(nought),
			measure.Degrees(nought), measure.Metres(metres))
		if err == nil || there != nil {
			test.Errorf("a height of %v m is a place: %v", metres, err)
		}
	}
	for _, north := range numbers.NotDeclinations {
		way, err := sky.NewEquatorial(measure.Degrees(nought),
			measure.Degrees(north))
		if err == nil || way != nil {
			test.Errorf("declination %v is a direction: %v", north, err)
		}
	}
	for _, east := range numbers.NotAscensions {
		way, err := sky.NewEquatorial(measure.Degrees(east),
			measure.Degrees(nought))
		if err == nil || way != nil {
			test.Errorf("ascension %v is a direction: %v", east, err)
		}
	}
}
