package sky

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// leapText is leapseconds.json: tai less utc since 1972, by the month
// each change took effect.
//
//go:embed leapseconds.json
var leapText []byte

// leap is one change in tai less utc: the utc month it took effect, and
// how many seconds tai was ahead from then on.
type leap struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Ahead float64 `json:"ahead"`
}

// leapTable is leapseconds.json read: every change, and the first utc
// day past its last bulletin, which the table cannot speak for.
type leapTable struct {
	Leaps []leap `json:"leaps"`
	Until struct {
		Year  int `json:"year"`
		Month int `json:"month"`
		Day   int `json:"day"`
	} `json:"until"`
}

// readLeaps is the table, or nil and the error that stopped it.
func readLeaps() (*leapTable, error) {
	var table leapTable
	if err := json.Unmarshal(leapText, &table); err != nil {
		return nil, err
	}
	return &table, nil
}

// leaps is the table, read when sky loads; leapsErr is why it could not
// be, which every instant that needs it is refused with.
var leaps, leapsErr = readLeaps()

// The day utc began counting whole si seconds.
const (
	utcBeganYear  = 1972 // defined by the iers: utc's whole seconds began
	utcBeganMonth = 1    // defined by the iers: in january
)

// The first day of a month, and the day after another.
const (
	firstDay = 1 // defined by the calendar: a month's first day
	nextDay  = 1 // defined by counting: the day after
)

// aheadOn is tai less utc in seconds on a utc day. returns nil and
// error for a day the table cannot speak for: before 1972, when utc's
// seconds were of other lengths, or past its last bulletin.
func aheadOn(mjd int64) (*float64, error) {
	if leapsErr != nil {
		return nil, leapsErr
	}
	began := dayNumber(utcBeganYear, utcBeganMonth, firstDay)
	until := dayNumber(leaps.Until.Year, leaps.Until.Month, leaps.Until.Day)
	if mjd < began || mjd >= until {
		year, month, day := dateOf(mjd)
		return nil, fmt.Errorf("sky: no leap seconds known for utc on "+
			"%d-%02d-%02d", year, month, day)
	}
	// from 1972 january on, some step has always taken effect
	var ahead float64
	for _, step := range leaps.Leaps {
		if dayNumber(step.Year, step.Month, firstDay) <= mjd {
			ahead = step.Ahead
		}
	}
	return &ahead, nil
}

// dayLength is how many seconds a day of a scale has: 86401 for a utc
// day that ended in a leap second, and 86400 for every other day,
// including a utc day the table cannot speak for.
func dayLength(scale Scale, mjd int64) float64 {
	if scale != UTC() {
		return secondsADay
	}
	today, errToday := aheadOn(mjd)
	tomorrow, errTomorrow := aheadOn(mjd + nextDay)
	if errToday != nil || errTomorrow != nil {
		return secondsADay
	}
	return secondsADay + *tomorrow - *today
}
