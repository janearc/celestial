package sky

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// leapText is tai less utc since 1972, as leapseconds.json holds it.
//
//go:embed leapseconds.json
var leapText []byte

// leap is the utc month tai less utc became so many seconds.
type leap struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Ahead float64 `json:"ahead"`
}

// leapTable is leapseconds.json read: every leap, and the first utc day
// the table cannot speak for.
type leapTable struct {
	Leaps []leap `json:"leaps"`
	Until struct {
		Year  int `json:"year"`
		Month int `json:"month"`
		Day   int `json:"day"`
	} `json:"until"`
}

// readLeaps reads leapseconds.json, or gives none and why.
func readLeaps() (*leapTable, error) {
	var table leapTable
	if err := json.Unmarshal(leapText, &table); err != nil {
		return nil, err
	}
	return &table, nil
}

// leaps is the table, read once.
var leaps = sync.OnceValues(readLeaps)

// The day utc took whole seconds, and the first day of a month, each
// defined.
const (
	utcBeganYear  = 1972 // defined by the iers: utc's whole seconds began
	utcBeganMonth = 1    // defined by the iers: in january
	firstDay      = 1    // defined by the calendar: a month's first day
	noneAhead     = 0.0  // defined: before the table, nothing is known
	nextDay       = 1    // defined by counting: the day after
)

// aheadOn is tai less utc in seconds on a utc day, or why it is not
// known: utc before 1972 had seconds of other lengths, and the table
// cannot speak past its last bulletin.
func aheadOn(mjd int64) (float64, error) {
	table, err := leaps()
	if err != nil {
		return noneAhead, err
	}
	until := table.Until
	if mjd < dayNumber(utcBeganYear, utcBeganMonth, firstDay) ||
		mjd >= dayNumber(until.Year, until.Month, until.Day) {
		year, month, day := dateOf(mjd)
		return noneAhead, fmt.Errorf("%w: no leap seconds known for "+
			"utc on %d-%02d-%02d", errScale, year, month, day)
	}
	ahead := noneAhead
	for _, one := range table.Leaps {
		if dayNumber(one.Year, one.Month, firstDay) <= mjd {
			ahead = one.Ahead
		}
	}
	return ahead, nil
}

// dayLength is how many seconds a day of a scale has: 86401 for a utc
// day that ended in a leap second, and 86400 for every other day.
func dayLength(scale Scale, mjd int64) float64 {
	if scale != UTC() {
		return secondsADay
	}
	today, errToday := aheadOn(mjd)
	tomorrow, errTomorrow := aheadOn(mjd + nextDay)
	if errToday != nil || errTomorrow != nil {
		return secondsADay
	}
	return secondsADay + tomorrow - today
}
