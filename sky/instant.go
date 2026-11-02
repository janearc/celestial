package sky

import (
	"fmt"
	"math"
)

// secondsADay is a day of TAI, TT and TDB, and of UTC without a leap second.
// Defined by the SI: a day of 86400 seconds.
const secondsADay = 86400.0

// mjdZero is the Julian date at the start of modified Julian day zero: 1858
// November 17, 0h. Defined by the modified Julian date.
const mjdZero = 2400000.5

// Instant stores a time scale, whole modified Julian day, and fraction of that
// day. A UTC leap-second day has 86401 seconds; its fraction is measured
// against that longer day.
type Instant struct {
	scale Scale
	day   int64
	part  float64
}

// FromJulian constructs an instant from two parts whose sum is a Julian date.
// Either part may hold the whole date. It rejects unknown scales and parts
// outside float64’s exact whole-day counting range.
func FromJulian(whole, fraction float64, scale Scale) (*Instant, error) {
	days := math.Floor(whole - mjdZero)
	if !scale.known() || !countable(days) || !countable(fraction) {
		return nil, fmt.Errorf("sky: julian date %v + %v on %v, "+
			"past what can be counted, or on no scale", whole,
			fraction, scale)
	}
	rest := (whole - mjdZero - days) + fraction
	moment := settled(scale, int64(days), rest)
	return &moment, nil
}

// Julian returns two parts: the Julian date at the start of the day and the
// fraction elapsed since then.
func (moment Instant) Julian() (whole, fraction float64) {
	return float64(moment.day) + mjdZero, moment.part
}

// Scale returns the instant’s time scale.
func (moment Instant) Scale() Scale { return moment.scale }

// settled normalizes a day and fraction on a uniform-day scale, carrying whole
// days out of the fraction.
func settled(scale Scale, day int64, part float64) Instant {
	whole := math.Floor(part)
	day += int64(whole)
	return Instant{scale: scale, day: day, part: part - whole}
}

// after shifts an instant by seconds on a uniform-day scale.
func (moment Instant) after(seconds float64) Instant {
	return settled(moment.scale, moment.day,
		moment.part+seconds/secondsADay)
}
