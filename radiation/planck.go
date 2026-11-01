package radiation

import (
	"math"

	"github.com/janearc/celestial/si"
)

// the parts of planck's law, each defined by it.
const (
	planckScale = 2 // defined by planck's law: 2hc^2
	planckPower = 5 // defined by planck's law: lambda^5
)

// Planck returns the spectral radiance of a black body at a wavelength in
// nanometres and a temperature in kelvin, by planck's law, in
// W m^-2 sr^-1 nm^-1: 2hc^2 / lambda^5 / (exp(hc / lambda k T) - 1).
//
// returns zero at or below zero kelvin: a body that cold gives no light.
// like the other conversions here, it trusts nm to be a wavelength.
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
