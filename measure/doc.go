// Package measure is typed quantities in natural units: the speed of
// light, the reduced planck constant and boltzmann's constant are all
// one, so mass, temperature and energy are one kind of thing, counted in
// MeV, and length and time are its inverse.
//
// Every quantity is a type, made from the unit it comes in and read back
// out in the unit it goes out in. A caller hands in si units and reads
// si units back; the natural units are held inside. A bare number never
// crosses a package line, so a mass can't be handed in where an age goes.
//
// exact.go turns the si's units into natural units through the si's own
// defining constants, which are exact. measured.go reads what is
// measured from celestial's research data. laws.go is what follows.
package measure
