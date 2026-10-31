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

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// Flux is the flux density a magnitude in one of the survey's bands
// stands for, and its error: F0 times 100 to the minus m over 5, where
// F0 is the band's zero point.
//
// The error is carried to first order: dF/dm is minus F times the
// natural log of 100, over 5. That holds when the error is small, as it
// is for a detection; an error near a magnitude makes a lopsided flux
// error this does not describe.
//
// Flux refuses a band the survey does not have, a magnitude that is not
// a finite number, an error below nothing or not finite, and a magnitude
// so bright its flux is past a float, and then gives no flux.
func (survey *Survey) Flux(band string, magnitude,
	magnitudeError float64) (*Flux, error) {
	zero, held := survey.zeros[band]
	if !held {
		return nil, fmt.Errorf("%w: %q in %s", errBand, band,
			survey.name)
	}
	if math.IsNaN(magnitude) || math.IsInf(magnitude, eitherSign) {
		return nil, fmt.Errorf("%w: %v", errMagnitude, magnitude)
	}
	if !(magnitudeError >= nothing) ||
		math.IsInf(magnitudeError, eitherSign) {
		return nil, fmt.Errorf("%w: an error of %v", errMagnitude,
			magnitudeError)
	}
	density := zero * math.Pow(pogsonFactor, -magnitude/pogsonStep)
	if math.IsInf(density, eitherSign) {
		return nil, fmt.Errorf("%w: %v in %s", errBright, magnitude,
			band)
	}
	slope := math.Log(pogsonFactor) / pogsonStep
	return &Flux{density: density, sigma: density * slope *
		magnitudeError}, nil
}
