// Package gw reads LIGO, Virgo, and Kagra catalogue events from the
// Gravitational Wave Open Science Center, called the centre here. Estimates and
// credible intervals are converted to SI as they are read; event times become
// sky.Instant values on the GPS scale.
//
// Parameters retain the centre’s names. A caller chooses which estimates to
// interpret; the reader does not guess their meaning. Missing values remain
// absent rather than becoming zero, and missing interval bounds remain unknown.
//
// Unit strings such as M_sun and Mpc use explicit conversions. Unknown unit
// strings return an error instead of leaving a value in ambiguous units.
package gw
