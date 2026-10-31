package hole

import "math"

// Ray is a geodesic's state: where it is in spacetime and its covariant
// momentum. For light the momentum is null; for a body it is timelike
// with unit mass.
type Ray struct{ position, momentum Vec4 }

// NewRay is a geodesic at a point in spacetime with a covariant momentum.
// Any two vectors make one; a ray whose parts are not numbers is stuck at
// once when marched.
func NewRay(position, momentum Vec4) Ray {
	return Ray{position: position, momentum: momentum}
}

// Position is where the ray is in spacetime, (t, x, y, z).
func (ray Ray) Position() Vec4 { return ray.position }

// Momentum is the ray's covariant momentum.
func (ray Ray) Momentum() Vec4 { return ray.momentum }

// Space is the ray's point in space.
func (ray Ray) Space() Vec3 { return ray.position.Space() }

// Fate is how a march ended.
type Fate int

// The fates.
const (
	Escaped  Fate = iota // past the far radius, and straight from there
	Captured             // not coming back: see March
	// the step budget ran out, or the ray stopped being a number
	Stuck
)

// limits is the radii a march decides a ray's fate by, worked out once
// for the ray before it is stepped.
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

// noReturn is the radius inside which a ray that is still going inward
// is never coming back out: the radius of the photon orbit that has the
// ray's own angular momentum about the axis for its energy.
//
// Light can circle a spinning hole at any radius between the prograde
// and retrograde orbits of the equator, on an orbit that leaves the
// equator, and at each radius only with one ratio of Lz to E:
//
//	Lz/E = -(r^3 - 3 M r^2 + A^2 r + A^2 M) / (A (r - M))
//
// by bardeen. A ray keeps that ratio, and whatever else it has, the places
// it can turn about lie either side of the orbit with its ratio: outside it
// if it came from outside, and then it is not inside going inward; inside it
// if it came from the hole, and then going inward it is past its turn.
//
// So inside that radius and going inward there is no turn ahead. The
// radius is M + y, with y the one root above nought of the cubic the
// equation makes,
//
//	y^3 + (A^2 + A Lz/E - 3 M^2) y - 2 M (M^2 - A^2) = 0
//
// which is below nought at nought and convex above it, so newton's
// method from above the root falls onto it and does not pass, and it is
// run until it stops falling.
//
// The innermost photon orbit, the prograde one, is inside all the
// others and bounds the answer below. A ray with no energy has no
// ratio, and the innermost serves for it. At nought spin every answer
// is 3M.
func (hole Hole) noReturn(point, momentum Vec4) float64 {
	innermost := hole.PhotonOrbit(true)
	turning := (point.x*momentum.y - point.y*momentum.x) / -momentum.time
	if math.IsInf(turning, eitherInfinity) || math.IsNaN(turning) {
		return innermost
	}
	spinSquared, massSquared := hole.spin*hole.spin, hole.mass*hole.mass
	linear := spinSquared + hole.spin*turning - cubicThree*massSquared
	constant := -cubicTwo * hole.mass * (massSquared - spinSquared)
	// the root is below the root of the linear term's size plus the cube
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

// goingInward is whether a point moving at a velocity is losing radius:
// the sign of the radius's gradient on the velocity, the gradient being
// along (x r^2, y r^2, z (r^2 + A^2)).
func (hole Hole) goingInward(point, velocity Vec4, radius float64) bool {
	radiusSquared := radius * radius
	return radiusSquared*(point.x*velocity.x+point.y*velocity.y)+
		(radiusSquared+hole.spin*hole.spin)*point.z*velocity.z < nothing
}

// Hamiltonian is hamilton's function, half the inverse metric on the
// momentum: nought along a null ray and minus a half along a body of unit
// mass, kept along any geodesic.
func (hole Hole) Hamiltonian(ray Ray) float64 {
	return hamiltonianHalf * Dot(ray.momentum, hole.Velocity(ray))
}

// Energy is the energy a geodesic keeps, minus the time momentum.
func (hole Hole) Energy(ray Ray) float64 { return -ray.momentum.time }

// Lz is the angular momentum about the spin axis a geodesic keeps.
func (hole Hole) Lz(ray Ray) float64 {
	return ray.position.x*ray.momentum.y - ray.position.y*ray.momentum.x
}

// Carter is the carter constant, which a geodesic keeps beside E and
// Lz. In the angle theta that the kerr-schild coordinates share with
// boyer and lindquist's, z = R cos theta, it is
//
//	Q = p_theta^2 + cos^2 theta (A^2 (mu^2 - E^2) + Lz^2 / sin^2 theta)
//
// with mu^2 minus twice H: nought for light, one for a body of unit
// mass. At nought spin it is the square of the angular momentum less
// Lz squared. On the disc inside the ring, where R is nought, it is not
// a number.
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

// acrossAxisSquared is the square of the angular momentum across the
// spin axis, for a momentum at a point of known radius: Lx squared plus
// Ly squared at nought spin, and with spin the first and last terms of
// Carter,
//
//	p_theta^2 + cos^2 theta Lz^2 / sin^2 theta
//
// It is worked as the squared length of one vector in the xy plane, so
// no angle is taken, nothing is divided by sin theta, and the spin axis
// is a point like any other. With spin a geodesic does not keep it;
// Carter is the part it keeps.
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

// Velocity is the contravariant velocity along the geodesic, dx by the
// affine parameter: the inverse metric on the momentum.
func (hole Hole) Velocity(ray Ray) Vec4 {
	return hole.Raise(ray.Space(), ray.momentum)
}
