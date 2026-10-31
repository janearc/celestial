package sky_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// nextStep is the step after one in a list. Defined by counting.
const nextStep = 1

// taiOf is a utc reading as an instant on tai, or the test stops.
func taiOf(test stopper, when timed) sky.Instant {
	test.Helper()
	civil := at(test, when, sky.UTC())
	return on(test, civil, sky.TAI())
}

// aheadOf is how many seconds tai is past a utc reading, or the test
// stops.
func aheadOf(test *testing.T, when timed) float64 {
	test.Helper()
	atomic := taiOf(test, when)
	wall := at(test, when, sky.TAI())
	return apart(atomic, wall)
}

// The anti-test: the leap-second table speaks for its first day and its
// last, and for no day outside them: the day before 1972 and the day the
// table ends on are refused, and the days either side of a leap carry the
// seconds of their own side and not of the other.
func TestTheTableEndsWhereItEnds(test *testing.T) {
	numbers := clockNumbers(test)
	edges := numbers.TableEdges
	for _, wrong := range []timed{edges.BeforeFirst, edges.Until} {
		moment, err := at(test, wrong, sky.UTC()).On(sky.TAI())
		if !errors.Is(err, sky.ErrScale()) || moment != nil {
			test.Errorf("%+v is in the table: %v", wrong, err)
		}
	}
	held := []struct {
		timed
		Ahead float64
	}{{edges.First.timed, edges.First.Ahead},
		{edges.Last.timed, edges.Last.Ahead}}
	for _, one := range numbers.EitherSide {
		held = append(held, struct {
			timed
			Ahead float64
		}{one.timed, one.Ahead})
	}
	for _, one := range held {
		if got := aheadOf(test, one.timed); !(math.Abs(got-one.Ahead) <=
			numbers.SecondsTolerance) {
			test.Errorf("%d-%d-%d: tai is %.6f s past utc; bulletin c says %g",
				one.Year, one.Month, one.Day, got, one.Ahead)
		}
	}
}

// The anti-test: tt, gps and tdb carry no leap seconds, so an instant on
// any of them is taken to tai wherever it falls, years outside the table
// utc keeps; only utc is refused there.
func TestScalesWithoutLeapSecondsNeedNoTable(test *testing.T) {
	numbers := clockNumbers(test)
	shift := map[string]float64{"tt": -numbers.TT.Ahead,
		"gps": numbers.GPS.BehindTAI}
	for _, when := range numbers.NoTableNeeded {
		for _, scale := range []sky.Scale{sky.TT(), sky.GPS(), sky.TDB()} {
			moment := at(test, when, scale)
			atomic, err := moment.On(sky.TAI())
			if err != nil || atomic == nil {
				test.Errorf("%v on %+v does not reach tai: %v", scale, when,
					err)
				continue
			}
			want, held := shift[scale.String()]
			if got := apart(*atomic, moment); held && !(math.Abs(got-want) <=
				numbers.SecondsTolerance) {
				test.Errorf("%v on %+v is %.6f s from tai, not %v", scale,
					when, got, want)
			}
		}
	}
}

// The anti-test: tdb is not tt under another name. Over the year the
// wobble reaches a millisecond, so a conversion that leaves the reading
// alone is caught.
func TestTDBIsNotTT(test *testing.T) {
	numbers := clockNumbers(test)
	whole, fraction := at(test, numbers.TDB.From, sky.TT()).Julian()
	var most float64
	for day := numbers.TDB.Every; day <= numbers.TDB.Days; day +=
		numbers.TDB.Every {
		moment, err := sky.FromJulian(whole+float64(day), fraction, sky.TT())
		if err != nil {
			test.Fatal(err)
		}
		barycentric := on(test, *moment, sky.TDB())
		off := math.Abs(apart(barycentric, *moment))
		most = math.Max(most, off)
	}
	if !(most >= numbers.TDBPeak) {
		test.Errorf("tdb is at most %.6f s from tt over the year; it "+
			"should reach %v", most, numbers.TDBPeak)
	}
}

// The anti-test: no time is made or lost across the leap second. The
// readings on utc, each taken to tai, are as far apart as the clock says,
// and the day that ends in a leap second is 86401 s long where its
// neighbour is 86400.
func TestNoTimeIsMadeOrLostAcrossALeapSecond(test *testing.T) {
	numbers := clockNumbers(test)
	steps := numbers.AcrossLeap
	for index, gap := range numbers.LeapGaps {
		first, second := steps[index], steps[index+nextStep]
		earlier, later := taiOf(test, first), taiOf(test, second)
		if got := apart(later, earlier); !(math.Abs(got-gap) <=
			numbers.SecondsTolerance) {
			test.Errorf("%+v to %+v is %.6f s on tai; the clock says %v",
				first, second, got, gap)
		}
	}
	for _, day := range numbers.DayLengths {
		from, to := taiOf(test, day.From), taiOf(test, day.To)
		if got := apart(to, from); !(math.Abs(got-day.Seconds) <=
			numbers.SecondsTolerance) {
			test.Errorf("the day from %+v is %.6f s; it should be %v", day.From,
				got, day.Seconds)
		}
	}
}

// The anti-test: gps time takes no leap second. Whatever the date, in
// 1980 or after the leap seconds of 1999 and 2016, the same instant is 19
// s apart on gps and tai, not more as utc has fallen behind.
func TestGPSTakesNoLeapSecond(test *testing.T) {
	numbers := clockNumbers(test)
	whens := []timed{numbers.GPS.Epoch}
	for _, when := range numbers.TAILessUTC {
		noon := when.timed
		noon.Hour = numbers.Noon
		whens = append(whens, noon)
	}
	for _, when := range whens {
		civil := at(test, when, sky.UTC())
		satellite := on(test, civil, sky.GPS())
		atomic := on(test, civil, sky.TAI())
		behind := apart(satellite, atomic)
		if !(math.Abs(behind+numbers.GPS.BehindTAI) <=
			numbers.SecondsTolerance) {
			test.Errorf("%+v: gps is %.6f s from tai, not -%v", when, behind,
				numbers.GPS.BehindTAI)
		}
	}
}

// The anti-test: utc that the leap-second table does not speak for has no
// gps seconds, and the number beside the refusal is nothing.
func TestNoGPSSecondsWhereUTCIsNotKnown(test *testing.T) {
	numbers := clockNumbers(test)
	for _, when := range []timed{numbers.TooEarly, numbers.TooLate} {
		seconds, err := at(test, when, sky.UTC()).GPSSeconds()
		if !errors.Is(err, sky.ErrScale()) || seconds != nought {
			test.Errorf("%+v utc gave gps seconds %v: %v", when, seconds, err)
		}
	}
}

// The anti-test: the ten days the gregorian calendar skipped are skipped
// and no more. The day after 1582 october 4 is october 15, and every day
// round the reform and round the leap days, julian and gregorian, is one
// day, 86400 s, after the one before it.
func TestTheReformSkipsTenDaysAndNoMore(test *testing.T) {
	numbers := clockNumbers(test)
	for _, pair := range numbers.Consecutive {
		before, after := at(test, pair.Day, sky.TT()), at(test, pair.Next,
			sky.TT())
		if got := apart(after, before); !(math.Abs(got-secondsADay) <=
			numbers.SecondsTolerance) {
			test.Errorf("%+v to %+v is %v s; a day is %v", pair.Day,
				pair.Next, got, secondsADay)
		}
	}
}

// The anti-test: a julian date a hair before midnight is the day before,
// at its last second, and not the next day with a time below nothing;
// the same date to the instant is midnight, and the two parts a date is
// given in make no difference.
func TestAJulianDateAHairBeforeMidnightIsTheDayBefore(test *testing.T) {
	numbers := clockNumbers(test)
	hair := numbers.Hair
	moment, err := sky.FromJulian(hair.Julian, -hair.Less, sky.TT())
	if err != nil {
		test.Fatal(err)
	}
	year, month, day, hour, minute, second := moment.Calendar()
	before := hair.DayBefore
	if year != before.Year || month != before.Month || day != before.Day ||
		hour != before.Hour || minute != before.Minute ||
		!(second >= hair.EarliestSecond && second < secondsAMinute) {
		test.Errorf("a hair before midnight is %d-%d-%d %d:%d:%.9f", year,
			month, day, hour, minute, second)
	}
	exact, err := sky.FromJulian(hair.Julian, nought, sky.TT())
	if err != nil {
		test.Fatal(err)
	}
	midnight := at(test, hair.Midnight, sky.TT())
	if got := apart(*exact, midnight); !(math.Abs(got) <=
		numbers.SecondsTolerance) {
		test.Errorf("midnight is %.6f s from its date", got)
	}
	split, err := sky.FromJulian(math.Floor(hair.Julian), hair.Julian-
		math.Floor(hair.Julian), sky.TT())
	if err != nil || *split != *exact {
		test.Errorf("a date in two parts is %v, not %v: %v", split, exact, err)
	}
}

// The anti-test: sidereal time is the earth's turning and no other clock's
// to give: tai, tt, tdb, gps and utc each give none, and no angle.
func TestOnlyUT1HasASiderealTime(test *testing.T) {
	numbers := clockNumbers(test)
	for _, scale := range []sky.Scale{sky.UTC(), sky.TAI(), sky.TT(),
		sky.TDB(), sky.GPS()} {
		angle, err := at(test, numbers.Sidereal.At, scale).MeanSidereal()
		if !errors.Is(err, sky.ErrScale()) || angle != nil {
			test.Errorf("%v gives a sidereal time: %v, %v", scale, angle,
				err)
		}
	}
}

// The anti-test: the south celestial pole stands below the horizon by the
// place's latitude, due south, whatever the time: an altitude below
// nothing is not folded up above it.
func TestTheSouthPoleIsBelowTheHorizon(test *testing.T) {
	numbers := clockNumbers(test)
	south := toward(test, nought, numbers.SouthPole)
	here := washington(test)
	latitude := here.Latitude().InDegrees()
	for _, hours := range numbers.PoleHours {
		turned := measure.Degrees(hours * numbers.Sexagesimal.DegreesAnHour)
		seen := here.Local(south, turned)
		if !(math.Abs(seen.Altitude().InDegrees()+latitude) <=
			numbers.PoleTolerance) ||
			!(math.Abs(seen.Azimuth().InDegrees()-numbers.DueSouth) <=
				numbers.PoleTolerance) {
			test.Errorf("at %g h the south pole is at azimuth %.9f, altitude "+
				"%.9f", hours, seen.Azimuth().InDegrees(),
				seen.Altitude().InDegrees())
		}
	}
}
