package sky_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/sky"
)

// The clock's parts, each defined by it.
const (
	secondsAnHour  = 3600.0 // defined by the clock
	secondsAMinute = 60.0   // defined by the clock
)

// onDay is an instant on a calendar day, its fraction turned into the
// hours, minutes and seconds it is.
func onDay(test *testing.T, date dated) sky.Instant {
	test.Helper()
	whole := math.Floor(date.Day)
	seconds := (date.Day - whole) * secondsADay
	hour := math.Floor(seconds / secondsAnHour)
	minute := math.Floor((seconds - hour*secondsAnHour) / secondsAMinute)
	return at(test, timed{Year: date.Year, Month: date.Month,
		Day: int(whole), Hour: int(hour), Minute: int(minute),
		Second: seconds - hour*secondsAnHour - minute*secondsAMinute},
		sky.TT())
}

// Every date meeus gives is the julian date he gives, to the thousandth
// of a day his figures are given to: his examples 7.a and 7.b and the
// dates of his table 7.a. Before 1582 october 15 the calendar is the
// julian one, as meeus has it.
//
// Each was read on 2026-10-31 as sonia keys transcribes it, with
// meeus's pages 61 and 62, in her go implementation of the book
// (github.com/soniakeys/meeus, mit licence, commit bfbd9ac; cited, not
// copied: none of her code is here). Three, and the reform, were read
// again in his earlier astronomical formulae for calculators (1980).
func TestADateIsTheJulianDateMeeusGives(test *testing.T) {
	numbers := clockNumbers(test)
	for _, date := range numbers.MeeusDates {
		whole, fraction := onDay(test, date).Julian()
		if got := whole + fraction; !(math.Abs(got-date.Julian) <=
			numbers.DateTolerance) {
			test.Errorf("%d-%d-%g is julian %.6f; meeus says %.6f",
				date.Year, date.Month, date.Day, got, date.Julian)
		}
	}
}

// A julian date goes back to the calendar date it came from: meeus's
// example 7.c, p. 64, as sonia keys transcribes it (cited above), and
// the rest of his dates.
func TestAJulianDateGoesBackToItsDate(test *testing.T) {
	numbers := clockNumbers(test)
	for _, date := range numbers.MeeusDates {
		moment, err := sky.FromJulian(date.Julian, nought, sky.TT())
		if err != nil {
			test.Fatal(err)
		}
		year, month, day, hour, minute, second := moment.Calendar()
		days := float64(day) + (float64(hour)*secondsAnHour+
			float64(minute)*secondsAMinute+second)/secondsADay
		if year != date.Year || month != date.Month ||
			!(math.Abs(days-date.Day) <= numbers.DateTolerance) {
			test.Errorf("julian %.2f is %d-%d-%.6f; meeus says %d-%d-%g",
				date.Julian, year, month, days, date.Year, date.Month,
				date.Day)
		}
		if moment.Scale() != sky.TT() {
			test.Errorf("julian %.2f is on %v", date.Julian, moment.Scale())
		}
	}
}

// The anti-test: dates that are no date are refused and give no instant:
// the ten days the gregorian calendar skipped, a thirteenth month,
// february the thirtieth, a time past the day, and no scale.
func TestADateThatIsNoDateIsRefused(test *testing.T) {
	numbers := clockNumbers(test)
	for _, wrong := range numbers.NotDates {
		moment, err := sky.Calendar(wrong.Year, wrong.Month, wrong.Day,
			wrong.Hour, wrong.Minute, wrong.Second, sky.TT())
		if err == nil || moment != nil {
			test.Errorf("%+v is a date: %v", wrong, err)
		}
	}
	leap := numbers.LeapDay
	for _, wrong := range numbers.NotTimes {
		moment, err := sky.Calendar(leap.Year, leap.Month, leap.Day,
			wrong.Hour, wrong.Minute, wrong.Second, sky.TT())
		if err == nil || moment != nil {
			test.Errorf("%+v is a time: %v", wrong, err)
		}
	}
	at(test, leap, sky.TT())
	at(test, numbers.JulianLeapDay, sky.TT())
	var none sky.Scale
	if moment, err := sky.Calendar(leap.Year, leap.Month, leap.Day,
		leap.Hour, leap.Minute, leap.Second, none); err == nil ||
		moment != nil {
		test.Errorf("a date on no scale is taken: %v", err)
	}
	if moment, err := sky.FromJulian(nought, nought, none); err == nil ||
		moment != nil {
		test.Errorf("a julian date on no scale is taken: %v", err)
	}
	if name := none.String(); name != "no scale" {
		test.Errorf("the zero scale is called %q", name)
	}
}
