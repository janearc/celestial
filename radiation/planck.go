package radiation

import (
	"math"

	"github.com/janearc/celestial/si"
)

// The parts of Planck's law, each defined by it.
const (
	planckScale = 2 // defined by planck's law: 2hc^2
	planckPower = 5 // defined by planck's law: lambda^5
)

// Planck returns black-body spectral radiance at wavelength nm and temperature
// kelvin, in W m^-2 sr^-1 nm^-1:
//
//	2hc² / (λ⁵ * (exp(hc / (λkT)) − 1)).
//
// At or below zero kelvin it returns zero. As with the low-level conversions,
// nm is assumed to be a valid wavelength.
func Planck(nm, kelvin float64) float64 {
	if kelvin <= nothing {
		return nothing
	}
	metres := NmToMetres(nm)
	energy := si.Planck * si.Speed
	perMetre := planckScale * energy * si.Speed /
		(math.Pow(metres, planckPower) *
			math.Expm1(energy/(metres*si.Boltzmann*kelvin)))
	return perMetre * nanometre
}
