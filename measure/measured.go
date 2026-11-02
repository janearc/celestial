package measure

import (
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
)

// What is measured, not defined, by its name in celestial's research
// data. Each is turned into natural units here, so nothing else ever sees
// the published unit.
const (
	fineStructure    = "fine-structure"
	electronMeV      = "electron-mev"
	protonMeV        = "proton-mev"
	daltonMeV        = "dalton-mev"
	hydrogen1Daltons = "hydrogen-1-daltons"
	helium4Daltons   = "helium-4-daltons"
	cosmosYears      = "cosmos-years"
)

// The IAU's nominal solar values, 2015 resolution B3. Each is defined: the
// resolution fixes them as exact conversion constants, whatever the Sun is
// measured to be. Read on 2026-10-31 in its solar conversion constants, from
// the copy at arXiv:1510.07674; game run extract prints them.
const (
	// sunGravitySI is the nominal GM of the Sun, in m3 s-2. Defined.
	sunGravitySI = 1.3271244e20
	// sunRadiusSI is the nominal radius of the Sun, in m. Defined.
	sunRadiusSI = 6.957e8
	// sunLuminositySI is the nominal luminosity of the Sun, in W. Defined.
	sunLuminositySI = 3.828e26
)

// sunGM is the Sun's GM in natural units, where it is a length: half the radius
// of the horizon a hole of the Sun's mass would have.
const sunGM = sunGravitySI * metre * metre * metre / (second * second)

// gravitation is G in natural units, MeV-2: one over the Planck mass squared. A
// body's mass found from its orbit carries G's uncertainty, so a body keeps its
// measured GM.
func gravitation() float64 {
	return si.Gravitation() * metre * metre * metre /
		(kilogram * second * second)
}

// Electron is the electron's mass.
func Electron() Mass { return Mass{research.Value(electronMeV)} }

// Proton is the proton's mass.
func Proton() Mass { return Mass{research.Value(protonMeV)} }

// Hydrogen1 is the mass of a hydrogen 1 atom, its electron and all.
func Hydrogen1() Mass { return Daltons(research.Value(hydrogen1Daltons)) }

// Helium4 is the mass of a helium 4 atom, its electrons and all.
func Helium4() Mass { return Daltons(research.Value(helium4Daltons)) }

// Sun is the IAU's nominal Sun: its GM, radius and luminosity as the IAU fixes
// them, and its mass from its GM over G.
func Sun() Nominal {
	return Nominal{gm: Length{sunGM}, radius: Length{sunRadiusSI * metre},
		luminosity: Power{sunLuminositySI * watt},
		mass:       Mass{sunGM / gravitation()}}
}

// Cosmos is how old the universe is.
func Cosmos() Duration {
	return Duration{research.Value(cosmosYears) * year}
}

// Nominal is a star's measures as the IAU fixes them for reference: its
// GM, radius and luminosity, and the mass that follows.
type Nominal struct {
	gm         Length
	radius     Length
	luminosity Power
	mass       Mass
}

// GM is the star's mass times G, as the IAU fixes it.
func (star Nominal) GM() Length { return star.gm }

// Radius is the star's radius.
func (star Nominal) Radius() Length { return star.radius }

// Luminosity is the star's luminosity.
func (star Nominal) Luminosity() Power { return star.luminosity }

// Mass is the star's mass, its GM over G.
func (star Nominal) Mass() Mass { return star.mass }
