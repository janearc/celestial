// Package radiation describes light by wavelength, from gamma rays to radio,
// and converts between nanometres, metres, frequency, and photon energy.
//
// Named ranges follow ISO 21348 terminology and the Fermi mission’s coverage.
// The published ISO standard was not read. The ranges were checked against
// Tobiska and Nusinov’s account of its draft; range comments record overlaps
// and differences from the inherited libtheme-css values.
//
// Profile maps wavelength intervals for false-colour use. The resulting colour
// still belongs to libtheme; this package determines where each wavelength
// goes, not how it looks on a screen.
package radiation
