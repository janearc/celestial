// Package radiation describes light by wavelength, from gamma rays to radio,
// and converts between nanometres, metres, frequency, and photon energy.
//
// Named ranges follow ISO 21348:2007 and the Fermi mission’s coverage, as
// celestial’s research ledger holds them. The standard is sold, so its ends
// are read in its published first pages to clause 6.5 and in its proof past
// that; the ledger says where each was read, and range comments, overlaps.
//
// Profile maps wavelength intervals for false-colour use. The resulting colour
// still belongs to libtheme; this package determines where each wavelength
// goes, not how it looks on a screen.
package radiation
