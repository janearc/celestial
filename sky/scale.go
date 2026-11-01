package sky

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/research"
)

// Scale is the clock an instant is on: utc, tai, tt, tdb, ut1 or gps.
// doc/sky.md says what each is. The zero Scale is none of them, and
// every instant refuses it.
type Scale struct{ name string }

// UTC is coordinated universal time, the civil clock, with leap seconds.
func UTC() Scale { return Scale{"utc"} }

// TAI is international atomic time.
func TAI() Scale { return Scale{"tai"} }

// TT is terrestrial time, tai and 32.184 s.
func TT() Scale { return Scale{"tt"} }

// TDB is barycentric dynamical time, tt and a wobble of under two
// thousandths of a second.
func TDB() Scale { return Scale{"tdb"} }

// UT1 is the earth's own turning.
func UT1() Scale { return Scale{"ut1"} }

// GPS is the global positioning system's time, tai less 19 s, the clock
// LIGO, Virgo and the open science centre give their times on.
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

// ttLessTAI is tt less tai, in seconds. Defined by the iau's definition
// of tt.
const ttLessTAI = 32.184

// taiLessGPS is tai less gps time, in seconds. Defined: gps time was set
// to utc at its epoch, 1980 january 6 0h utc, when tai was 19 s ahead of
// utc, and gps time has never taken a leap second since (IS-GPS-200,
// section 3.3.4).
const taiLessGPS = 19.0

// On is the same instant on another scale. ut1 needs iers's measurements
// of the earth's turning, which sky does not hold yet, so it is refused
// by name. On refuses no scale, and then gives no instant.
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

// toTAI is the instant on tai.
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

// fromTAI is an instant on tai told on another scale.
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

// relabel is the same day and fraction read on another scale: a step in
// a conversion, never an answer.
func (moment Instant) relabel(scale Scale) Instant {
	moment.scale = scale
	return moment
}

// The epoch and the century kaplan's terms count from, each defined:
// j2000.0, julian date 2451545.0 on tt, and the julian century of 36525
// days.
const (
	// j2000 is the julian date of j2000.0. Defined by the iau.
	j2000 = 2451545.0
	// daysACentury is a julian century. Defined by the iau.
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

// tdbLessTT is tdb less tt in seconds at an instant on tt or tdb, by all
// seven terms of kaplan, usno circular 179 (2005), equation 2.6, as
// celestial's research data holds them: good, the circular says, to
// about 10 microseconds from 1600 to 2200.
//
// Six are periodic; the seventh's amplitude grows by the century, so it
// is its term times the centuries from j2000.
func tdbLessTT(moment Instant) float64 {
	whole, fraction := moment.Julian()
	centuries := ((whole - j2000) + fraction) / daysACentury
	var sum float64
	for _, term := range kaplanPeriodic {
		sum += tdbTerm(term, centuries)
	}
	return sum + centuries*tdbTerm(kaplanSecular, centuries)
}

// tdbTerm is one term of the circular's series at so many centuries
// from j2000: its amplitude times the sine of its rate times the
// centuries, plus its phase.
func tdbTerm(term string, centuries float64) float64 {
	amplitude := research.Value(term + amplitudeOfTDB)
	turned := research.Value(term+rateOfTDB)*centuries +
		research.Value(term+phaseOfTDB)
	return amplitude * math.Sin(turned)
}
