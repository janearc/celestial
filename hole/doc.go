// Package hole supplies black-hole maths for a renderer: Kerr geometry in
// Kerr–Schild form, null and timelike geodesics from Hamilton’s equations,
// conserved quantities, redshift, and the frame of a falling camera.
//
// Only the standard library is imported. The package describes geometry and
// motion, not materials or finished pictures. gpu.go supplies float32 shader
// parameters and CPU reference stepping; it does not execute GPU work.
//
// G and c are one. Mass sets the length scale, coordinates are (t, x, y, z),
// the spin axis is z, the metric signature is (-, +, +, +), and momentum is
// covariant. Formula-derived constants and their sources are in numbers.go.
// Chosen integration tolerances are in choices.json, with their authors.
//
// The intended boundary leaves relativity here and moves rendering policy to
// Slayer. March, PaceIn, Hovering, ClosestApproach, DeflectionAngle, Fate, the
// choices used by Advance and ClassifyRay, choices.json, and gpu.go are part of
// that proposed migration. They remain implemented in this checkout.
//
// Step already advances a geodesic at a caller-supplied interval. Test
// expectations and their provenance live in testdata/hole.json.
package hole
