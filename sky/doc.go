// Package sky provides time scales, calendars, observing sites, and sky
// coordinates. It builds on measure; doc/sky.md describes the wider design.
//
// An Instant stores a modified Julian day and its fraction separately to
// preserve resolution. It supports UTC, TAI, TT, TDB, UT1, and GPS, with
// UTC leap seconds from the IERS table.
//
// For a small field, Tangent gives a gnomonic projection with east and north
// Offsets. Ellipse describes positional uncertainty on that plane.
package sky
