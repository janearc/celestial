package photometry

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/radiation"
)

// Flux is a flux density and its error, held in W m^-2 Hz^-1.
type Flux struct{ density, sigma float64 }

// InSI is the flux density and its error in W m^-2 Hz^-1.
func (flux *Flux) InSI() (density, sigma float64) {
	return flux.density, flux.sigma
}

// InJanskys is the flux density and its error in janskys.
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

// Flux is the flux density a magnitude in one of the survey's bands
// stands for, and its error: F0 times 100 to the minus m over 5, where
// F0 is the band's zero point.
//
// The error is carried to first order: dF/dm is minus F times the
// natural log of 100, over 5. That holds when the error is small, as it
// is for a detection; an error near a magnitude makes a lopsided flux
// error this does not describe.
//
// Flux refuses, and then gives no flux:
//   - a band the survey does not have
//   - an error below nothing
//   - a magnitude whose flux a float64 cannot hold: too bright, past the
//     largest, or too faint, below the least normal, where it would lose
//     its digits
//   - an error whose flux error a float64 cannot hold
//
// A magnitude that is not a number, or is either infinity, gives a flux
// outside that range, and an error that is not a number is not at least
// nothing, so they are refused with them.
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
