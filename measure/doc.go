// Package measure represents physical quantities in natural units with c,
// reduced Planck’s constant, and Boltzmann’s constant equal to one. Mass,
// temperature, and energy are stored in MeV; length and time in MeV^-1.
//
// Public constructors and accessors use named units, so callers can work in SI
// while calculations retain natural units inside. Distinct types keep a mass
// from being passed where an age belongs.
//
// exact.go derives conversions from SI defining constants. measured.go loads
// measured values and defines nominal solar references. laws.go contains the
// physical relationships built on those quantities.
package measure
