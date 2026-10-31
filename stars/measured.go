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

// burnedShare is the share of a star's mass that becomes helium while it
// burns on the main sequence: about a tenth, picked from stellar models.
//
// It is too small past about twenty suns, whose burning cores are
// larger, so heavy stars die here up to four times too young. A
// published table of lifetimes replaces it.
func burnedShare() float64 { return research.Value(burnedShareName) }

// fusionYield is the share of hydrogen's mass that becomes energy when
// it burns to helium: what four hydrogens weigh, less one helium, over
// the four. Neutrinos carry a little of it away.
func fusionYield() float64 {
	burned := measure.Hydrogen1().Times(protonsPerHelium)
	return burned.Minus(measure.Helium4()).Over(burned)
}
