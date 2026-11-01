package sky

import "math"

// The numbers of meeus, astronomical algorithms, chapter 7, which turns
// dates into julian days and back. Each is defined by his algorithm.
const (
	// julianYearDays is defined by meeus: days in a julian year.
	julianYearDays = 365.25
	// yearOffset is defined by meeus: the years moved to before -4712.
	yearOffset = 4716
	// monthDays is defined by meeus: days a shifted month.
	monthDays = 30.6001
	// mjdOffset is defined by meeus and the mjd: 2400000.5 and 1524.5.
	mjdOffset = 2401525
	// marchFirst is defined by meeus: january and february count last.
	marchFirst = 2
	// monthsMoved is defined by meeus: january is month 13.
	monthsMoved = 12
	// gregorianShift is defined by meeus: the reform's two days.
	gregorianShift = 2
	// centuryEvery is defined by meeus: a century of years.
	centuryEvery = 100
	// leapCentury is defined by meeus: one century in four leaps.
	leapCentury = 4
	// mjdToJulianDay is defined by meeus and the mjd: a julian day's
	// number.
	mjdToJulianDay = 2400001
	// gregorianZero is defined by meeus: the reform's alpha.
	gregorianZero = 1867216.25
	// gregorianDays is defined by meeus: days a gregorian century.
	gregorianDays = 36524.25
	// marchShift is defined by meeus: days from march of -4716.
	marchShift = 1524
	// yearShift is defined by meeus: the year's shift in days.
	yearShift = 122.1
	// lastShifted is defined by meeus: the months that wrap.
	lastShifted = 14
	// backToJanuary is defined by meeus: month 14 is january.
	backToJanuary = 13
	// backAMonth is defined by meeus: a shifted month back.
	backAMonth = 1
	// backToYear is defined by meeus: the year of january and february.
	backToYear = 4715
	// shiftedFebruary is defined by meeus: a shifted year's last month.
	shiftedFebruary = 2
	// alphaOne is defined by meeus: A = Z + 1 + alpha - alpha/4.
	alphaOne = 1
	// nextMonth is defined by meeus: the month after, M + 1.
	nextMonth = 1
	// nextYear is defined by meeus: january counts in the year before.
	nextYear = 1
)

// Calendar is the date and time of the instant on its own scale, by
// meeus, astronomical algorithms, chapter 7: gregorian from 1582 october
// 15 and julian before.
func (moment Instant) Calendar() (year, month, day, hour, minute int,
	second float64) {
	year, month, day = dateOf(moment.day)
	seconds := moment.part * dayLength(moment.scale, moment.day)
	if seconds >= secondsADay {
		return year, month, day, lastHour, lastMinute,
			seconds - (secondsADay - secondsAMinute)
	}
	hour = int(seconds / secondsAnHour)
	minute = int((seconds - float64(hour)*secondsAnHour) / secondsAMinute)
	past := float64(hour)*secondsAnHour + float64(minute)*secondsAMinute
	return year, month, day, hour, minute, seconds - past
}

// dateOf is the date a modified julian day falls on.
func dateOf(mjd int64) (year, month, day int) {
	whole := float64(mjd + mjdToJulianDay)
	shifted := whole
	if mjd >= gregorianFrom {
		alpha := math.Floor((whole - gregorianZero) / gregorianDays)
		leaps := math.Floor(alpha / leapCentury)
		shifted = whole + alphaOne + alpha - leaps
	}
	counted := shifted + marchShift
	years := math.Floor((counted - yearShift) / julianYearDays)
	days := math.Floor(julianYearDays * years)
	months := math.Floor((counted - days) / monthDays)
	day = int(counted - days - math.Floor(monthDays*months))
	month = int(months) - backAMonth
	if months >= lastShifted {
		month = int(months) - backToJanuary
	}
	year = int(years) - yearOffset
	if month <= shiftedFebruary {
		year = int(years) - backToYear
	}
	return year, month, day
}

// dayNumber is the modified julian day a date starts, by meeus,
// astronomical algorithms, chapter 7.
func dayNumber(year, month, day int) int64 {
	gregorian := year > reformYear || year == reformYear &&
		(month > reformMonth ||
			month == reformMonth && day >= skippedUntil)
	if month <= marchFirst {
		year, month = year-nextYear, month+monthsMoved
	}
	var shift float64 // meeus's B: none for a julian date
	if gregorian {
		century := math.Floor(float64(year) / centuryEvery)
		leaps := math.Floor(century / leapCentury)
		shift = gregorianShift - century + leaps
	}
	days := math.Floor(julianYearDays*float64(year+yearOffset)) +
		math.Floor(monthDays*float64(month+nextMonth)) + float64(day) +
		shift
	return int64(days) - mjdOffset
}
