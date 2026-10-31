package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// mostDUT1 is how far iers keeps ut1 from utc, in seconds: within 0.9 s,
// by leap seconds. Defined by the iers's rule.
const mostDUT1 = 0.9

// UT1 is a utc instant on ut1, the earth's own turning, given iers's
// dut1 for the day: ut1 less utc. It refuses an instant not on utc and a
// dut1 past 0.9 s, which iers never lets it reach, and then gives no
// instant.
func (moment Instant) UT1(dut1 measure.Duration) (*Instant, error) {
	offset := dut1.InSeconds()
	if moment.scale != UTC() || !(math.Abs(offset) <= mostDUT1) {
		return nil, fmt.Errorf("%w: ut1 is utc and a dut1 within 0.9 s",
			errScale)
	}
	seconds := moment.part * dayLength(UTC(), moment.day)
	turning := settled(UT1(), moment.day, (seconds+offset)/secondsADay)
	return &turning, nil
}

// The iau's 1982 expression for greenwich mean sidereal time, as meeus,
// astronomical algorithms, equation 12.4, gives it. Each term is defined
// by that expression, which is what gmst is.
//
// Read on 2026-10-31 at its source, aoki and others, "the new definition
// of universal time" (astron. astrophys. 105, 359, 1982), equation 14,
// by eye from the ads scan: 67310.54841 s + (876600 h + 8640184.812866 s)
// T + 0.093104 s T^2 - 6.2 x 10^-6 s T^3.
//
// Over 240 seconds of time a degree and 36525 days a century those are
// 280.460618375, 360.98564736629, 0.000387933333 and 1 / 38709677.4;
// these four are meeus's roundings of them, well inside the tests.
const (
	// gmstAtJ2000 is in degrees. Defined by the iau 1982.
	gmstAtJ2000 = 280.46061837
	// gmstPerDay is in degrees a day. Defined by the iau 1982.
	gmstPerDay = 360.98564736629
	// gmstQuadratic is in degrees a century squared. Defined by the iau
	// 1982.
	gmstQuadratic = 0.000387933
	// gmstCubic is its cubic term's divisor. Defined by the iau 1982.
	gmstCubic = 38710000.0
	// degreesATurn is defined by the degree.
	degreesATurn = 360.0
)

// MeanSidereal is greenwich mean sidereal time at an instant on ut1: how
// far the earth has turned against the stars, by the iau's expression of
// 1982. It refuses an instant on another scale, and then gives no angle.
func (moment Instant) MeanSidereal() (*measure.Angle, error) {
	if moment.scale != UT1() {
		return nil, fmt.Errorf("%w: sidereal time is the earth's "+
			"turning, asked of ut1", errScale)
	}
	whole, fraction := moment.Julian()
	days := (whole - j2000) + fraction
	centuries := days / daysACentury
	degrees := gmstAtJ2000 + gmstPerDay*days +
		centuries*centuries*(gmstQuadratic-centuries/gmstCubic)
	within := math.Mod(degrees, degreesATurn)
	turned := math.Mod(within+degreesATurn, degreesATurn)
	angle := measure.Degrees(turned)
	return &angle, nil
}
