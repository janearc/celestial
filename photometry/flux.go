package photometry

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/radiation"
)

// Flux stores spectral flux density and its uncertainty in W m^-2 Hz^-1.
type Flux struct{ density, sigma float64 }

// InSI returns flux density and uncertainty in W m^-2 Hz^-1.
func (flux *Flux) InSI() (density, sigma float64) {
	return flux.density, flux.sigma
}

// InJanskys returns flux density and uncertainty in janskys.
func (flux *Flux) InJanskys() (density, sigma float64) {
	perJansky := radiation.JanskyToSI(oneJansky)
	return flux.density / perJansky, flux.sigma / perJansky
}

// oneJansky is a flux density of one jansky. Defined by the unit.
const oneJansky = 1

// nothing is no error at all. Defined by the scale.
const nothing = 0.0

// leastNormal is the least flux a float64 holds with all its digits.
// Defined by float64: its least normal number, 2^-1022.
const leastNormal = 0x1p-1022

// Flux converts a magnitude and uncertainty in a survey band using F = F0 *
// 100^(-m/5), with the band’s published zero point F0.
//
// Uncertainty is propagated to first order using dF/dm = -F*ln(100)/5. That
// approximation suits small detection errors; uncertainty near one magnitude
// produces an asymmetric flux interval this type cannot express.
//
// Unknown bands, negative or NaN uncertainty, nonfinite magnitude, and fluxes
// outside float64’s normal range return an error. The propagated uncertainty
// must also be representable.
func (survey *Survey) Flux(band string, magnitude,
	magnitudeError float64) (*Flux, error) {
	zero, held := survey.zeros[band]
	if !held {
		return nil, fmt.Errorf("photometry: %q is not a band of %s",
			band, survey.name)
	}
	if !(magnitudeError >= nothing) {
		return nil, fmt.Errorf("photometry: an error of %v "+
			"magnitudes, below %v", magnitudeError, nothing)
	}
	density := zero * math.Pow(pogsonFactor, -magnitude/pogsonStep)
	if !(density >= leastNormal && density <= math.MaxFloat64) {
		return nil, fmt.Errorf("photometry: magnitude %v in %s is "+
			"a flux of %v, outside %v to %v", magnitude, band,
			density, leastNormal, math.MaxFloat64)
	}
	slope := math.Log(pogsonFactor) / pogsonStep
	sigma := density * slope * magnitudeError
	if !(sigma <= math.MaxFloat64) {
		return nil, fmt.Errorf("photometry: an error of %v "+
			"magnitudes is a flux error of %v, past %v",
			magnitudeError, sigma, math.MaxFloat64)
	}
	return &Flux{density: density, sigma: sigma}, nil
}
