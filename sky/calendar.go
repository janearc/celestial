package sky

import (
	"fmt"
	"time"
)

// The Gregorian reform: the day after 1582 October 4, the last Julian day, was
// October 15, the first Gregorian one.
const (
	reformYear   = 1582 // defined by the reform
	reformMonth  = 10   // defined by the reform: october
	skippedFrom  = 4    // defined by the reform: the last julian day
	skippedUntil = 15   // defined by the reform: the first gregorian day
	// gregorianFrom is October 15 as a modified Julian day, defined by the
	// reform.
	gregorianFrom = -100840
)

// The months and the leap years.
const (
	firstMonth   = 1  // defined by the calendar: january
	monthsAYear  = 12 // defined by the calendar
	february     = 2  // defined by the calendar
	leapFebruary = 29 // defined by the calendar: february in a leap year
	// nonLeapYear is any year with no February 29, which daysIn asks Go's
	// time package about; defined by the calendar.
	nonLeapYear     = 2001
	leapEvery       = 4   // defined by both calendars: a year in four
	centuryYears    = 100 // defined by the gregorian: no leap century,
	fourCenturies   = 400 // defined by the gregorian: but every fourth
	nothingLeftOver = 0   // defined by division: a year divides evenly
)

// The hours, minutes and seconds of a day.
const (
	firstHour      = 0      // defined by the clock
	lastHour       = 23     // defined by the clock
	lastMinute     = 59     // defined by the clock
	noTime         = 0      // defined by the clock: an hour's start
	secondsAMinute = 60.0   // defined by the clock
	secondsAnHour  = 3600.0 // defined by the clock
)

// Calendar is the instant at a date and a time of day on a scale.
//
// Dates from 1582 October 15 are Gregorian, and dates before it are julian.
// Years are counted as astronomers count them: year zero is the year before
// year one.
//
// A UTC day that ends in a leap second has a 60th second in its last
// minute, and no other day does. Returns an error for:
//   - a scale sky does not know
//   - a date neither calendar has, or one of the ten days of 1582
//     october the reform skipped
//   - a time of day the day does not have
func Calendar(year, month, day, hour, minute int, second float64,
	scale Scale) (*Instant, error) {
	if !scale.known() {
		return nil, fmt.Errorf("sky: a date on %v", scale)
	}
	if err := dated(year, month, day); err != nil {
		return nil, err
	}
	start := dayNumber(year, month, day)
	length := dayLength(scale, start)
	most := secondsAMinute
	if hour == lastHour && minute == lastMinute {
		most = length - (secondsADay - secondsAMinute)
	}
	if hour < firstHour || hour > lastHour || minute < noTime ||
		minute > lastMinute || !(second >= noTime && second < most) {
		return nil, fmt.Errorf("sky: %02d:%02d:%g is no time of that "+
			"day", hour, minute, second)
	}
	seconds := float64(hour)*secondsAnHour +
		float64(minute)*secondsAMinute + second
	return &Instant{scale: scale, day: start, part: seconds / length}, nil
}

// dated returns nil for a year, month and day that make a date, and for
// any that do not, an error saying why.
func dated(year, month, day int) error {
	if month < firstMonth || month > monthsAYear || day < firstDay ||
		day > daysIn(year, month) {
		return fmt.Errorf("sky: %d-%d-%d is no date", year, month, day)
	}
	if year == reformYear && month == reformMonth && day > skippedFrom &&
		day < skippedUntil {
		return fmt.Errorf("sky: 1582 october %d was skipped", day)
	}
	return nil
}

// daysIn is how many days a month has, in the calendar its year uses.
//
// The two calendars differ only in which years have a February 29, so daysIn
// decides that, and asks Go's time package for every other month in a year
// without one.
func daysIn(year, month int) int {
	if month == february && leapYear(year) {
		return leapFebruary
	}
	next := time.Month(month + firstMonth)
	// day zero of the next month, which time reads as the last of this
	lastOfThisMonth := firstDay - nextDay
	return time.Date(nonLeapYear, next, lastOfThisMonth, noTime, noTime,
		noTime, noTime, time.UTC).Day()
}

// leapYear is whether a year has a February 29.
//
// To 1582, in the Julian calendar, every fourth year is a leap year. After it,
// in the Gregorian, every fourth year is too, except a century, unless it is a
// fourth century: 1900 is not, and 2000 is.
func leapYear(year int) bool {
	fourth := ((year % leapEvery) + leapEvery) % leapEvery
	if year <= reformYear {
		return fourth == nothingLeftOver
	}
	return fourth == nothingLeftOver && (year%centuryYears !=
		nothingLeftOver || year%fourCenturies == nothingLeftOver)
}
