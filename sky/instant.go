package sky

import (
	"errors"
	"math"
)

// secondsADay is a day of tai, tt and tdb, and of utc without a leap
// second. Defined by the si: a day of 86400 seconds.
const secondsADay = 86400.0

// mjdZero is the julian date at the start of modified julian day
// nought: 1858 november 17, 0h. Defined by the modified julian date.
const mjdZero = 2400000.5

// errDate is a date or a julian date that is no moment.
var errDate = errors.New("celestial: no such date")

// ErrDate is a date or a julian date that is no moment: no such day in
// the calendar, a time past the day's end, or a number that is none.
func ErrDate() error { return errDate }

// Instant is a moment on a scale, held in two parts so none of its
// resolution is lost: a whole modified julian day, and how far through
// that day it is, from nought up to one. On a utc day ending in a leap
// second the day is 86401 s, and the fraction is of that.
type Instant struct {
	scale Scale
	day   int64
	part  float64
}

// FromJulian is the instant at a julian date given in two parts whose
// sum is the date; one part may carry it all and the other nought. It
// refuses no scale, a date that is no finite number, and one so far off
// its days cannot be counted exactly, and then gives no instant.
func FromJulian(whole, fraction float64, scale Scale) (*Instant, error) {
	sum := whole + fraction
	days := math.Floor(whole - mjdZero)
	if !scale.known() || math.IsNaN(sum) || math.IsInf(sum, eitherSign) ||
		!countable(days) {
		return nil, errDate
	}
	rest := (whole - mjdZero - days) + fraction
	moment := settled(scale, int64(days), rest)
	return &moment, nil
}

// Julian is the instant's julian date in two parts: the julian date of
// the start of its day, and the fraction of the day since.
func (moment Instant) Julian() (whole, fraction float64) {
	return float64(moment.day) + mjdZero, moment.part
}

// Scale is the clock the instant is on.
func (moment Instant) Scale() Scale { return moment.scale }

// settled is a day and a fraction with the whole days moved out of the
// fraction, on a scale whose days are all one length.
func settled(scale Scale, day int64, part float64) Instant {
	whole := math.Floor(part)
	day += int64(whole)
	return Instant{scale: scale, day: day, part: part - whole}
}

// after is the instant so many seconds later, on a scale whose days are
// all one length.
func (moment Instant) after(seconds float64) Instant {
	return settled(moment.scale, moment.day,
		moment.part+seconds/secondsADay)
}
