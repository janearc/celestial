// Package celestial holds the physics of the sky in one place, for the
// ray tracer, the catalogue readers and the gravitational-wave work alike.
//
// Package celestial itself holds nothing yet. Its packages each hold one
// subject:
//   - units and constants, each cited as defined or measured
//   - time and its scales, frames on the sky and places on Earth
//   - spectra, light and the atmosphere
//   - stars, and gravitation from Newton's to a Kerr hole's
//   - the file formats the data comes in: HDF5 now, FITS to come
//   - LIGO and Virgo's events; their strain, read through hdf5, to come
//
// Exported quantities are in SI, and a package that works in natural
// units inside converts at its edge, both ways. Three do not yet: hole
// works in units of its hole's mass, as relativity does, and radiation
// and bands take nanometres and electronvolts, as libtheme-css's did.
package celestial
