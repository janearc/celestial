// Package celestial brings the physics of the sky together for renderers,
// catalogue readers, and gravitational-wave work. The root package has no
// runtime API; each subject lives in its own package.
//
// The implemented foundations include units and constants, time and sky
// coordinates, spectra and atmospheres, stars, point-mass gravity, orbits
// through a potential (on the gpu too), Kerr geometry, HDF5 reading, and
// the open science centre event estimates. Strain analysis remains planned.
//
// Most public quantities use SI. Packages using natural units convert at the
// boundary. The documented exceptions are hole, which uses geometric units set
// by its mass, and radiation and bands, which retain nanometres and
// electronvolts where inherited from the spectral APIs.
package celestial
