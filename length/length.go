// Package length names astronomical and physical length units in metres. Use SI
// prefixes for multiples: si.Kilo(Parsec) or si.Micro(Metre).
//
// All listed units except PlanckLength are exact definitions, cited beside
// them. PlanckLength depends on the measured gravitational constant G.
package length

import (
	"math"

	"github.com/janearc/celestial/si"
)

// The lengths, in metres.
const (
	// Metre is the SI's unit of length. Defined by the SI since 1983, by
	// the speed of light: the distance light travels in 1/299792458 of a
	// second.
	Metre = 1.0

	// Angstrom is a unit outside the si that x-ray papers still use.
	// Defined as 1e-10 metres by the SI Brochure, 9th edition, section
	// 4, table 8, read on 2026-10-31: "1 Å = 0.1 nm = 10−10 m".
	Angstrom = 1e-10

	// AU is the astronomical unit. Defined in metres by the IAU in 2012
	// (resolution B2), as the IAU's 2015 resolutions restate it in B3's
	// endnote 4, read on 2026-10-31. It began as the Earth's mean distance
	// from the Sun and is now a definition.
	AU = 149597870700.0

	// Parsec is the distance at which one AU subtends one arcsecond:
	// 648000/pi AU, defined by the IAU's 2015 resolution B2, footnote 4,
	// read on 2026-10-31. 648000 is the arcseconds in half a circle.
	Parsec = arcsecondsInHalfACircle / math.Pi * AU

	// LightYear is the distance light travels in a julian year: c times
	// 365.25 days of 86400 seconds. Both are exact, so the light-year is
	// too, and there is no measured figure in it.
	LightYear = si.Speed * julianYear
)

// The parts of the definitions above.
const (
	// degreesInHalfACircle is defined by the degree.
	degreesInHalfACircle = 180
	// arcsecondsInADegree is defined by the arcsecond: sixty arcminutes
	// of sixty arcseconds.
	arcsecondsInADegree = 3600
	// arcsecondsInHalfACircle is defined by the two above.
	arcsecondsInHalfACircle = degreesInHalfACircle * arcsecondsInADegree
	// daysInAJulianYear is defined by the IAU for the light-year.
	daysInAJulianYear = 365.25
	// secondsInADay is defined by the SI: a day of 86400 seconds.
	secondsInADay = 86400
	// julianYear is the IAU's year for the light-year, in seconds.
	// Defined by the two above.
	julianYear = daysInAJulianYear * secondsInADay
	// pisInATurn is how many pi radians make a whole turn. Defined by
	// the radian: a turn is two pi.
	pisInATurn = 2
	// hbar is the reduced Planck constant, h over two pi. Defined, as
	// h is.
	hbar = si.Planck / (pisInATurn * math.Pi)
	// cubed is the power c is raised to in the Planck length. Defined by
	// its formula.
	cubed = 3
)

// PlanckLength is sqrt(hbar G / c^3), about 1.616e-35 metres. It is the
// one length here that is measured, not defined: hbar and c are exact,
// but G carries the uncertainty si.Gravitation states, and the Planck
// length inherits half of it.
func PlanckLength() float64 {
	lightCubed := math.Pow(si.Speed, cubed)
	return math.Sqrt(hbar * si.Gravitation() / lightCubed)
}
