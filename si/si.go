// Package si holds the physical constants celestial uses and the si's
// decimal prefixes. The five constants here, Planck, Speed, Charge,
// Boltzmann and Avogadro, are si defining constants: exact, not
// measured, each typed here once.
//
// A measured constant, such as G, is read from celestial's research data
// by name, through a function here.
package si

import "github.com/janearc/celestial/research"

// The si's defining constants, as the SI Brochure, 9th edition (BIPM,
// 2019), table 1, fixes them.
const (
	// Planck is h, in joule seconds. Defined by the si since 2019.
	Planck = 6.62607015e-34
	// Speed is c, the speed of light, in metres a second. Defined by
	// the si since 1983.
	Speed = 299792458.0
	// Charge is e, the charge of one electron, in coulombs. Defined by
	// the si since 2019.
	Charge = 1.602176634e-19
	// Boltzmann is k, in joules a kelvin. Defined by the si since 2019.
	Boltzmann = 1.380649e-23
	// Avogadro is the number of things in a mole. Defined by the si
	// since 2019.
	Avogadro = 6.02214076e23
)

// Gravitation is G, the newtonian constant of gravitation, in cubic
// metres a kilogram a second squared, as celestial's research data holds
// it.
//
// It is measured, not defined: the 2022 value of the Committee on Data of
// the International Science Council, good to 2.2 parts in 100000.
func Gravitation() float64 {
	return research.Value(gravitation)
}

// gravitation is G's name in the research data.
const gravitation = "gravitation"
