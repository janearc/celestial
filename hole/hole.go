package hole

import (
	"fmt"
	"math"
)

// Hole stores mass M and spin A = J/M in geometric length units. The spin axis
// is z, with rotation from x toward y. Use NewHole to validate the range before
// calling the geometric methods.
type Hole struct{ mass, spin float64 }

// The masses hole's arithmetic holds. radius and kernel take the fourth power
// of a point's distance, and a hole must hold the points of its own horizon,
// which lies between the mass and twice it.
//
// So the fourth power of twice the mass must not pass the largest float64, and
// the fourth power of the mass must not fall below the least normal one, where
// it would lose its digits.
var (
	heaviest = math.Pow(math.MaxFloat64, fourthRoot) / horizonInMasses
	lightest = math.Pow(leastNormal, fourthRoot)
)

// The numbers the masses above come from.
const (
	fourthRoot  = 0.25      // defined: a fourth root is the power a quarter
	leastNormal = 0x1p-1022 // defined by float64: its least normal number
	// horizonInMasses is the farthest a horizon lies, in masses. Defined by
	// Kerr's metric: the horizon is M + root(M² - a²), and 2M at a = 0.
	horizonInMasses = 2
)

// NewHole constructs a Kerr hole with mass M and spin A = J/M, both in length
// units with G = c = 1. It requires a finite positive mass within the
// arithmetic bounds, and 0 <= A < M.
//
// Mass limits are about 1.2e-77 to 5.8e76 in the caller’s length unit. The
// fourth powers used by radius and kernel must resolve the hole’s own horizon
// without overflow or loss of normal precision; see lightest and heaviest.
//
// Negative spin is represented by reversing the z axis instead. A = M is
// excluded because the two horizons coincide; above it there is no black-hole
// horizon. Zero spin is allowed as the Schwarzschild limit used by the
// closed-form tests. Invalid inputs return nil and an error.
//
// Coordinates farther than about 1.2e77 overflow the radius calculation
// regardless of mass. Constructor checks cannot prevent that; choose a suitable
// coordinate scale.
func NewHole(mass, spin float64) (*Hole, error) {
	if !(mass >= lightest && mass <= heaviest) {
		return nil, fmt.Errorf("hole: a mass of %v, outside %v to %v",
			mass, lightest, heaviest)
	}
	if !(spin >= nothing && spin < mass) {
		return nil, fmt.Errorf("hole: a spin of %v, outside %v to "+
			"below %v", spin, nothing, mass)
	}
	return &Hole{mass: mass, spin: spin}, nil
}

// Mass returns the geometric mass M, which sets the length scale.
func (hole Hole) Mass() float64 { return hole.mass }

// Spin returns A = J/M in the same length unit as Mass. The dimensionless spin
// is A/M.
func (hole Hole) Spin() float64 { return hole.spin }

// Radius returns the Kerr–Schild radial coordinate: the Schwarzschild radius
// coordinate at zero spin, or the oblate coordinate with spin. It is zero on
// the disc bounded by the ring singularity.
//
// Very near that disc, cancellation can also round the result to zero. The
// marcher avoids relying on this region: bodies stop at the inner horizon,
// while light uses its capture and horizon stopping rules.
func (hole Hole) Radius(position Vec3) float64 {
	return hole.radius(position.x, position.y, position.z)
}

// radius evaluates Radius from separate spatial components used by the
// integration code.
func (hole Hole) radius(x, y, z float64) float64 {
	positionSquaredSum := x*x + y*y + z*z
	spinSquared := hole.spin * hole.spin
	discriminant := positionSquaredSum - spinSquared
	rootedSquare := math.Sqrt(discriminant*discriminant +
		radiusFour*spinSquared*z*z)
	// within a hair of the disc the sum is the difference of two numbers
	// that agree to all their digits, and rounding may leave it below zero:
	// that is zero, and not the root of a negative
	radiusSquared := radiusHalf * (discriminant + rootedSquare)
	return math.Sqrt(math.Max(nothing, radiusSquared))
}

// kernel returns the Kerr–Schild scalar and null covector, with metric = flat +
// scalar * vector * vector. On the disc inside the ring, radius is zero and the
// form is undefined; the implementation returns a flat-space fallback, not a
// physical model of that region.
func (hole Hole) kernel(position Vec3) (scalar float64, vector Vec4) {
	radius := hole.Radius(position)
	spin := hole.spin
	radiusSquared, spinSquared := radius*radius, spin*spin
	denominator := radiusSquared*radiusSquared +
		spinSquared*position.z*position.z
	if denominator == nothing || radius == nothing {
		return nothing, Vec4{time: kernelTime}
	}
	scalar = scalarFactor * hole.mass * radius * radiusSquared / denominator
	radiusSpinSum := radiusSquared + spinSquared
	vector = Vec4{
		time: kernelTime,
		x:    (radius*position.x + spin*position.y) / radiusSpinSum,
		y:    (radius*position.y - spin*position.x) / radiusSpinSum,
		z:    position.z / radius,
	}
	return scalar, vector
}

// flat is the flat metric's entry on the diagonal at a row: minus along
// time and plus along each axis of space.
func flat(row int) float64 {
	if row == timeAxis {
		return timeSign
	}
	return spaceSign
}

// Metric is the covariant metric at a point: flat plus scalar * vector *
// vector.
func (hole Hole) Metric(position Vec3) Mat4 {
	scalar, vector := hole.kernel(position)
	parts := vector.parts()
	var metric Mat4
	for i := range dimensions {
		for column := range dimensions {
			metric[i][column] = scalar * parts[i] * parts[column]
		}
		metric[i][i] += flat(i)
	}
	return metric
}

// Inverse is the contravariant metric at a point: flat minus scalar * vector *
// vector with vector raised, which the Kerr-schild form gives exactly.
func (hole Hole) Inverse(position Vec3) Mat4 {
	scalar, vector := hole.kernel(position)
	raisedVector := Vec4{
		time: -vector.time, x: vector.x, y: vector.y, z: vector.z,
	}
	parts := raisedVector.parts()
	var metric Mat4
	for i := range dimensions {
		for column := range dimensions {
			metric[i][column] = -scalar * parts[i] * parts[column]
		}
		metric[i][i] += flat(i)
	}
	return metric
}

// Lower is a contravariant vector made covariant with the metric at a
// point.
func (hole Hole) Lower(position Vec3, vector Vec4) Vec4 {
	scalar, null := hole.kernel(position)
	return lowered(scalar, null, vector)
}

// Raise is a covariant vector made contravariant with the inverse at a
// point.
func (hole Hole) Raise(position Vec3, covector Vec4) Vec4 {
	scalar, null := hole.kernel(position)
	return raised(scalar, null, covector)
}

// lowered is a vector made covariant by a kernel already in hand: the
// flat metric on it, plus scalar times the null covector times the null
// covector on it. The metric is a flat one and a rank of one more, and
// this uses it without building its sixteen entries.
//
// The null covector's time part is one, always, and is not multiplied
// by here or in raised.
func lowered(scalar float64, null, vector Vec4) Vec4 {
	pull := scalar * (vector.time + null.x*vector.x +
		null.y*vector.y + null.z*vector.z)
	return Vec4{
		time: pull - vector.time,
		x:    pull*null.x + vector.x,
		y:    pull*null.y + vector.y,
		z:    pull*null.z + vector.z,
	}
}

// raised is a covector made contravariant by a kernel already in hand:
// the flat inverse on it, less scalar times the null vector times the
// null vector on it, the null vector being the null covector with its
// time part turned about.
func raised(scalar float64, null, covector Vec4) Vec4 {
	pull := scalar * (covector.x*null.x + covector.y*null.y +
		covector.z*null.z - covector.time)
	return Vec4{
		time: pull - covector.time,
		x:    covector.x - pull*null.x,
		y:    covector.y - pull*null.y,
		z:    covector.z - pull*null.z,
	}
}

// Horizon returns the outer horizon radius, 2M at zero spin.
func (hole Hole) Horizon() float64 {
	return hole.mass + hole.horizonSplit()
}

// innerHorizon is the inner horizon's radius: zero at zero spin, where it is
// the singularity, and up to M at full spin.
func (hole Hole) innerHorizon() float64 {
	return hole.mass - hole.horizonSplit()
}

// horizonSplit is how far the two horizons lie either side of M.
func (hole Hole) horizonSplit() float64 {
	return math.Sqrt(math.Max(nothing,
		hole.mass*hole.mass-hole.spin*hole.spin))
}

// PhotonSphere returns the Schwarzschild reference radius 3M. For a spinning
// hole, PhotonOrbit gives the distinct prograde and retrograde equatorial
// radii.
func (hole Hole) PhotonSphere() float64 {
	return photonSphereFactor * hole.mass
}

// PhotonOrbit is the radius of the circular photon orbit in the equator,
// prograde or retrograde: 3M at zero spin, down to M prograde and up to 4M
// retrograde at full spin, by Bardeen, press and teukolsky's equation 2.18,
// read on 2026-10-31 (numbers.go cites the paper).
//
// Every other photon orbit, the ones that leave the equator, lies between the
// two.
func (hole Hole) PhotonOrbit(prograde bool) float64 {
	against := hole.spin / hole.mass
	if prograde {
		against = -against
	}
	angle := orbitTurn * math.Acos(against) / orbitThird
	return orbitScale * hole.mass * (orbitOne + math.Cos(angle))
}

// Shadow returns the Schwarzschild shadow impact parameter √27 M for a distant
// observer. It is not a rotating-hole shadow contour.
func (hole Hole) Shadow() float64 {
	return math.Sqrt(shadowSquared) * hole.mass
}

// ISCO is the innermost stable circular orbit, prograde or retrograde: 6M at
// zero spin, down to M prograde and up to 9M retrograde at full spin, by
// Bardeen, press and teukolsky's equation 2.21, read on 2026-10-31 (numbers.go
// cites the paper).
func (hole Hole) ISCO(prograde bool) float64 {
	mass := hole.mass
	normalizedSpin := hole.spin / mass
	spinSquared := normalizedSpin * normalizedSpin
	cubes := math.Cbrt(iscoOne+normalizedSpin) +
		math.Cbrt(iscoOne-normalizedSpin)
	term1 := iscoOne + math.Cbrt(iscoOne-spinSquared)*cubes
	term2 := math.Sqrt(iscoThree*spinSquared + term1*term1)
	discriminant := math.Sqrt(
		(iscoThree - term1) * (iscoThree + term1 + iscoTwo*term2),
	)
	if prograde {
		return mass * (iscoThree + term2 - discriminant)
	}
	return mass * (iscoThree + term2 + discriminant)
}
