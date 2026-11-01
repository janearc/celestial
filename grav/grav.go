// Package grav is bodies under gravity, and the gravity between them.
// doc/grav.md is the plan for what comes next.
//
// For now a body is known only by its mass, which is all an orbit needs.
// A body with matter, an age and a spin waits for the published tables
// that give its size.
package grav

import (
	"fmt"

	"github.com/janearc/celestial/measure"
)

// Body is a body under gravity.
type Body struct {
	mass measure.Mass
	gm   measure.Length
}

// Point returns a body known only by its mass, the way an orbit sees it.
// returns nil and error for a mass that is not finite and above zero.
func Point(mass measure.Mass) (*Body, error) {
	if !mass.Positive() {
		return nil, refuse("a body of %v suns", mass.InSolarMasses())
	}
	return newBody(mass), nil
}

// newBody returns a body of a mass already checked, with its GM worked
// out once.
func newBody(mass measure.Mass) *Body {
	return &Body{mass: mass, gm: measure.GM(mass)}
}

// Mass returns the body's mass.
func (body Body) Mass() measure.Mass { return body.mass }

// GM returns the body's mass times G. In natural units that's a length:
// half the horizon radius of a still hole of that mass.
func (body Body) GM() measure.Length { return body.gm }

// refuse returns an error for something a constructor won't make.
func refuse(format string, about ...any) error {
	return fmt.Errorf("grav: "+format, about...)
}
