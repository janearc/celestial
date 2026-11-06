package orbit

import (
	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/si"
)

// Kepler is the pull of one point mass at the centre.
type Kepler struct{ gm float64 }

// NewKepler is the pull of a body at the centre.
func NewKepler(body grav.Body) Kepler {
	// the body's gm is in natural units, where c is one: a length that
	// is gm over c squared. so gm in metres cubed per second squared is
	// that length times c squared
	metres := body.GM().InMetres() * si.Speed * si.Speed
	return Kepler{gm: metres / (kiloparsec * kilometre * kilometre)}
}

// GM is the body's mass times the gravitational constant, in kiloparsecs
// times kilometres per second squared.
func (field Kepler) GM() float64 { return field.gm }

// Pull is the inverse-square pull toward the centre. At the centre
// itself it is not a number, and an orbit through it runs away.
func (field Kepler) Pull(at Place) Pull {
	distance := at.at.norm()
	cubed := distance * distance * distance
	return Pull{at.at.times(-field.gm / cubed)}
}
