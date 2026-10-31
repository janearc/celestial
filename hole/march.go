package hole

import "math"

// pace is a step's size at a radius: a fiftieth of the radius, so the
// steps are fine near the hole, where the bending is, and coarse far
// from it, where space is flat and a long step is exact.
//
// The step is in the affine parameter, which is about distance for a
// ray from far away whose energy is one, and proper time for a body. It
// is not distance for light of any other energy: that light goes as
// many times too far in a step as its energy is from one.
//
// PaceIn is the step for a ray, and is what March takes.
func (hole Hole) pace(radius float64) float64 {
	return choices().pace * radius
}

// PaceIn is the step for a ray where it stands: the step in which its
// time moves on, or back, by choices.json's time share of its radius.
//
// Far from the hole that is the same share of the radius in distance
// too, whatever the light's energy, since a photon's energy has no say in
// where it goes.
//
// Near the hole's edge it is less, and for an arriving ray that came
// from the edge it shrinks without end: that light left the edge in the
// far past, its time runs away as the ray comes to it, and so does its
// momentum, which a step sized by distance would overshoot.
//
// A ray crossing the horizon the other way, light that fell in, keeps its
// time in hand and crosses at a full step, as the chart lets.
func (hole Hole) PaceIn(ray Ray) float64 {
	return choices().time * hole.Radius(ray.Space()) /
		math.Abs(hole.timeVelocity(ray))
}

// Hovering is whether an arriving ray has come from the hole's edge:
// its time runs away backward, choices.json's hover ratio times its
// energy's, and the march would never get there. Such a ray is black.
//
// Light leaving the edge outward has its time run as fast forward, and
// is not hovering: it gets out.
func (hole Hole) Hovering(ray Ray) bool {
	return runsAway(hole.timeVelocity(ray), ray.momentum.time)
}

// timeVelocity is the time part of Velocity alone, dt by the affine
// parameter: above nought for a ray marched forward in time and below
// for an arriving one, and never nought, anywhere, since the chart's
// time is a time inside the horizon too.
func (hole Hole) timeVelocity(ray Ray) float64 {
	scalar, vector := hole.kernel(ray.Space())
	momentum := ray.momentum
	return -momentum.time + scalar*(-momentum.time+
		vector.x*momentum.x+vector.y*momentum.y+vector.z*momentum.z)
}

// runsAway is whether a ray's time runs backward choices.json's hover
// ratio times faster than its energy.
func runsAway(timeVelocity, timeMomentum float64) bool {
	return -timeVelocity > choices().hover*math.Abs(timeMomentum)
}

// March carries a ray until it is past far from the hole, captured, or
// out of steps, and says which; the ray handed back is where it
// stopped. It marches a leaving ray forward and an arriving ray back
// alike.
//
// A ray is captured as soon as it cannot come back: when it is inside
// the photon orbit that has its own angular momentum and is still going
// inward.
//
// For light seen from outside that is well short of the horizon, three M
// round a still hole, and the ray is handed back from there; that pixel is
// black.
//
// Light sent forward from inside the horizon is always going inward, and
// is captured where it starts. Light sent outward from just outside the
// horizon is not captured, however near: it gets out.
//
// An arriving ray is not captured for being inside the horizon: an eye
// that has fallen through still sees the light that fell in after it,
// and March carries that light back out to where it came from. Inside
// the horizon an arriving ray is captured when it is found hovering.
//
// Each step is a runge-kutta step of PaceIn's size, and a step that
// would cross the far radius is taken anyway, so the ray may end a
// little past it. A ray that is not a number is stuck at once.
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

// pacedStep is one step of a march, from a point whose radius is in
// hand: a runge-kutta step of PaceIn's size, or no step at all and true
// if the march gives the ray up there as captured.
//
// The time velocity that sizes the step and says whether the ray hovers
// is the time part of the step's first rate, and the velocity that says
// whether it is going inward is the rest of it, so the rate is worked
// out once here and the step is given it.
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

// ImpactParameter is how far from the hole's centre a ray of light from
// far away would pass were it not bent: its angular momentum over its
// energy, the whole angular momentum, across the spin axis and about
// it, so the answer is the same in whatever plane the ray lies.
//
// Round a still hole the ray keeps it, and is captured when it is less
// than Shadow.
//
// Round a spinning one it is the ray's distance from the centre on the sky
// of whoever is far along it, right in the limit of far away; there is no
// one radius to hold it against there, since capture is decided by Lz over
// E and Carter over E squared apart.
//
// A ray with no energy has none, and the answer is infinite.
func (hole Hole) ImpactParameter(ray Ray) float64 {
	position := ray.Space()
	angularMomentum := hole.Lz(ray)
	squared := hole.acrossAxisSquared(
		position, ray.momentum, hole.Radius(position),
	) + angularMomentum*angularMomentum
	return math.Sqrt(squared) / math.Abs(hole.Energy(ray))
}

// ClosestApproach is the least radius a ray comes to: it is marched, as
// March marches it, until it has turned back outward, been captured, or
// taken all its steps, and the least radius it stood at, the last
// included, is the answer.
//
// A ray on its way out from the start answers the radius it started at,
// and a captured ray the radius it was given up at, which is inside its
// photon orbit and says no more than that.
//
// The answer is the least of the radii stepped at, so it is over by up
// to the square of half a step over twice the radius: half a part in
// ten thousand. For a ray that is not a number it is not a number, at
// once.
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

// DeflectionAngle is the angle between the way a ray sets out and the
// way it leaves past the far radius, in radians from nought to pi: how
// far the hole bent it, in whatever plane it was bent.
//
// The two ways are the ray's velocity in space, which is where it is
// going only where space is flat, so the start and the far radius MUST
// both be far from the hole.
//
// A ray that goes more than half round reads as the angle between its two
// ways, not as its turning. A ray that does not escape has no deflection,
// and the answer is not a number.
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

// angleBetween is the angle between two directions in space, from
// nought to pi, by the arctangent of the cross product's length over
// the dot product: it keeps its digits at small angles, where the
// arccosine of the dot product alone loses them.
func angleBetween(first, second Vec3) float64 {
	return math.Atan2(first.cross(second).length(), first.dot(second))
}
