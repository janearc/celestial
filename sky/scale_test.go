package sky_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/sky"
)

// on is an instant on another scale, or the test stops.
func on(test stopper, moment sky.Instant, scale sky.Scale) sky.Instant {
	test.Helper()
	moved, err := moment.On(scale)
	if err != nil {
		test.Fatal(err)
	}
	return *moved
}

// Tai less utc is as iers publishes it in bulletin c: 10 s from 1972,
// 36 s through 2016, 37 s from 2017 january 1.
func TestTAIIsUTCAndItsLeapSeconds(test *testing.T) {
	numbers := clockNumbers(test)
	for _, when := range numbers.TAILessUTC {
		noon := when.timed
		noon.Hour = numbers.Noon
		civil := at(test, noon, sky.UTC())
		atomic := on(test, civil, sky.TAI())
		wall := at(test, noon, sky.TAI())
		if got := apart(atomic, wall); !(math.Abs(got-when.Ahead) <=
			numbers.SecondsTolerance) {
			test.Errorf("%d-%d-%d: tai is %.6f s past utc; iers says %g",
				when.Year, when.Month, when.Day, got, when.Ahead)
		}
	}
}

// The test and the anti-test: the leap second at the end of 2016 is a
// real second of utc, and it is 37 s before the new year on tai; a leap
// second where there was none is no time, and gives no instant.
func TestALeapSecondIsAnInstant(test *testing.T) {
	numbers := clockNumbers(test)
	leap := numbers.LeapSecond
	civil := at(test, leap.At, sky.UTC())
	atomic := on(test, civil, sky.TAI())
	want := at(test, leap.OnTAI, sky.TAI())
	if got := apart(atomic, want); !(math.Abs(got) <=
		numbers.SecondsTolerance) {
		test.Errorf("the leap second is %.6f s from its tai", got)
	}
	none := numbers.NoLeapSecond
	if moment, err := sky.Calendar(none.Year, none.Month, none.Day,
		none.Hour, none.Minute, none.Second, sky.UTC()); !errors.Is(err,
		sky.ErrDate()) || moment != nil {
		test.Errorf("a leap second where there was none is taken: %v", err)
	}
}

// Tt is tai and 32.184 s, by the iau's definition, and going there and
// back is the same instant.
func TestTTIsTAIAndThirtyTwoSeconds(test *testing.T) {
	numbers := clockNumbers(test)
	atomic := at(test, numbers.TT.At, sky.TAI())
	dynamic := on(test, atomic, sky.TT())
	wall := at(test, numbers.TT.At, sky.TT())
	if got := apart(dynamic, wall); !(math.Abs(got-numbers.TT.Ahead) <=
		numbers.SecondsTolerance) {
		test.Errorf("tt is %.6f s past tai; the iau says %v", got,
			numbers.TT.Ahead)
	}
	back := on(test, dynamic, sky.TAI())
	if got := apart(back, atomic); !(math.Abs(got) <= numbers.SecondsTolerance) {
		test.Errorf("tai to tt and back moves %.9f s", got)
	}
	if same := on(test, atomic, sky.TAI()); same != atomic {
		test.Errorf("tai on tai is %v, not %v", same, atomic)
	}
}

// Tdb keeps within two milliseconds of tt over a year, and comes back.
func TestTDBIsWithinTwoMillisecondsOfTT(test *testing.T) {
	numbers := clockNumbers(test)
	whole, fraction := at(test, numbers.TDB.From, sky.TT()).Julian()
	for day := numbers.TDB.Every; day <= numbers.TDB.Days; day +=
		numbers.TDB.Every {
		moment, err := sky.FromJulian(whole+float64(day), fraction,
			sky.TT())
		if err != nil {
			test.Fatal(err)
		}
		barycentric := on(test, *moment, sky.TDB())
		if got := apart(barycentric, *moment); !(math.Abs(got) <=
			numbers.TDB.MostSeconds) {
			test.Errorf("day %d: tdb is %.6f s from tt", day, got)
		}
		back := on(test, barycentric, sky.TT())
		if got := apart(back, *moment); !(math.Abs(got) <=
			numbers.SecondsTolerance) {
			test.Errorf("day %d: tt to tdb and back moves %.9f s", day,
				got)
		}
	}
}

// The anti-test: utc before 1972 had seconds of other lengths, ut1 needs
// iers's measurements of the earth's turning, a leap second past what
// iers has published is not known, and no scale is none: each is
// refused, by name, and gives no instant.
func TestWhatIsNotKnownIsRefused(test *testing.T) {
	numbers := clockNumbers(test)
	var none sky.Scale
	for _, asked := range []struct {
		from  sky.Instant
		scale sky.Scale
	}{
		{at(test, numbers.TooEarly, sky.UTC()), sky.TAI()},
		{at(test, numbers.TooLate, sky.UTC()), sky.TAI()},
		{at(test, numbers.UT1Day, sky.TT()), sky.UT1()},
		{at(test, numbers.UT1Day, sky.UT1()), sky.TT()},
		{at(test, numbers.UT1Day, sky.TT()), none},
		{at(test, numbers.TooLate, sky.TAI()), sky.UTC()},
	} {
		moment, err := asked.from.On(asked.scale)
		if !errors.Is(err, sky.ErrScale()) || moment != nil {
			test.Errorf("%v went to %v: %v", asked.from.Scale(),
				asked.scale, err)
		}
	}
}

// Utc to tai and back is the same instant, every half second across the
// leap second that ended 2016, the leap second itself included.
func TestUTCGoesToTAIAndBackAcrossALeapSecond(test *testing.T) {
	numbers := clockNumbers(test)
	for _, step := range numbers.AcrossLeap {
		civil := at(test, step, sky.UTC())
		atomic := on(test, civil, sky.TAI())
		back := on(test, atomic, sky.UTC())
		year, month, day, hour, minute, second := back.Calendar()
		if year != step.Year || month != step.Month || day != step.Day ||
			hour != step.Hour || minute != step.Minute ||
			!(math.Abs(second-step.Second) <= numbers.SecondsTolerance) {
			test.Errorf("%+v utc comes back %d-%d-%d %d:%d:%.6f", step,
				year, month, day, hour, minute, second)
		}
	}
}

// Every scale says its name.
func TestEveryScaleSaysItsName(test *testing.T) {
	for name, scale := range map[string]sky.Scale{"utc": sky.UTC(),
		"tai": sky.TAI(), "tt": sky.TT(), "tdb": sky.TDB(),
		"ut1": sky.UT1(), "gps": sky.GPS()} {
		if scale.String() != name {
			test.Errorf("%s is called %q", name, scale.String())
		}
	}
}

// Tdb less tt is all seven terms of kaplan's equation 2.6 (usno circular
// 179), worked here from the test data's own copy of the circular's
// coefficients: at j2000, today, and 1600 and 2200, where the seventh
// term, which grows by the century, is largest. A term left out, or the
// seventh not scaled by the centuries, is off by microseconds.
func TestTDBIsKaplansSevenTerms(test *testing.T) {
	kaplan := clockNumbers(test).Kaplan
	for _, julian := range kaplan.Julians {
		moment, err := sky.FromJulian(julian, nought, sky.TT())
		if err != nil {
			test.Fatal(err)
		}
		barycentric := on(test, *moment, sky.TDB())
		got := apart(barycentric, *moment)
		centuries := (julian - kaplan.J2000) / kaplan.Century
		want := centuries * kaplan.Secular.at(centuries)
		for _, each := range kaplan.Periodic {
			want += each.at(centuries)
		}
		if !(math.Abs(got-want) <= kaplan.Tolerance) {
			test.Errorf("at jd %v tdb less tt is %.12f s; equation 2.6 "+
				"gives %.12f", julian, got, want)
		}
	}
}

// at is a term of kaplan's series at so many julian centuries.
func (each term) at(centuries float64) float64 {
	return each.Amplitude * math.Sin(each.Rate*centuries+each.Phase)
}
