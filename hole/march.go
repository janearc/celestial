package hole

import "math"

// pace chooses an affine step as a configured fraction of radius, smaller near
// the hole and larger in the nearly flat far field. For unit-energy light it is
// approximately a distance step; other photon energies need PaceIn to follow
// the same spatial path.
func (hole Hole) pace(radius float64) float64 {
	return choices().pace * radius
}

// PaceIn sizes a ray’s step so coordinate time changes by the chosen fraction
// of its current radius. Far away, this also gives an energy-independent
// spatial step.
//
// Near the horizon, an incoming ray traced toward its distant-past origin can
// have unbounded time rate and momentum. The step shrinks to avoid overshooting
// that limit. Ingoing physical light crossing the horizon keeps a finite step
// in this chart.
//
// PaceIn uses marching choices and belongs to the proposed move of rendering
// policy to Slayer.
func (hole Hole) PaceIn(ray Ray) float64 {
	return choices().time * hole.Radius(ray.Space()) /
		math.Abs(hole.timeVelocity(ray))
}

// Hovering identifies incoming rays whose backward time rate exceeds the
// configured ratio to conserved energy. Integration would approach the horizon
// origin without reaching it; the marcher treats it as black. Outgoing light
// with a large forward rate is not hovering.
//
// This stopping policy is part of the proposed move to Slayer.
func (hole Hole) Hovering(ray Ray) bool {
	return runsAway(hole.timeVelocity(ray), ray.momentum.time)
}

// timeVelocity evaluates dt/dlambda without the spatial components. Its sign
// distinguishes physical forward-time rays from incoming backward-traced rays.
func (hole Hole) timeVelocity(ray Ray) float64 {
	scalar, vector := hole.kernel(ray.Space())
	momentum := ray.momentum
	return -momentum.time + scalar*(-momentum.time+
		vector.x*momentum.x+vector.y*momentum.y+vector.z*momentum.z)
}

// runsAway checks the configured backward-time-rate to energy threshold.
func runsAway(timeVelocity, timeMomentum float64) bool {
	return -timeVelocity > choices().hover*math.Abs(timeMomentum)
}

// March integrates until a ray escapes farRadius, is captured, or exhausts its
// step budget. The returned ray is the state at stopping, for either forward
// light or backward-traced incoming light.
//
// An inward physical ray is captured once it lies inside the photon orbit
// associated with its Lz/E and cannot return. At zero spin this can stop at 3M,
// well outside the horizon; the returned endpoint is not a horizon
// intersection.
//
// Forward light inside the horizon is captured immediately. Outward light just
// outside can escape. An incoming ray inside the horizon is instead allowed to
// trace back to light that fell in after the observer; hovering supplies its
// separate stopping condition.
//
// Steps use PaceIn and fourth-order Runge–Kutta. A step crossing the far radius
// is retained, so the endpoint may overshoot. Nonfinite state is stuck
// immediately.
//
// March is included in the proposed move of rendering policy to Slayer.
func (hole Hole) March(ray Ray, farRadius float64, steps int) (Ray, Fate) {
	point, momentum := ray.position, ray.momentum
	bounds := hole.limitsOf(point, momentum)
	fate := Stuck
	for range steps {
		radius := hole.radius(point.x, point.y, point.z)
		if math.IsNaN(radius) {
			break
		}
		if radius > farRadius {
			fate = Escaped
			break
		}
		var givenUp bool
		if point, momentum, givenUp = hole.pacedStep(
			point, momentum, radius, bounds,
		); givenUp {
			fate = Captured
			break
		}
	}
	return Ray{position: point, momentum: momentum}, fate
}

// pacedStep reuses the first Hamiltonian rate to size a step, test hovering and
// inward motion, and perform Runge–Kutta integration. It returns capture=true
// without stepping when the stopping rule applies.
func (hole Hole) pacedStep(
	point, momentum Vec4, radius float64, bounds limits,
) (Vec4, Vec4, bool) {
	velocity, momentumChange := hole.rateAt(point, momentum, radius)
	notComingBack := radius < bounds.noReturn &&
		hole.goingInward(point, velocity, radius)
	hovering := radius < bounds.horizon &&
		runsAway(velocity.time, momentum.time)
	if notComingBack || hovering {
		return point, momentum, true
	}
	point, momentum = hole.stepFrom(
		point, momentum, velocity, momentumChange,
		choices().time*radius/math.Abs(velocity.time),
	)
	return point, momentum, false
}

// ImpactParameter returns total angular momentum divided by energy: the distant
// straight-line miss distance without deflection. It includes components along
// and across the spin axis.
//
// At zero spin it is conserved and capture occurs below Shadow. With spin it
// describes the distant sky position, not a universal capture threshold; Lz/E
// and Carter/E² determine capture separately. Zero energy returns infinity.
func (hole Hole) ImpactParameter(ray Ray) float64 {
	position := ray.Space()
	angularMomentum := hole.Lz(ray)
	squared := hole.acrossAxisSquared(
		position, ray.momentum, hole.Radius(position),
	) + angularMomentum*angularMomentum
	return math.Sqrt(squared) / math.Abs(hole.Energy(ray))
}

// ClosestApproach returns the smallest radius sampled while stepping until an
// outward turn, capture, or exhausted budget, including the last state. An
// initially outgoing ray returns its starting radius.
//
// For captured light this only bounds the radius where integration stopped
// inside the photon orbit. Sampling can overestimate the minimum by roughly
// (half-step)²/(2r), about half a part in ten thousand at the chosen pace.
//
// Nonfinite state returns NaN, including a ray placed at infinity: such a
// ray cannot be marched, and infinity would imply it came no closer.
//
// This method is moving to Slayer with the march; see the package doc.
func (hole Hole) ClosestApproach(ray Ray, steps int) float64 {
	point, momentum := ray.position, ray.momentum
	bounds := hole.limitsOf(point, momentum)
	radius := hole.radius(point.x, point.y, point.z)
	leastRadius := radius
	turnedBack := whole + choices().turnedBack
	// go on only while the ray has not turned back; a radius that is not
	// a number fails the comparison, and that stops the march too
	for range steps {
		if !(radius <= leastRadius*turnedBack) {
			break
		}
		var givenUp bool
		if point, momentum, givenUp = hole.pacedStep(
			point, momentum, radius, bounds,
		); givenUp {
			break
		}
		radius = hole.radius(point.x, point.y, point.z)
		leastRadius = math.Min(leastRadius, radius)
	}
	return leastRadius
}

// DeflectionAngle returns the angle from the initial spatial velocity to the
// escaped velocity, in radians from zero to pi. Both endpoints must be far
// enough away for spatial coordinates to approximate flat-space directions.
//
// This is the endpoint angle, not accumulated winding: a ray circling more than
// halfway is still reported in [0, pi]. A ray that does not escape returns NaN.
// This marching helper is part of the proposed move to Slayer.
func (hole Hole) DeflectionAngle(
	ray Ray, farRadius float64, steps int,
) float64 {
	end, fate := hole.March(ray, farRadius, steps)
	if fate != Escaped {
		return math.NaN()
	}
	setOut := hole.Velocity(ray).Space()
	leaves := hole.Velocity(end).Space()
	return angleBetween(setOut, leaves)
}

// angleBetween returns atan2(|first × second|, first · second) in [0, pi],
// preserving small-angle precision lost by acos alone.
func angleBetween(first, second Vec3) float64 {
	return math.Atan2(first.cross(second).length(), first.dot(second))
}
