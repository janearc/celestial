// Package radiation speaks of radiation by its wavelength in nanometres,
// from gamma rays to radio. It moves between wavelength, energy, frequency
// and length in metres.
//
// It names the ranges of radiation under iso 21348's names, and the fermi
// mission's. The iso ranges were never read against the standard, which
// is not open; they are as libtheme-css typed them, and some overlap, as
// each range's doc says.
//
// A Profile maps one range of wavelengths onto another, which is what a
// false color profile is made of.
//
// What the radiation looks like is not here. That is a colour, and colour
// is libtheme's.
package radiation
