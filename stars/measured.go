package stars

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
)

// What stars takes on trust beyond nature's own constants, which are
// measure's.

// protonsPerHelium is how many hydrogen nuclei become one helium nucleus
// when a star burns hydrogen. Defined by the reaction.
const protonsPerHelium = 4

// burnedShareName is the name in celestial's research data of the share
// of a star's mass that becomes helium on the main sequence.
const burnedShareName = "burned-share"

// burnedShare is the chosen fraction of stellar mass converted to helium during
// main-sequence burning, about one tenth.
//
// Above roughly twenty solar masses the fixed fraction underestimates core
// size, shortening model lifetimes by up to a factor of four. Published
// lifetime tracks are the intended replacement.
func burnedShare() float64 { return research.Value(burnedShareName) }

// fusionYield returns the fraction of hydrogen mass released by fusion: (four
// hydrogen atoms minus one helium atom) divided by the four hydrogen atoms.
// Neutrinos carry away some of that energy.
func fusionYield() float64 {
	burned := measure.Hydrogen1().Times(protonsPerHelium)
	return burned.Minus(measure.Helium4()).Over(burned)
}
