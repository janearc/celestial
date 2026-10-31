// Package grav is bodies, and the gravity between them: the gravBody
// of the engineering guide. doc/grav.md is its contract.
//
// Every value grav takes in was made by a constructor that refused what
// it couldn't be, and every value it hands back is reached only through
// its methods.
//
// This is the floor: a body known only by its mass, which is all an
// orbit needs. The full door, NewBody, comes with the tables it needs.
package grav

import (
	"fmt"

	"github.com/janearc/celestial/measure"
)

// Body is a body under gravity
type Body struct {
	mass measure.Mass
	gm   measure.Length
}

// Point is a body known only by its mass, the way an orbit sees it. It
// refuses a mass not above nought, or not finite, and then gives no body.
func Point(mass measure.Mass) (*Body, error) {
	if !mass.Positive() {
		return nil, refuse("a body of %v suns", mass.InSolarMasses())
	}
	return newBody(mass), nil
}

// newBody builds a body that has passed the door, working out its GM
// once
func newBody(mass measure.Mass) *Body {
	return &Body{mass: mass, gm: measure.GM(mass)}
}

// Mass is the body's mass
func (body Body) Mass() measure.Mass { return body.mass }

// GM is the body's mass times G: in natural units a length, half the
// horizon radius of a hole of that mass
func (body Body) GM() measure.Length { return body.gm }

// refuse is a constructor's error, saying what it would not make
func refuse(format string, about ...any) error {
	return fmt.Errorf("grav: "+format, about...)
}
