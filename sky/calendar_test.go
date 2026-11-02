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

// Every date Meeus gives is the Julian date he gives, to the thousandth of a
// day his figures are given to: his examples 7.a and 7.b and the dates of his
// table 7.a. Before 1582 October 15 the calendar is the Julian one, as Meeus
// has it.
//
// Each was read on 2026-10-31 as sonia keys transcribes it, with Meeus's pages
// 61 and 62, in her go implementation of the book (github.com/soniakeys/meeus,
// mit licence, commit bfbd9ac; cited, not copied: none of her code is here).
//
// Three, and the reform, were read again in his earlier astronomical formulae
// for calculators (1980).
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

// A Julian date goes back to the calendar date it came from: Meeus's example
// 7.c, p. 64, as sonia keys transcribes it (cited above), and the rest of his
// dates.
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

// Every calendar's leap year has a february 29: a gregorian fourth year,
// the first gregorian leap year, a gregorian fourth century, and a
// julian leap year before year one.
func TestEveryLeapYearHasAFebruary29(test *testing.T) {
	for _, leap := range clockNumbers(test).LeapDays {
		at(test, leap, sky.TT())
	}
}

// A date goes to its instant and back to itself: the last julian day and
// the first gregorian one, a leap day and the day after it, and the end
// of a february with no leap day.
func TestADateGoesToItsInstantAndBack(test *testing.T) {
	for _, date := range clockNumbers(test).RoundTrips {
		year, month, day, _, _, _ := at(test, date, sky.TT()).Calendar()
		if year != date.Year || month != date.Month || day != date.Day {
			test.Errorf("%d-%d-%d came back %d-%d-%d", date.Year,
				date.Month, date.Day, year, month, day)
		}
	}
}

// The anti-test: dates that are no date are refused and give no instant: the
// ten days the Gregorian calendar skipped, a thirteenth month, February the
// thirtieth, a time past the day, and no scale.
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
