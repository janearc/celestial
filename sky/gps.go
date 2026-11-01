package sky

import (
	"fmt"
	"math"
)

// gpsEpochDay is the modified julian day of the gps epoch, 1980 january
// 6, at whose start gps time reads nought. Defined by IS-GPS-200,
// section 3.3.4, read on 2026-10-31 in revision M: "midnight on the
// night of January 5, 1980/morning of January 6, 1980".
const gpsEpochDay = 44244

// FromGPS is the instant a gps time names: seconds of gps time since the
// gps epoch, 1980 january 6 0h utc. The LIGO and Virgo detectors, and
// the open science centre that publishes their data, give their times
// this way.
//
// FromGPS refuses a time that is not a finite number, or so far off its
// days cannot be counted exactly, and then gives no instant.
func FromGPS(seconds float64) (*Instant, error) {
	days := math.Floor(seconds / secondsADay)
	if !countable(days) {
		return nil, fmt.Errorf("sky: gps time %v s, past what can be "+
			"counted", seconds)
	}
	rest := seconds - days*secondsADay
	moment := settled(GPS(), gpsEpochDay+int64(days), rest/secondsADay)
	return &moment, nil
}

// mostDays is the most days an instant counts from its zero, either
// way: the largest whole number a float counts exactly, past which a day
// number is not one. Defined by float64.
const mostDays = 1 << 53

// countable is whether a number of days is a number, and few enough to
// count exactly. A NaN fails the comparison, and is not.
func countable(days float64) bool { return math.Abs(days) < mostDays }

// GPSSeconds is the instant as seconds of gps time since the gps epoch,
// FromGPS undone. returns nil and error for what On refuses: an instant
// on ut1, or on utc where its leap seconds are not known.
func (moment Instant) GPSSeconds() (*float64, error) {
	on, err := moment.On(GPS())
	if err != nil {
		return nil, err
	}
	toTheDay := float64(on.day-gpsEpochDay) * secondsADay
	intoTheDay := on.part * secondsADay
	seconds := toTheDay + intoTheDay
	return &seconds, nil
}
