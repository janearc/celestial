package measure

import (
	"math"

	"github.com/janearc/celestial/si"
)

// The parts of the units below, each defined.
const (
	// electronvoltsInAnMeV is defined by the SI's prefix mega, 10^6.
	electronvoltsInAnMeV = 1e6
	// pisInATurn is how many pi radians make a turn. Defined by the
	// radian: a turn is two pi.
	pisInATurn = 2
	// fullTurn is a whole circle in radians, which is how h and the
	// reduced constant differ. Defined by the two above.
	fullTurn = pisInATurn * math.Pi
	// secondsPerDay is defined by the SI: a day of 86400 seconds.
	secondsPerDay = 86400.0
	// daysPerYear is defined by the IAU's Julian year, the year of
	// astronomy.
	daysPerYear = 365.25
)

// The bridges from the SI to natural units, defined by the SI's defining
// constants, so no conversion through them adds any doubt.
const (
	// joulesPerMeV is an MeV in joules. Defined by e and mega.
	joulesPerMeV = si.Charge * electronvoltsInAnMeV
	// reduced is the reduced Planck constant, h over a full turn, in J s.
	// Defined, as h is.
	reduced = si.Planck / fullTurn
)

// One of each SI unit, in natural units, each defined by the constants above.
// Mass and temperature come out in MeV, length and time in one over MeV, and
// power, energy over time, in MeV squared.
const (
	kilogram     = si.Speed * si.Speed / joulesPerMeV
	second       = joulesPerMeV / reduced
	metre        = joulesPerMeV / (reduced * si.Speed)
	joule        = kilogram * metre * metre / (second * second)
	kelvin       = si.Boltzmann * joule
	watt         = joule / second
	squareMetre  = metre * metre
	year         = daysPerYear * secondsPerDay * second
	electronvolt = si.Charge * joule
)
