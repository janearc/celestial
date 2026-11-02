package radiation

import (
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/si"
)

// Radiation runs from gamma rays, a few thousandths of a nanometre long,
// to radio waves, metres and more. radiation speaks of all of it in one
// unit, the nanometre, through the si's own exact constants.

// Base units have one name; multiples use SI prefixes. A keV is
// si.Kilo(Electronvolt), and the hydrogen line is 1420.405751 * si.Mega(Hertz).
// Energy and frequency convert through photon relations; metres use MetresToNm.
const (
	// Electronvolt is one electronvolt, the unit energies here are in.
	// Defined: the unit itself.
	Electronvolt = 1.0
	// Hertz is one hertz, the unit frequencies here are in. Defined: the
	// unit itself.
	Hertz = 1.0
)

// FromEnergy returns photon wavelength in nanometres from energy in
// electronvolts.
func FromEnergy(ev float64) float64 { return hc / ev }

// Energy returns photon energy in electronvolts from wavelength in nanometres,
// reversing FromEnergy.
func Energy(nm float64) float64 { return hc / nm }

// nanometre is one nanometre, in metres.
var nanometre = si.Nano(length.Metre)

// hc is a photon's energy times its wavelength, in electronvolt nanometres.
var hc = si.Planck * si.Speed / si.Charge / nanometre

// MetresToNm converts metres to nanometres. For example, MetresToNm(3.4 *
// si.Micro(length.Metre)) gives WISE W1’s 3400 nm.
func MetresToNm(metres float64) float64 { return metres / nanometre }

// NmToMetres converts nanometres to metres, reversing MetresToNm.
func NmToMetres(nm float64) float64 { return nm * nanometre }

// HertzToNm converts frequency in hertz to wavelength in nanometres.
func HertzToNm(hz float64) float64 { return MetresToNm(si.Speed / hz) }

// NmToHertz converts nanometres to hertz, reversing HertzToNm.
func NmToHertz(nm float64) float64 { return si.Speed / NmToMetres(nm) }
