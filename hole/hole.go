package hole

import (
	"fmt"
	"math"
)

// Hole is a black hole: its mass M and its spin A, the angular momentum
// per unit mass, from nought for schwarzschild up to just under M. The
// spin axis is z, and the hole turns from x toward y.
//
// Only NewHole makes one, and it makes only a hole the other methods are
// right for.
type Hole struct{ mass, spin float64 }

// the masses hole's arithmetic holds. radius and kernel take the fourth
// power of a point's distance, and a hole must hold the points of its
// own horizon, which lies between the mass and twice it.
//
// So the fourth power of twice the mass must not pass the largest
// float64, and the fourth power of the mass must not fall below the
// least normal one, where it would lose its digits.
var (
	heaviest = math.Pow(math.MaxFloat64, fourthRoot) / horizonInMasses
	lightest = math.Pow(leastNormal, fourthRoot)
)

// The numbers the masses above come from.
const (
	fourthRoot  = 0.25      // defined: a fourth root is the power a quarter
	leastNormal = 0x1p-1022 // defined by float64: its least normal number
	// horizonInMasses is the farthest a horizon lies, in masses. Defined
	// by kerr's metric: the horizon is M + root(M² - a²), and 2M at a = 0.
	horizonInMasses = 2
)

// NewHole returns a kerr hole of the given mass and spin, both lengths
// in units where G = c = 1: spin is a = J/M, the angular momentum per
// unit mass, so a spin and a mass compare directly. returns nil and error
// for:
//   - a mass outside what hole's arithmetic holds, about 1.2e-77 to
//     5.8e76 in the caller's unit of length: past those, the hole could
//     not place the points of its own horizon; see heaviest and lightest
//   - a spin below zero: a hole turning the other way is this one seen
//     from below, so turn z about instead
//   - a spin at or past the mass: at a = M the inner and outer horizons
//     meet, and past it there is no horizon and no black hole
//
// NaN, either infinity, and a mass of zero or below fall outside these
// ranges, so they are refused with them.
//
// a spin of zero is a hole: no real one stands still, but it is the
// limit the textbook's closed forms are checked against.
//
// a point farther than about 1.2e77 from the hole overflows radius,
// whatever the hole, since radius takes the fourth power of its
// distance. NewHole cannot guard that; the caller's units can.
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

// Mass is the hole's mass M, which is also its unit of length.
func (hole Hole) Mass() float64 { return hole.mass }

// Spin is the hole's spin A, the angular momentum per unit mass, in the
// same unit as the mass.
func (hole Hole) Spin() float64 { return hole.spin }

// Radius is the kerr-schild radius at a point: the schwarzschild r at
// nought spin, and the oblate coordinate that reaches the ring singularity
// with spin.
//
// On the disc the ring spans it is nought, and within a hair of that
// disc it rounds to nought as well, being there the difference of two
// numbers that agree to all their digits. No march comes there: a body
// is stopped at the inner horizon and light at the outer.
func (hole Hole) Radius(position Vec3) float64 {
	return hole.radius(position.x, position.y, position.z)
}

// radius is Radius of a point given as three numbers, which is how the march
// holds it.
func (hole Hole) radius(x, y, z float64) float64 {
	positionSquaredSum := x*x + y*y + z*z
	spinSquared := hole.spin * hole.spin
	discriminant := positionSquaredSum - spinSquared
	rootedSquare := math.Sqrt(discriminant*discriminant +
		radiusFour*spinSquared*z*z)
	// within a hair of the disc the sum is the difference of two numbers
	// that agree to all their digits, and rounding may leave it below
	// nought: that is nought, and not the root of a negative
	radiusSquared := radiusHalf * (discriminant + rootedSquare)
	return math.Sqrt(math.Max(nothing, radiusSquared))
}

// kernel is the kerr-schild scalar and the null covector at a
// point: the metric is flat plus scalar * vector * vector.
//
// On the disc inside the ring the radius is nought and the kernel has
// no value; it answers flat space there, which nothing should lean on.
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

// Inverse is the contravariant metric at a point: flat minus scalar *
// vector * vector with vector raised, which the kerr-schild form gives
// exactly.
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

// Horizon is the outer horizon's radius: 2M at nought spin.
func (hole Hole) Horizon() float64 {
	return hole.mass + hole.horizonSplit()
}

// innerHorizon is the inner horizon's radius: nought at nought spin,
// where it is the singularity, and up to M at full spin.
func (hole Hole) innerHorizon() float64 {
	return hole.mass - hole.horizonSplit()
}

// horizonSplit is how far the two horizons lie either side of M.
func (hole Hole) horizonSplit() float64 {
	return math.Sqrt(math.Max(nothing,
		hole.mass*hole.mass-hole.spin*hole.spin))
}

// PhotonSphere is the radius of the circular photon orbit of a hole
// with no spin, 3M; with spin the orbits split prograde and retrograde
// and this is the schwarzschild value. PhotonOrbit is the two.
func (hole Hole) PhotonSphere() float64 {
	return photonSphereFactor * hole.mass
}

// PhotonOrbit is the radius of the circular photon orbit in the equator,
// prograde or retrograde: 3M at nought spin, down to M prograde and up
// to 4M retrograde at full spin, by bardeen, press and teukolsky's
// equation 2.18, read on 2026-10-31 (numbers.go cites the paper).
//
// Every other photon orbit, the ones that leave the equator, lies between
// the two.
func (hole Hole) PhotonOrbit(prograde bool) float64 {
	against := hole.spin / hole.mass
	if prograde {
		against = -against
	}
	angle := orbitTurn * math.Acos(against) / orbitThird
	return orbitScale * hole.mass * (orbitOne + math.Cos(angle))
}

// Shadow is the radius of the shadow a hole with no spin casts on the
// sky of a far observer, root twenty-seven M: the impact parameter at
// which a ray is just captured.
func (hole Hole) Shadow() float64 {
	return math.Sqrt(shadowSquared) * hole.mass
}

// ISCO is the innermost stable circular orbit, prograde or retrograde:
// 6M at nought spin, down to M prograde and up to 9M retrograde at
// full spin, by bardeen, press and teukolsky's equation 2.21, read on
// 2026-10-31 (numbers.go cites the paper).
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
