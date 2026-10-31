package sky_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// timed is a date and a time of day, as the test data writes it.
type timed struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Day    int     `json:"day"`
	Hour   int     `json:"hour"`
	Minute int     `json:"minute"`
	Second float64 `json:"second"`
}

// term is one term of kaplan's series for tdb less tt: an amplitude in
// seconds, a rate in radians a julian century and a phase in radians.
type term struct {
	Amplitude float64 `json:"amplitude"`
	Rate      float64 `json:"rate"`
	Phase     float64 `json:"phase"`
}

// arc is an angle in degrees, or hours, minutes and seconds.
type arc struct {
	Degrees float64 `json:"degrees"`
	Hours   float64 `json:"hours"`
	Minutes float64 `json:"minutes"`
	Seconds float64 `json:"seconds"`
}

// dated is a calendar date, its day carrying the time as a fraction,
// and the julian date meeus gives for it.
type dated struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Day    float64 `json:"day"`
	Julian float64 `json:"julian"`
}

// clocked is the numbers sky's clock tests use, from testdata/sky.json.
type clocked struct {
	Sexagesimal struct {
		MinutesADegree float64 `json:"minutes a degree"`
		SecondsADegree float64 `json:"seconds a degree"`
		DegreesAnHour  float64 `json:"degrees an hour"`
	} `json:"sexagesimal"`
	Sidereal struct {
		At               timed   `json:"at"`
		Time             arc     `json:"time"`
		SecondsADegree   float64 `json:"seconds a degree of time"`
		ToleranceSeconds float64 `json:"tolerance seconds"`
	} `json:"sidereal"`
	DUT1 struct {
		Seconds   float64 `json:"seconds"`
		TooMuch   float64 `json:"too much"`
		Tolerance float64 `json:"tolerance"`
	} `json:"dut1"`
	Washington struct {
		Latitude  arc `json:"latitude"`
		Longitude arc `json:"longitude"`
	} `json:"washington"`
	Venus struct {
		Ascension   arc     `json:"ascension"`
		Declination arc     `json:"declination"`
		Azimuth     float64 `json:"azimuth"`
		Altitude    float64 `json:"altitude"`
		Tolerance   float64 `json:"tolerance"`
	} `json:"venus"`
	PoleHours       []float64 `json:"pole hours"`
	PoleDeclination float64   `json:"pole declination"`
	PoleTolerance   float64   `json:"pole tolerance"`
	NotPlaces       []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"not places"`
	NotDeclinations []float64 `json:"not declinations"`
	MeeusDates      []dated   `json:"meeus dates"`
	DateTolerance   float64   `json:"date tolerance"`
	NotDates        []timed   `json:"not dates"`
	LeapDay         timed     `json:"leap day"`
	JulianLeapDay   timed     `json:"julian leap day"`
	WashingtonUp    float64   `json:"washington height"`
	NotTimes        []timed   `json:"not times"`
	Noon            int       `json:"noon"`
	TAILessUTC      []struct {
		timed
		Ahead float64 `json:"ahead"`
	} `json:"tai less utc"`
	SecondsTolerance float64 `json:"seconds tolerance"`
	LeapSecond       struct {
		At    timed `json:"at"`
		OnTAI timed `json:"on tai"`
	} `json:"leap second"`
	NoLeapSecond timed `json:"no leap second"`
	TT           struct {
		At    timed   `json:"at"`
		Ahead float64 `json:"ahead"`
	} `json:"tt"`
	TDB struct {
		From        timed   `json:"from"`
		Days        int     `json:"days"`
		Every       int     `json:"every"`
		MostSeconds float64 `json:"most seconds"`
	} `json:"tdb"`
	Kaplan struct {
		Periodic  []term    `json:"periodic"`
		Secular   term      `json:"secular"`
		J2000     float64   `json:"j2000"`
		Century   float64   `json:"century days"`
		Julians   []float64 `json:"julian dates"`
		Tolerance float64   `json:"tolerance"`
	} `json:"kaplan 2.6"`
	TooEarly   timed   `json:"too early"`
	TooLate    timed   `json:"too late"`
	UT1Day     timed   `json:"ut1 day"`
	AcrossLeap []timed `json:"across the leap"`
	GPS        struct {
		Wall        timed     `json:"wall"`
		BehindTAI   float64   `json:"behind tai"`
		Epoch       timed     `json:"epoch"`
		GW150914    float64   `json:"gw150914"`
		GW150914UTC timed     `json:"gw150914 utc"`
		GW150914Tol float64   `json:"gw150914 tolerance"`
		Drawn       span      `json:"drawn seconds"`
		NotSeconds  []float64 `json:"not seconds"`
	} `json:"gps"`
	NotJulian  []float64 `json:"not julian dates"`
	TableEdges struct {
		BeforeFirst timed `json:"before the first"`
		First       struct {
			timed
			Ahead float64 `json:"ahead"`
		} `json:"first"`
		Last struct {
			timed
			Ahead float64 `json:"ahead"`
		} `json:"last"`
		Until timed `json:"until"`
	} `json:"table edges"`
	EitherSide []struct {
		timed
		Ahead float64 `json:"ahead"`
	} `json:"either side of a leap"`
	NoTableNeeded []timed   `json:"no leap table needed"`
	TDBPeak       float64   `json:"tdb peak seconds"`
	LeapGaps      []float64 `json:"across the leap gaps"`
	DayLengths    []struct {
		From    timed   `json:"from"`
		To      timed   `json:"to"`
		Seconds float64 `json:"seconds"`
	} `json:"day lengths"`
	Consecutive []struct {
		Day  timed `json:"day"`
		Next timed `json:"next"`
	} `json:"consecutive days"`
	Hair struct {
		Julian         float64 `json:"julian"`
		Less           float64 `json:"less"`
		EarliestSecond float64 `json:"earliest second"`
		Midnight       timed   `json:"midnight"`
		DayBefore      timed   `json:"day before"`
	} `json:"hair before midnight"`
	SouthPole float64 `json:"south pole declination"`
	DueSouth  float64 `json:"due south degrees"`
}

// clockNumbers reads the test numbers, or the test stops.
func clockNumbers(test testing.TB) clocked {
	test.Helper()
	var numbers clocked
	raw, err := os.ReadFile("testdata/sky.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// at is the instant at a date and time on a scale, or the test stops.
func at(test stopper, when timed, scale sky.Scale) sky.Instant {
	test.Helper()
	moment, err := sky.Calendar(when.Year, when.Month, when.Day, when.Hour,
		when.Minute, when.Second, scale)
	if err != nil {
		test.Fatal(err)
	}
	return *moment
}

// secondsADay is a day of tai, tt and tdb. Defined by the si.
const secondsADay = 86400.0

// apart is how many seconds one instant is after another, both read on
// the same scale's julian dates.
func apart(later, earlier sky.Instant) float64 {
	laterWhole, laterFraction := later.Julian()
	whole, fraction := earlier.Julian()
	return ((laterWhole - whole) + (laterFraction - fraction)) * secondsADay
}

// degreesOf is an arc in degrees: its degrees, or its hours of fifteen
// degrees each, and its minutes and seconds of that unit, the sign the
// degrees' or the hours'.
func degreesOf(numbers clocked, angle arc) float64 {
	sixty := numbers.Sexagesimal
	unit := one
	if angle.Hours != nought {
		unit = sixty.DegreesAnHour
	}
	large := angle.Degrees + angle.Hours*sixty.DegreesAnHour
	parts := angle.Minutes/sixty.MinutesADegree +
		angle.Seconds/sixty.SecondsADegree
	return math.Copysign(math.Abs(large)+unit*parts, large)
}

// one is a single unit. Defined by counting.
const one = 1.0

// angleOf is an arc as an angle.
func angleOf(numbers clocked, angle arc) measure.Angle {
	return measure.Degrees(degreesOf(numbers, angle))
}
