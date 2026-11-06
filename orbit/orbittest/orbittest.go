// Package orbittest is what orbit's tests and gpu/orbit's share: a
// centre to orbit, the closed forms of a point mass's orbits, and how
// well a run of steps resolves one.
package orbittest

import (
	"math"

	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/measure"
)

// The package's own numbers, each defined.
const (
	fullTurn = 2 * math.Pi // defined: a turn is two pi radians
	whole    = 1.0         // defined: the whole of a share
	halves   = 2           // defined: a period has two halves
	kinetic  = 2           // defined: the half in half the speed squared
)

// Centre is a point mass of so many suns. It panics on a mass no point
// can have, since a test with no centre has nothing to run.
func Centre(suns float64) grav.Body {
	body, err := grav.Point(measure.SolarMasses(suns))
	if err != nil {
		panic(err)
	}
	return *body
}

// Period is the period of an orbit of a semi-major axis about a gm, in
// units of time.
func Period(gm, axis float64) float64 {
	return fullTurn * math.Sqrt(axis*axis*axis/gm)
}

// Half is half a period.
func Half(period float64) float64 { return period / halves }

// Circular is the speed of a circular orbit at a radius about a gm.
func Circular(gm, radius float64) float64 { return math.Sqrt(gm / radius) }

// Apocentre is an orbit's farthest point from a gm and its speed there,
// for a semi-major axis and an eccentricity.
func Apocentre(gm, axis, eccentricity float64) (far, speed float64) {
	far = axis * (whole + eccentricity)
	return far, math.Sqrt(gm * (whole - eccentricity) / far)
}

// Pericentre is an orbit's nearest point to the centre and its speed
// there, for a semi-major axis and an eccentricity.
func Pericentre(gm, axis, eccentricity float64) (near, speed float64) {
	near = axis * (whole - eccentricity)
	return near, math.Sqrt(gm * (whole + eccentricity) / near)
}

// Energy is a star's energy for each unit of its mass in a point mass's
// field, in kilometres per second squared, at a speed and a distance.
func Energy(gm, speed, distance float64) float64 {
	moving, pulled := Energies(gm, speed, distance)
	return moving - pulled
}

// Energies is the two parts of that energy, each positive: the energy
// of the star's motion, and the size of the energy of its place in the
// field. Their difference is the energy.
//
// Their sum is what a drift in the energy is held against: the two can
// nearly cancel while each is large, and a drift small against either
// is then large against the difference.
func Energies(gm, speed, distance float64) (moving, pulled float64) {
	return speed * speed / kinetic, gm / distance
}

// Size is the length of a triple.
func Size(x, y, z float64) float64 { return math.Sqrt(x*x + y*y + z*z) }

// Shares is the step's shares of a pass about a gm, at a speed and a
// distance, for steps of a length: how far the star moves in one step
// and how far the pull moves it over one step squared, each as a share
// of the distance.
//
// It is what orbit's Pass.Steps reports, from the point mass's closed
// form.
func Shares(gm, speed, near, dt float64) (moved, pulled float64) {
	return speed * dt / near, dt * dt * gm / (near * near * near)
}

// Resolved is how well steps of a length resolve an orbit about a gm at
// its nearest pass: a factor times the square of the step's share of the
// pass in the star's motion plus its share in the pull, and a floor
// under it. The leapfrog's errors are of this size.
func Resolved(gm, speed, near, dt, factor, floor float64) float64 {
	moved, pulled := Shares(gm, speed, near, dt)
	return Room(moved, pulled, factor, floor)
}

// Room is how well a pass with the step's shares is resolved, as
// Resolved measures it, for any field: a factor times the square of the
// share in the motion plus the share in the pull, and a floor under it.
// It is what a pass's own shares from Pass.Steps give.
func Room(moved, pulled, factor, floor float64) float64 {
	return factor*(moved*moved+pulled) + floor
}

// Within says whether a number is within room of another.
func Within(got, want, room float64) bool {
	return math.Abs(got-want) <= room
}
