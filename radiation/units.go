package radiation

import (
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/si"
)

// radiation runs from gamma rays, a few thousandths of a nanometre long,
// to radio waves, metres and more. to speak of all of it in one unit we
// convert to nanometres. the constants this takes are the si's own, fixed
// by definition, taken from the si package where they are used.

// the units radiation is spoken in. each has one name, and its multiples
// come from the si's prefixes: a keV is si.Kilo(Electronvolt), the
// hydrogen line is 1420.405751 * si.Mega(Hertz).
//
// an eV is an energy, not a wavelength, but most papers on gamma radiation
// speak in eV, so FromEnergy and Energy move between the two. wavelengths
// are in nanometres, and lengths in metres go through MetresToNm.
const (
	// Electronvolt is one electronvolt, the unit energies here are in.
	// Defined: the unit itself.
	Electronvolt = 1.0
	// Hertz is one hertz, the unit frequencies here are in. Defined: the
	// unit itself.
	Hertz = 1.0
)

// FromEnergy is the wavelength, in nanometres, of a photon carrying an
// energy in electronvolts.
func FromEnergy(ev float64) float64 { return hc / ev }

// Energy is the energy, in electronvolts, of a photon of a wavelength in
// nanometres: FromEnergy, undone.
func Energy(nm float64) float64 { return hc / nm }

// nanometre is one nanometre, in metres.
var nanometre = si.Nano(length.Metre)

// hc is a photon's energy times its wavelength, in electronvolt nanometres.
var hc = si.Planck * si.Speed / si.Charge / nanometre

// MetresToNm is a length in metres as nanometres, the unit every range
// here is in: MetresToNm(3.4 * si.Micro(length.Metre)) is WISE's first
// band, 3400 nm.
func MetresToNm(metres float64) float64 { return metres / nanometre }

// NmToMetres is nanometres as metres: MetresToNm, undone.
func NmToMetres(nm float64) float64 { return nm * nanometre }

// HertzToNm is the wavelength, in nanometres, of light of a frequency in
// hertz.
func HertzToNm(hz float64) float64 { return MetresToNm(si.Speed / hz) }

// NmToHertz is the frequency, in hertz, of light of a wavelength in
// nanometres: HertzToNm, undone.
func NmToHertz(nm float64) float64 { return si.Speed / NmToMetres(nm) }
