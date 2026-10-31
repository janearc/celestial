package sky

import (
	"fmt"
	"time"
)

// The calendar's own numbers, each defined by it.
const (
	// gregorianFrom is 1582 october 15 as a modified julian day.
	// Defined by the reform.
	gregorianFrom = -100840
	// reformYear is defined by the gregorian reform.
	reformYear = 1582
	// reformMonth is defined by the gregorian reform: october.
	reformMonth = 10
	// skippedFrom is defined: october 4 was the last julian day.
	skippedFrom = 4
	// skippedUntil is defined: october 15 was the first gregorian one.
	skippedUntil = 15
	// february is defined by the calendar.
	february = 2
	// leapFebruary is defined: february's days in a leap year.
	leapFebruary = 29
	// monthsAYear is defined by the calendar.
	monthsAYear = 12
	// firstMonth is defined by the calendar.
	firstMonth = 1
	// firstHour is defined by the clock.
	firstHour = 0
	// lastHour is defined by the clock.
	lastHour = 23
	// lastMinute is defined by the clock.
	lastMinute = 59
	// secondsAMinute is defined by the clock.
	secondsAMinute = 60.0
	// secondsAnHour is defined by the clock.
	secondsAnHour = 3600.0
	// nonLeapYear is defined: a year with no february 29.
	nonLeapYear = 2001
	// leapEvery is defined by both calendars: a leap year in four.
	leapEvery = 4
	// centuryYears is defined by the gregorian calendar.
	centuryYears = 100
	// fourCenturies is defined by the gregorian calendar.
	fourCenturies = 400
	// nothingLeftOver is defined by division: an even split.
	nothingLeftOver = 0
	// noTime is defined by the clock: the start of an hour or minute.
	noTime = 0
)

// Calendar is the instant at a date and time on a scale. From 1582
// october 15 the date is gregorian, and before it julian; years are
// counted as astronomers do, with year nought before year one.
//
// A utc day that ended in a leap second has a second of 60 in its last
// minute; no other day does. Calendar refuses no scale, no such date and
// no such time, and then gives no instant.
func Calendar(year, month, day, hour, minute int, second float64,
	scale Scale) (*Instant, error) {
	if !scale.known() {
		return nil, fmt.Errorf("%w: %v", errDate, scale)
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
		minute > lastMinute || !(second >= noSeconds && second < most) {
		return nil, fmt.Errorf("%w: %02d:%02d:%g", errDate, hour,
			minute, second)
	}
	seconds := float64(hour)*secondsAnHour +
		float64(minute)*secondsAMinute + second
	return &Instant{scale: scale, day: start, part: seconds / length}, nil
}

// dated is why a year, month and day are no date, or nothing.
func dated(year, month, day int) error {
	if month < firstMonth || month > monthsAYear || day < firstDay ||
		day > daysIn(year, month) {
		return fmt.Errorf("%w: %d-%d-%d", errDate, year, month, day)
	}
	if year == reformYear && month == reformMonth && day > skippedFrom &&
		day < skippedUntil {
		return fmt.Errorf("%w: 1582 october %d was skipped", errDate,
			day)
	}
	return nil
}

// daysIn is how many days a month has, in the calendar of its year. Only
// february differs between the julian and gregorian calendars, so every
// other month is as the standard library's calendar has it.
func daysIn(year, month int) int {
	if month == february && leapYear(year) {
		return leapFebruary
	}
	next := time.Month(month + firstMonth)
	lastOfThisMonth := firstDay - nextDay
	return time.Date(nonLeapYear, next, lastOfThisMonth, noTime, noTime,
		noTime, noTime, time.UTC).Day()
}

// leapYear is whether a year has a february 29: every fourth in the
// julian calendar, and not centuries but every fourth one in the
// gregorian.
func leapYear(year int) bool {
	fourth := ((year % leapEvery) + leapEvery) % leapEvery
	if year <= reformYear {
		return fourth == nothingLeftOver
	}
	return fourth == nothingLeftOver && (year%centuryYears !=
		nothingLeftOver || year%fourCenturies == nothingLeftOver)
}
