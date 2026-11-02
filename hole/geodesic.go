package hole

import "math"

// Ray stores spacetime position and covariant momentum. Light has null
// momentum; the massive-body model uses timelike unit mass.
type Ray struct{ position, momentum Vec4 }

// NewRay stores a position and covariant momentum without validating their
// physical relation. Nonfinite state is allowed to propagate as an invalid ray;
// March reports it as stuck.
func NewRay(position, momentum Vec4) Ray {
	return Ray{position: position, momentum: momentum}
}

// Position returns (t, x, y, z).
func (ray Ray) Position() Vec4 { return ray.position }

// Momentum returns the covariant momentum.
func (ray Ray) Momentum() Vec4 { return ray.momentum }

// Space returns the spatial position.
func (ray Ray) Space() Vec3 { return ray.position.Space() }

// Fate records why a march stopped. It belongs to the proposed move of
// rendering policy to Slayer; see the package documentation.
type Fate int

// The fates.
const (
	Escaped  Fate = iota // past the far radius, and straight from there
	Captured             // not coming back: see March
	// The step budget ran out, or the ray stopped being a number
	Stuck
)

// limits caches the radius thresholds used to classify one ray before stepping
// begins.
type limits struct {
	horizon float64 // inside it an arriving ray may be found hovering
	// inside it a ray still going inward is not coming back
	noReturn float64
}

// limitsOf is the limits for a ray.
func (hole Hole) limitsOf(point, momentum Vec4) limits {
	return limits{
		horizon:  hole.Horizon(),
		noReturn: hole.noReturn(point, momentum),
	}
}

// noReturn finds the photon-orbit radius associated with the ray’s conserved
// Lz/E. An inward ray inside that radius cannot turn back outward. Bardeen’s
// spherical-orbit relation is
//
//	Lz/E = -(r³ - 3Mr² + A²r + A²M) / (A(r - M)).
//
// With r = M+y, solve the positive root of
//
//	y³ + (A² + A Lz/E - 3M²)y - 2M(M² - A²) = 0.
//
// The cubic is negative at zero and convex above it. Newton iteration from
// above the positive root approaches without crossing and stops when the
// iterate no longer decreases.
//
// The prograde equatorial orbit bounds all spherical photon orbits from below.
// It also serves when energy is zero and Lz/E is undefined. At zero spin the
// result is always 3M.
//
// For a given Lz/E, turning points lie outside the matching orbit for rays
// arriving from outside, or inside for rays originating near the hole. Once
// inside and still moving inward, no outward turn remains.
func (hole Hole) noReturn(point, momentum Vec4) float64 {
	innermost := hole.PhotonOrbit(true)
	turning := (point.x*momentum.y - point.y*momentum.x) / -momentum.time
	if math.IsInf(turning, eitherInfinity) || math.IsNaN(turning) {
		return innermost
	}
	spinSquared, massSquared := hole.spin*hole.spin, hole.mass*hole.mass
	linear := spinSquared + hole.spin*turning - cubicThree*massSquared
	constant := -cubicTwo * hole.mass * (massSquared - spinSquared)
	// The root is below the root of the linear term's size plus the cube
	// root of the constant's
	y := math.Sqrt(math.Abs(linear)) + math.Cbrt(-constant)
	for {
		cubic := y*y*y + linear*y + constant
		next := y - cubic/(cubicDegree*y*y+linear)
		if !(next < y) {
			break
		}
		y = next
	}
	return math.Max(innermost, hole.mass+y)
}

// goingInward tests whether spatial velocity decreases Kerr radius, using the
// gradient proportional to (x r², y r², z(r² + A²)).
func (hole Hole) goingInward(point, velocity Vec4, radius float64) bool {
	radiusSquared := radius * radius
	return radiusSquared*(point.x*velocity.x+point.y*velocity.y)+
		(radiusSquared+hole.spin*hole.spin)*point.z*velocity.z < nothing
}

// Hamiltonian returns H = g^{ab}p_a p_b / 2. It is conserved along a geodesic:
// zero for light and -1/2 for a unit-mass body.
func (hole Hole) Hamiltonian(ray Ray) float64 {
	return hamiltonianHalf * Dot(ray.momentum, hole.Velocity(ray))
}

// Energy returns conserved energy, minus the time component of momentum.
func (hole Hole) Energy(ray Ray) float64 { return -ray.momentum.time }

// Lz returns conserved angular momentum about the spin axis.
func (hole Hole) Lz(ray Ray) float64 {
	return ray.position.x*ray.momentum.y - ray.position.y*ray.momentum.x
}

// Carter returns the conserved Carter constant, alongside energy and Lz.
// Kerr–Schild and Boyer–Lindquist coordinates share z = R cos(theta):
//
//	Q = p_theta² + cos²(theta) *
//	    (A²(mu² - E²) + Lz² / sin²(theta)).
//
// Here mu² = -2H: zero for light and one for a unit-mass body. At zero spin Q
// is total angular momentum squared minus Lz². It is NaN on the disc inside the
// ring, where R is zero.
func (hole Hole) Carter(ray Ray) float64 {
	position := ray.Space()
	radius := hole.Radius(position)
	cosTheta := position.z / radius
	energy := hole.Energy(ray)
	massSquared := massSquaredPerH * hole.Hamiltonian(ray)
	return hole.acrossAxisSquared(position, ray.momentum, radius) +
		cosTheta*cosTheta*hole.spin*hole.spin*
			(massSquared-energy*energy)
}

// acrossAxisSquared is the square of the angular momentum across the spin axis,
// for a momentum at a point of known radius: Lx squared plus Ly squared at zero
// spin, and with spin the first and last terms of Carter,
//
//	p_theta^2 + cos^2 theta Lz^2 / sin^2 theta
//
// It is worked as the squared length of one vector in the xy plane, so no angle
// is taken, nothing is divided by sin theta, and the spin axis is a point like
// any other. With spin a geodesic does not keep it; Carter is the part it
// keeps.
func (hole Hole) acrossAxisSquared(
	position Vec3, momentum Vec4, radius float64,
) float64 {
	radiusSpinSum := radius*radius + hole.spin*hole.spin
	cosTheta := position.z / radius
	around := radius * momentum.z
	acrossX := cosTheta*radiusSpinSum*momentum.x - around*position.x
	acrossY := cosTheta*radiusSpinSum*momentum.y - around*position.y
	return (acrossX*acrossX + acrossY*acrossY) / radiusSpinSum
}

// Velocity raises covariant momentum to give dx/dlambda along the geodesic.
func (hole Hole) Velocity(ray Ray) Vec4 {
	return hole.Raise(ray.Space(), ray.momentum)
}
