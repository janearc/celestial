// Package gw is gravitational waves as the LIGO, Virgo and Kagra
// detectors publish them through the gravitational wave open science
// centre, called the centre here: the events of its catalogues, each
// with its estimates and their credible intervals, read into si units.
//
// gw knows the centre's format, not its parameters' meanings: an event
// holds every estimate the centre gives under the centre's own name for
// it, and a caller asks for one by that name. A value the centre leaves
// blank is absent, never nought, and a bound it leaves blank is unknown.
//
// Units arrive as strings, M_sun or Mpc; each is turned into si as it is
// read, by a table of the strings the centre writes, and a string not in
// the table is refused. Time is the event's gps time, as a sky.Instant.
package gw
