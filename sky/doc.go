// Package sky is where things are, seen from places, at instants,
// in frames: the sun over a hillside, a star through the sun's lens, a
// merger's waves at two detectors. doc/sky.md is its design.
//
// A small field around a source is a Tangent plane, the gnomonic
// projection, with positions on it as east and north Offsets; a
// position's error is an Ellipse whose outline lies on that plane.
//
// Package sky is built on measure.
package sky
