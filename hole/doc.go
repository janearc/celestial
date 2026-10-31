// Package hole is the maths of a black hole for a renderer: a kerr
// hole in kerr-schild form, null and timelike geodesics marched by
// hamilton's equations, the quantities a geodesic keeps, the redshift
// between two observers, and the frame of a camera falling in.
//
// It imports nothing but the standard library, so the relativity can
// leave; what it does not know is anything about light or pictures.
//
// One file leans toward a renderer, gpu.go: the hole and a camera as
// the float32 constants a shader takes, and a step that reports what a
// ray keeps, to hold a shader's own steps against. There is no GPU in
// it and nothing of a picture.
//
// Units: G and c are one, and a hole of mass M has its lengths in M. The
// signature is minus, plus, plus, plus; a point is (t, x, y, z) and a
// momentum is covariant.
//
// Where its numbers come from. The Kerr-Schild form, the radii of a
// Kerr hole and the runge-kutta step are defined by their formulas, and
// numbers.go says by which.
//
// The march's tolerances are choices, not physics. They are in
// choices.json, built in, each with what it says and who picked it.
//
// The tests' own numbers are in testdata/hole.json, each with where it
// came from.
package hole
