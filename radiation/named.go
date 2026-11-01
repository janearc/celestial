package radiation

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
)

// the units papers give a band's ends in, each brought to a wavelength
// in nanometres by its own function. a unit comes here when a caller's
// data uses it.

// KeVToNm returns the wavelength of a photon of the given energy in keV.
// returns nil and error for an energy that gives no wavelength.
func KeVToNm(kev float64) (*float64, error) {
	nm := FromEnergy(si.Kilo(kev))
	return wavelength(nm, kev, "keV")
}

// MeVToNm returns the wavelength of a photon of the given energy in MeV.
// returns nil and error for an energy that gives no wavelength.
func MeVToNm(mev float64) (*float64, error) {
	nm := FromEnergy(si.Mega(mev))
	return wavelength(nm, mev, "MeV")
}

// GeVToNm returns the wavelength of a photon of the given energy in GeV.
// returns nil and error for an energy that gives no wavelength.
func GeVToNm(gev float64) (*float64, error) {
	nm := FromEnergy(si.Giga(gev))
	return wavelength(nm, gev, "GeV")
}

// TeVToNm returns the wavelength of a photon of the given energy in TeV.
// returns nil and error for an energy that gives no wavelength.
func TeVToNm(tev float64) (*float64, error) {
	nm := FromEnergy(si.Tera(tev))
	return wavelength(nm, tev, "TeV")
}

// MHzToNm returns the wavelength of light of the given frequency in MHz.
// returns nil and error for a frequency that gives no wavelength.
func MHzToNm(mhz float64) (*float64, error) {
	nm := HertzToNm(si.Mega(mhz))
	return wavelength(nm, mhz, "MHz")
}

// GHzToNm returns the wavelength of light of the given frequency in GHz.
// returns nil and error for a frequency that gives no wavelength.
func GHzToNm(ghz float64) (*float64, error) {
	nm := HertzToNm(si.Giga(ghz))
	return wavelength(nm, ghz, "GHz")
}

// MicronsToNm returns a wavelength in microns in nanometres. returns nil
// and error for a length that is no wavelength.
func MicronsToNm(um float64) (*float64, error) {
	nm := MetresToNm(si.Micro(um))
	return wavelength(nm, um, "um")
}

// errWavelength is the error for a value that gives no wavelength.
var errWavelength = errors.New("radiation: no wavelength")

// ErrWavelength returns the error for a value that gives no wavelength:
// one whose wavelength is not finite and above zero. zero, negative, NaN
// and infinite values all give it.
func ErrWavelength() error { return errWavelength }

// wavelength returns nm when it is a wavelength, finite and above zero.
// returns nil and error naming the value and unit it came from when not.
func wavelength(nm, given float64, unit string) (*float64, error) {
	if !(nm > nothing) || math.IsInf(nm, eitherSign) {
		return nil, fmt.Errorf("%w: %v %s comes to %v nm",
			errWavelength, given, unit, nm)
	}
	return &nm, nil
}
