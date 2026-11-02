package hole

import "math"

// Static returns the four-velocity of an observer held at fixed coordinates,
// normalizing the time axis. It returns NaN components where such an observer
// cannot exist.
//
// For a spinning hole this boundary is the ergosurface: 2M in the equatorial
// plane and the horizon on the axis. At zero spin it is the horizon everywhere.
func (hole Hole) Static(position Vec3) Vec4 {
	return hole.orbiting(position, nothing)
}

// Circular returns prograde circular four-velocity at the position, using the
// Kerr correction to Kepler’s angular rate as measured at infinity. That rate
// agrees in these and Boyer–Lindquist coordinates.
//
// Inside the prograde photon orbit this motion is not timelike and returns NaN
// components. Where admissible, it can exist inside the ergosphere even though
// Static cannot.
func (hole Hole) Circular(position Vec3) Vec4 {
	radius := hole.Radius(position)
	sqrtMass := math.Sqrt(hole.mass)
	rate := sqrtMass / (radius*math.Sqrt(radius) + hole.spin*sqrtMass)
	return hole.orbiting(position, rate)
}

// orbiting normalizes a combination of the time axis and rotation about z at
// the supplied angular velocity. The result is NaN where that motion cannot
// describe a timelike observer.
func (hole Hole) orbiting(position Vec3, angularVelocity float64) Vec4 {
	velocity := Vec4{
		time: whole,
		x:    -angularVelocity * position.y,
		y:    angularVelocity * position.x,
	}
	norm := Dot(hole.Lower(position, velocity), velocity)
	// The root of a negative is not a number, which is the answer wanted
	scale := math.Sqrt(timelikeNorm / norm)
	return Vec4{time: scale, x: scale * velocity.x, y: scale * velocity.y}
}

// Frame is an observer’s orthonormal tetrad: four-velocity first, then spatial
// axes. Metric Gram–Schmidt starts from the coordinate x, y, and z axes.
type Frame struct{ observer, xAxis, yAxis, zAxis Vec4 }

// Observer is the frame's first vector, the observer's four-velocity.
func (frame Frame) Observer() Vec4 { return frame.observer }

// XAxis is the frame's second vector, the spatial axis made from x.
func (frame Frame) XAxis() Vec4 { return frame.xAxis }

// YAxis is the frame's third vector, the spatial axis made from y.
func (frame Frame) YAxis() Vec4 { return frame.yAxis }

// ZAxis is the frame's fourth vector, the spatial axis made from z.
func (frame Frame) ZAxis() Vec4 { return frame.zAxis }

// rows returns the observer and spatial axes in that order.
func (frame Frame) rows() [dimensions]Vec4 {
	return [dimensions]Vec4{
		frame.observer, frame.xAxis, frame.yAxis, frame.zAxis,
	}
}

// along combines the observer’s time axis with a local spatial direction.
func (frame Frame) along(direction Vec3) Vec4 {
	return Vec4{
		time: frame.observer.time + direction.x*frame.xAxis.time +
			direction.y*frame.yAxis.time +
			direction.z*frame.zAxis.time,
		x: frame.observer.x + direction.x*frame.xAxis.x +
			direction.y*frame.yAxis.x + direction.z*frame.zAxis.x,
		y: frame.observer.y + direction.x*frame.xAxis.y +
			direction.y*frame.yAxis.y + direction.z*frame.zAxis.y,
		z: frame.observer.z + direction.x*frame.xAxis.z +
			direction.y*frame.yAxis.z + direction.z*frame.zAxis.z,
	}
}

// components contracts a covector with the observer and three spatial axes, in
// tetrad order.
func (frame Frame) components(covector Vec4) Vec4 {
	return Vec4{
		time: Dot(covector, frame.observer),
		x:    Dot(covector, frame.xAxis),
		y:    Dot(covector, frame.yAxis),
		z:    Dot(covector, frame.zAxis),
	}
}

// product caches the local metric kernel for repeated inner products.
type product struct {
	scalar float64
	null   Vec4
}

// of is the inner product of two vectors in the metric.
func (metric product) of(first, second Vec4) float64 {
	return Dot(lowered(metric.scalar, metric.null, first), second)
}

// axis orthogonalizes a seed against the observer and previously constructed
// axes, then normalizes it. Timelike projection has the opposite sign from
// projection along spatial axes.
func (metric product) axis(seed, observer Vec4, earlier ...Vec4) Vec4 {
	vector := seed.minus(observer, -metric.of(seed, observer))
	for _, axis := range earlier {
		vector = vector.minus(axis, metric.of(vector, axis))
	}
	norm := math.Sqrt(metric.of(vector, vector))
	return vector.scaled(whole / norm)
}

// FrameOf constructs a tetrad at position for the supplied observer
// four-velocity.
func (hole Hole) FrameOf(position Vec3, observer Vec4) Frame {
	scalar, null := hole.kernel(position)
	metric := product{scalar: scalar, null: null}
	xAxis := metric.axis(Vec4{x: whole}, observer)
	yAxis := metric.axis(Vec4{y: whole}, observer, xAxis)
	zAxis := metric.axis(Vec4{z: whole}, observer, xAxis, yAxis)
	return Frame{
		observer: observer, xAxis: xAxis, yAxis: yAxis, zAxis: zAxis,
	}
}

// pointAt places a spatial point at coordinate time zero.
func pointAt(position Vec3) Vec4 {
	return Vec4{x: position.x, y: position.y, z: position.z}
}

// Null constructs outgoing light in a local frame direction, ignoring the
// direction’s length. The observer’s time axis plus the normalized spatial
// direction is lowered to covariant momentum.
//
// The ray begins at coordinate time zero and has unit energy as measured by
// that observer.
func (hole Hole) Null(position Vec3, frame Frame, direction Vec3) Ray {
	if length := direction.length(); length > nothing {
		direction = direction.scaled(whole / length)
	}
	momentum := hole.Lower(position, frame.along(direction))
	return Ray{position: pointAt(position), momentum: momentum}
}

// Seen returns the unit direction from which a frame’s observer sees the ray’s
// light, including aberration. Spatial tetrad components are normalized with
// the sign appropriate to the momentum convention.
//
// Physical photon momentum is reversed to point toward its origin. Arriving
// already negates that momentum, so it needs no reversal. The time contraction
// distinguishes them; Seen therefore reverses Arriving’s construction of a sky
// direction.
func (hole Hole) Seen(frame Frame, ray Ray) Vec3 {
	components := frame.components(ray.momentum)
	direction := components.Space()
	magnitude := direction.length()
	if magnitude == nothing {
		return direction
	}
	// The length to divide by carries the facing as its sign
	signed := math.Copysign(magnitude, components.time)
	return direction.scaled(whole / signed)
}

// Falling drops a unit-mass body from rest on the x axis for integration by
// Advance. initialRadius is the Kerr–Schild radius; with spin, the
// corresponding x coordinate is sqrt(radius² + A²).
func (hole Hole) Falling(initialRadius float64) Ray {
	x := math.Sqrt(initialRadius*initialRadius + hole.spin*hole.spin)
	return hole.Dropped(Vec3{x: x})
}

// ringDistance returns distance to the equatorial ring singularity of radius A.
// At zero spin the ring collapses to the centre.
func (hole Hole) ringDistance(position Vec3) float64 {
	fromRingInPlane := math.Hypot(position.x, position.y) - hole.spin
	return math.Hypot(fromRingInPlane, position.z)
}

// Advance integrates a body through proper time and reports whether it is
// inside the horizon. Steps use the configured fraction of pace at the smaller
// of radius and distance to the ring.
//
// The inside flag uses the horizon itself, not the ray capture margin.
// Integration stops before leaving the supported chart: near the ring or its
// spanning disc within the configured singularity margin, and at a spinning
// hole’s inner horizon. Remaining proper time is unspent.
//
// Beyond the inner horizon, off-equator trajectories can turn outward through a
// region this chart does not cover. Near the singularity, pace tends to zero
// and stepping into the zero-radius flat fallback would be meaningless.
//
// Advance uses choices.json and is included in the proposed migration of
// marching policy to Slayer; see the package documentation.
func (hole Hole) Advance(ray Ray, properTime float64) (Ray, bool) {
	innerHorizon := hole.innerHorizon()
	remainingProperTime := properTime
	chosen := choices()
	for remainingProperTime > nothing {
		position := ray.Space()
		radius := hole.Radius(position)
		reach := math.Min(radius, hole.ringDistance(position))
		margin := chosen.margin * hole.mass
		if reach < margin || radius < innerHorizon {
			return ray, true
		}
		stepSize := math.Min(
			remainingProperTime, hole.pace(reach)*chosen.advance,
		)
		ray = hole.Step(ray, stepSize)
		remainingProperTime -= stepSize
	}
	return ray, hole.Radius(ray.Space()) < hole.Horizon()
}

// Arriving constructs light seen coming from a local frame direction for
// backward camera tracing. Its negated photon momentum advances the integration
// parameter while coordinate time runs backward. Observer energy contractions
// Dot(P,u) are then positive.
//
// At zero spin it follows the spatial path of Null sent toward the same
// direction. Frame dragging separates those paths with spin.
func (hole Hole) Arriving(position Vec3, frame Frame, direction Vec3) Ray {
	ray := hole.Null(position, frame, direction.scaled(-whole))
	negated := ray.momentum.scaled(-whole)
	return Ray{position: ray.position, momentum: negated}
}

// Shift returns photon energy at the eye divided by energy at the emitter,
// contracting each endpoint momentum with that observer’s four-velocity. Above
// one is bluer; below one is redder. Relative motion and gravitational redshift
// both contribute.
//
// Only the time component of momentum is generally conserved. Reusing one
// endpoint’s momentum at the other is valid only when that observer’s velocity
// has no spatial components.
func Shift(momentumAtEye, observer, momentumAtEmitter, emitter Vec4) float64 {
	return Dot(momentumAtEye, observer) / Dot(momentumAtEmitter, emitter)
}

// Dropped releases a unit-mass body from rest at an arbitrary point and
// coordinate time zero. Falling is the x-axis version.
func (hole Hole) Dropped(position Vec3) Ray {
	return Ray{
		position: pointAt(position),
		momentum: hole.Lower(position, hole.Static(position)),
	}
}

// RayClassification describes the outcome assigned by ClassifyRay.
type RayClassification int

// The classifications.
const (
	RayEscapes RayClassification = iota // got far away
	RayPlunges                          // was captured
	// did neither in the steps it was given
	RayCirculates
)

// classOf is the classification each fate of a march stands for.
var classOf = [...]RayClassification{
	Escaped:  RayEscapes,
	Captured: RayPlunges,
	Stuck:    RayCirculates,
}

// ClassifyRay marches a trajectory to classify it as escaping, plunging, or
// circulating. Escape uses the larger of the configured multiple of starting
// radius and the fixed far-radius floor. Exhausting the classification step
// budget assigns circulating.
//
// This is a finite integration classification: a bound orbit extending beyond
// the threshold can read as escaping, and a nonfinite ray as circulating. It
// uses marching choices and belongs to the proposed move of rendering policy to
// Slayer.
func (hole Hole) ClassifyRay(ray Ray) RayClassification {
	chosen := choices()
	startRadius := hole.Radius(ray.Space())
	farRadius := math.Max(
		chosen.farFactor*startRadius, chosen.farFloor*hole.mass,
	)
	_, fate := hole.March(ray, farRadius, chosen.steps)
	return classOf[fate]
}
