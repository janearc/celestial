package sky

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/research"
)

// Scale identifies UTC, TAI, TT, TDB, UT1, or GPS time. The zero value is
// invalid and is rejected when constructing or converting instants.
type Scale struct{ name string }

// UTC is coordinated universal time, the civil clock, with leap seconds.
func UTC() Scale { return Scale{"utc"} }

// TAI is international atomic time.
func TAI() Scale { return Scale{"tai"} }

// TT is terrestrial time: TAI plus exactly 32.184 seconds.
func TT() Scale { return Scale{"tt"} }

// TDB is barycentric dynamical time, differing periodically from TT by less
// than about two milliseconds.
func TDB() Scale { return Scale{"tdb"} }

// UT1 is time based on Earth’s rotation.
func UT1() Scale { return Scale{"ut1"} }

// GPS is the continuous satellite time scale, TAI minus 19 seconds, also used
// for LIGO, Virgo, and the open science centre event timestamps.
func GPS() Scale { return Scale{"gps"} }

// String is the scale's name, or "no scale" for the zero Scale.
func (scale Scale) String() string {
	if !scale.known() {
		return "no scale"
	}
	return scale.name
}

// known is whether a scale is one of the six.
func (scale Scale) known() bool { return scale.name != "" }

// ttLessTAI is TT less TAI, in seconds. Defined by the IAU's definition of tt.
const ttLessTAI = 32.184

// taiLessGPS is TAI less GPS time, in seconds. Defined: GPS time was set to UTC
// at its epoch, 1980 January 6 0h UTC, when TAI was 19 s ahead of UTC, and GPS
// time has never taken a leap second since (IS-GPS-200, section 3.3.4).
const taiLessGPS = 19.0

// On expresses the same instant on another supported scale. Unknown scales
// and UTC dates outside the leap-second table’s coverage return an error.
// UT1 conversions require Earth-rotation data; use UT1 with a measured DUT1
// instead.
func (moment Instant) On(scale Scale) (*Instant, error) {
	switch {
	case !scale.known():
		return nil, fmt.Errorf("sky: an instant on %v", scale)
	case scale == moment.scale:
		return &moment, nil
	case scale == UT1() || moment.scale == UT1():
		return nil, errors.New("sky: ut1 needs iers's earth " +
			"orientation, not held yet")
	}
	atomic, err := moment.toTAI()
	if err != nil {
		return nil, err
	}
	return atomic.fromTAI(scale)
}

// toTAI converts an instant to TAI.
func (moment Instant) toTAI() (*Instant, error) {
	switch moment.scale {
	case TT():
		on := moment.relabel(TAI()).after(-ttLessTAI)
		return &on, nil
	case GPS():
		on := moment.relabel(TAI()).after(taiLessGPS)
		return &on, nil
	case TDB():
		dynamic := moment.relabel(TT()).after(-tdbLessTT(moment))
		return dynamic.toTAI()
	case UTC():
		return moment.utcToTAI()
	}
	return &moment, nil
}

// fromTAI converts a TAI instant to the requested scale.
func (atomic Instant) fromTAI(scale Scale) (*Instant, error) {
	on := atomic
	switch scale {
	case TT():
		on = atomic.relabel(TT()).after(ttLessTAI)
	case GPS():
		on = atomic.relabel(GPS()).after(-taiLessGPS)
	case TDB():
		dynamic := atomic.relabel(TT()).after(ttLessTAI)
		on = dynamic.relabel(TDB()).after(tdbLessTT(dynamic))
	case UTC():
		return atomic.taiToUTC()
	}
	return &on, nil
}

// relabel reinterprets the same day and fraction on another scale as an
// intermediate conversion step, not a completed conversion.
func (moment Instant) relabel(scale Scale) Instant {
	moment.scale = scale
	return moment
}

// The epoch and the century Kaplan's terms count from, each defined: j2000.0,
// Julian date 2451545.0 on TT, and the Julian century of 36525 days.
const (
	// j2000 is the Julian date of j2000.0. Defined by the iau.
	j2000 = 2451545.0
	// daysACentury is a Julian century. Defined by the iau.
	daysACentury   = 36525.0
	kaplanFirst    = "tdb-first"
	kaplanSecond   = "tdb-second"
	kaplanThird    = "tdb-third"
	kaplanFourth   = "tdb-fourth"
	kaplanFifth    = "tdb-fifth"
	kaplanSixth    = "tdb-sixth"
	kaplanSecular  = "tdb-secular"
	amplitudeOfTDB = "-amplitude"
	rateOfTDB      = "-rate"
	phaseOfTDB     = "-phase"
)

// kaplanPeriodic is the six periodic terms of the circular's equation
// 2.6, in its order.
var kaplanPeriodic = []string{kaplanFirst, kaplanSecond, kaplanThird,
	kaplanFourth, kaplanFifth, kaplanSixth}

// tdbLessTT is TDB less TT in seconds at an instant on TT or TDB, by all seven
// terms of Kaplan, USNO circular 179 (2005), equation 2.6, as celestial's
// research data holds them: good, the circular says, to about 10 microseconds
// from 1600 to 2200.
//
// Six are periodic; the seventh's amplitude grows by the century, so it is its
// term times the centuries from j2000.
func tdbLessTT(moment Instant) float64 {
	whole, fraction := moment.Julian()
	centuries := ((whole - j2000) + fraction) / daysACentury
	var sum float64
	for _, term := range kaplanPeriodic {
		sum += tdbTerm(term, centuries)
	}
	return sum + centuries*tdbTerm(kaplanSecular, centuries)
}

// tdbTerm evaluates one periodic term at Julian centuries from J2000: amplitude
// times sin(rate * centuries + phase).
func tdbTerm(term string, centuries float64) float64 {
	amplitude := research.Value(term + amplitudeOfTDB)
	turned := research.Value(term+rateOfTDB)*centuries +
		research.Value(term+phaseOfTDB)
	return amplitude * math.Sin(turned)
}
