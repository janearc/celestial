package radiation

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
)

// The units papers give a band's ends in, each brought to a wavelength
// in nanometres by its own function. A unit joins them when a caller's
// data uses it.

// KeVToNm converts photon energy in keV to wavelength in nanometres. It returns
// an error if the resulting wavelength is not positive and finite.
func KeVToNm(kev float64) (*float64, error) {
	nm := FromEnergy(si.Kilo(kev))
	return wavelength(nm, kev, "keV")
}

// MeVToNm converts photon energy in MeV to wavelength in nanometres. It returns
// an error if the resulting wavelength is not positive and finite.
func MeVToNm(mev float64) (*float64, error) {
	nm := FromEnergy(si.Mega(mev))
	return wavelength(nm, mev, "MeV")
}

// GeVToNm converts photon energy in GeV to wavelength in nanometres. It returns
// an error if the resulting wavelength is not positive and finite.
func GeVToNm(gev float64) (*float64, error) {
	nm := FromEnergy(si.Giga(gev))
	return wavelength(nm, gev, "GeV")
}

// TeVToNm converts photon energy in TeV to wavelength in nanometres. It returns
// an error if the resulting wavelength is not positive and finite.
func TeVToNm(tev float64) (*float64, error) {
	nm := FromEnergy(si.Tera(tev))
	return wavelength(nm, tev, "TeV")
}

// MHzToNm converts frequency in MHz to wavelength in nanometres. It returns an
// error if the resulting wavelength is not positive and finite.
func MHzToNm(mhz float64) (*float64, error) {
	nm := HertzToNm(si.Mega(mhz))
	return wavelength(nm, mhz, "MHz")
}

// GHzToNm converts frequency in GHz to wavelength in nanometres. It returns an
// error if the resulting wavelength is not positive and finite.
func GHzToNm(ghz float64) (*float64, error) {
	nm := HertzToNm(si.Giga(ghz))
	return wavelength(nm, ghz, "GHz")
}

// MicronsToNm converts micrometres to nanometres, rejecting a result that is
// not positive and finite.
func MicronsToNm(um float64) (*float64, error) {
	nm := MetresToNm(si.Micro(um))
	return wavelength(nm, um, "um")
}

// wavelength validates a positive finite result and otherwise returns an error
// naming the original value and unit.
func wavelength(nm, given float64, unit string) (*float64, error) {
	if !(nm > nothing) || math.IsInf(nm, eitherSign) {
		return nil, fmt.Errorf("radiation: %v %s comes to %v nm, no "+
			"wavelength", given, unit, nm)
	}
	return &nm, nil
}
