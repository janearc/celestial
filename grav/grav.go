// Package grav describes bodies through their gravitational mass. The current
// Body stores mass and GM, enough for point-mass calculations.
//
// Composition, age, spin, and radii derived from published structural models
// remain the design in doc/grav.md, not additional implemented constructors.
package grav

import (
	"fmt"

	"github.com/janearc/celestial/measure"
)

// Body stores a point mass and its gravitational parameter GM.
type Body struct {
	mass measure.Mass
	gm   measure.Length
}

// Point constructs a body known only by mass, as a point-mass orbit sees it.
// The mass must be positive and finite; invalid input returns no body and an
// error.
func Point(mass measure.Mass) (*Body, error) {
	if !mass.Positive() {
		return nil, fmt.Errorf("grav: a body of %v suns",
			mass.InSolarMasses())
	}
	return newBody(mass), nil
}

// newBody caches GM for a mass that has already been validated.
func newBody(mass measure.Mass) *Body {
	return &Body{mass: mass, gm: measure.GM(mass)}
}

// Mass returns the body's mass.
func (body Body) Mass() measure.Mass { return body.mass }

// GM returns mass times G as a length in natural units: half the Schwarzschild
// horizon radius for that mass.
func (body Body) GM() measure.Length { return body.gm }
