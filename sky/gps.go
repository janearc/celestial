package sky

import (
	"fmt"
	"math"
)

// gpsEpochDay is the modified Julian day of the GPS epoch, 1980 January 6, at
// whose start GPS time reads zero. Defined by IS-GPS-200, section 3.3.4, read
// on 2026-10-31 in revision M: "midnight on the night of January 5,
// 1980/morning of January 6, 1980".
const gpsEpochDay = 44244

// FromGPS converts seconds since 1980-01-06 00:00 UTC to an instant on the GPS
// scale used by LIGO, Virgo, and the open science centre. It rejects nonfinite
// values and dates whose whole days float64 cannot count exactly.
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

// mostDays is the largest whole day count representable exactly by float64 in
// either direction. Defined by IEEE 754's binary64: its significand is 53 bits,
// 52 stored and one implied, so every whole number to 2^53 is held exactly.
const mostDays = 1 << 53

// countable checks that a day count is finite and within exact integer range.
// NaN fails the comparison.
func countable(days float64) bool { return math.Abs(days) < mostDays }

// GPSSeconds returns seconds since the GPS epoch, reversing FromGPS. It reports
// conversion errors from On, including UT1 input and UTC dates outside the leap
// table’s coverage.
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
