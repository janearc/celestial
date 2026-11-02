package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// mostDUT1 is how far IERS keeps UT1 from UTC, in seconds: within 0.9 s, by
// leap seconds. Defined by the IERS's rule.
const mostDUT1 = 0.9

// UT1 converts a UTC instant using the day’s measured DUT1 = UT1 − UTC in
// seconds. It requires UTC input and a finite offset within the IERS limit of
// 0.9 seconds.
func (moment Instant) UT1(dut1 measure.Duration) (*Instant, error) {
	offset := dut1.InSeconds()
	if moment.scale != UTC() || !(math.Abs(offset) <= mostDUT1) {
		return nil, fmt.Errorf("sky: ut1 is utc and a dut1 within %v "+
			"s, not %v and %v s", mostDUT1, moment.scale, offset)
	}
	seconds := moment.part * dayLength(UTC(), moment.day)
	turning := settled(UT1(), moment.day, (seconds+offset)/secondsADay)
	return &turning, nil
}

// The IAU's 1982 expression for Greenwich mean sidereal time, as Meeus,
// astronomical algorithms, equation 12.4, gives it. Each term is defined by
// that expression, which is what GMST is.
//
// Read on 2026-10-31 at its source, Aoki and others, "the new definition of
// universal time" (astron. astrophys. 105, 359, 1982), equation 14, by eye from
// the ADS scan: 67310.54841 s + (876600 h + 8640184.812866 s) T + 0.093104 s
// T^2 - 6.2 x 10^-6 s T^3.
//
// Over 240 seconds of time a degree and 36525 days a century those are
// 280.460618375, 360.98564736629, 0.000387933333 and 1 / 38709677.4; these four
// are Meeus's roundings of them, well inside the tests.
const (
	// gmstAtJ2000 is in degrees. Defined by the IAU 1982.
	gmstAtJ2000 = 280.46061837
	// gmstPerDay is in degrees a day. Defined by the IAU 1982.
	gmstPerDay = 360.98564736629
	// gmstQuadratic is in degrees a century squared. Defined by the IAU
	// 1982.
	gmstQuadratic = 0.000387933
	// gmstCubic is its cubic term's divisor. Defined by the IAU 1982.
	gmstCubic = 38710000.0
)

// MeanSidereal returns Greenwich mean sidereal time from a UT1 instant, using
// the IAU 1982 expression. An instant on another scale returns an error rather
// than being treated as UT1.
func (moment Instant) MeanSidereal() (*measure.Angle, error) {
	if moment.scale != UT1() {
		return nil, fmt.Errorf("sky: sidereal time is the earth's "+
			"turning, asked of ut1, not %v", moment.scale)
	}
	whole, fraction := moment.Julian()
	days := (whole - j2000) + fraction
	centuries := days / daysACentury
	degrees := gmstAtJ2000 + gmstPerDay*days +
		centuries*centuries*(gmstQuadratic-centuries/gmstCubic)
	angle := measure.Degrees(intoATurn(degrees, degreesInATurn))
	return &angle, nil
}
