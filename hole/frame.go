package hole

import "math"

// Static is the four-velocity of an observer held still at a point: the
// time axis scaled to unit length.
//
// Nothing is still inside the ergosurface, where the hole's spin drags
// everything round with it: on the equator that is inside two M at any
// spin, on the axis it is the horizon, and at nought spin it is the
// horizon everywhere. There the answer is not a number.
func (hole Hole) Static(position Vec3) Vec4 {
	return hole.orbiting(position, nothing)
}

// Circular is the four-velocity of matter going round the hole on a
// circle at a point, prograde, as a thin disc's does: kepler's rate
// with the hole's correction, which is the same rate by the clock at
// infinity in these coordinates as in boyer and lindquist's.
//
// Inside the prograde photon orbit nothing goes round at that rate, and
// the answer is not a number. It can be had inside the ergosphere,
// where Static cannot.
func (hole Hole) Circular(position Vec3) Vec4 {
	radius := hole.Radius(position)
	sqrtMass := math.Sqrt(hole.mass)
	rate := sqrtMass / (radius*math.Sqrt(radius) + hole.spin*sqrtMass)
	return hole.orbiting(position, rate)
}

// orbiting is the four-velocity of an observer going round the spin
// axis at a point, at an angular velocity by the clock at infinity: the
// time axis and the turn about z in that proportion, scaled to unit
// length.
//
// Where nothing can go round at that rate, the answer is not a number.
func (hole Hole) orbiting(position Vec3, angularVelocity float64) Vec4 {
	velocity := Vec4{
		time: whole,
		x:    -angularVelocity * position.y,
		y:    angularVelocity * position.x,
	}
	norm := Dot(hole.Lower(position, velocity), velocity)
	// the root of a negative is not a number, which is the answer wanted
	scale := math.Sqrt(timelikeNorm / norm)
	return Vec4{time: scale, x: scale * velocity.x, y: scale * velocity.y}
}

// Frame is an observer's orthonormal tetrad at a point: the observer's
// four-velocity first, then three spatial axes square to it and to one
// another in the metric, made from the coordinate axes by gram and
// schmidt, x first.
type Frame struct{ observer, xAxis, yAxis, zAxis Vec4 }

// Observer is the frame's first vector, the observer's four-velocity.
func (frame Frame) Observer() Vec4 { return frame.observer }

// XAxis is the frame's second vector, the spatial axis made from x.
func (frame Frame) XAxis() Vec4 { return frame.xAxis }

// YAxis is the frame's third vector, the spatial axis made from y.
func (frame Frame) YAxis() Vec4 { return frame.yAxis }

// ZAxis is the frame's fourth vector, the spatial axis made from z.
func (frame Frame) ZAxis() Vec4 { return frame.zAxis }

// rows are the frame's four vectors in order, the observer's first.
func (frame Frame) rows() [dimensions]Vec4 {
	return [dimensions]Vec4{
		frame.observer, frame.xAxis, frame.yAxis, frame.zAxis,
	}
}

// along is the observer's time axis plus a direction along the spatial
// axes.
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

// components are the parts of a covariant vector on the frame: along the
// four-velocity first, then along the x, y and z axes.
func (frame Frame) components(covector Vec4) Vec4 {
	return Vec4{
		time: Dot(covector, frame.observer),
		x:    Dot(covector, frame.xAxis),
		y:    Dot(covector, frame.yAxis),
		z:    Dot(covector, frame.zAxis),
	}
}

// product is the metric at one point, as a kernel in hand, taking inner
// products.
type product struct {
	scalar float64
	null   Vec4
}

// of is the inner product of two vectors in the metric.
func (metric product) of(first, second Vec4) float64 {
	return Dot(lowered(metric.scalar, metric.null, first), second)
}

// axis is a seed made square to the observer and to the axes made before
// it, and then made unit length.
//
// The observer is timelike, so the part along it comes out with the other
// sign from the parts along the spatial axes.
func (metric product) axis(seed, observer Vec4, earlier ...Vec4) Vec4 {
	vector := seed.minus(observer, -metric.of(seed, observer))
	for _, axis := range earlier {
		vector = vector.minus(axis, metric.of(vector, axis))
	}
	norm := math.Sqrt(metric.of(vector, vector))
	return vector.scaled(whole / norm)
}

// FrameOf is the tetrad of an observer with four-velocity observer at a
// point.
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

// pointAt is a point in space at time nought, as a point in spacetime.
func pointAt(position Vec3) Vec4 {
	return Vec4{x: position.x, y: position.y, z: position.z}
}

// Null is a ray of light leaving a point in the direction direction as the
// observer with tetrad frame sees it: direction is a direction in the
// observer's own space, its length not used.
//
// The ray's momentum is the observer's time axis plus the unit direction
// along the spatial axes, lowered, so that it is null.
//
// The ray starts at time nought, and its energy as that observer measures
// it is one.
func (hole Hole) Null(position Vec3, frame Frame, direction Vec3) Ray {
	if length := direction.length(); length > nothing {
		direction = direction.scaled(whole / length)
	}
	momentum := hole.Lower(position, frame.along(direction))
	return Ray{position: pointAt(position), momentum: momentum}
}

// Seen is the direction an observer with tetrad frame sees a ray's light
// come from, a unit direction in the observer's own space, which is
// where the light appears on that observer's sky after aberration.
//
// It is the momentum's parts along the spatial axes, normalised, and
// turned to face the way the light came: about, for a photon's own
// momentum, and not at all for an arriving ray's, which Arriving has
// negated already.
//
// The momentum's part along the observer's time axis says which it is,
// below nought for a photon and above for an arriving ray, so Seen of the
// ray Arriving built from a direction is that direction.
func (hole Hole) Seen(frame Frame, ray Ray) Vec3 {
	components := frame.components(ray.momentum)
	direction := components.Space()
	magnitude := direction.length()
	if magnitude == nothing {
		return direction
	}
	// the length to divide by carries the facing as its sign
	signed := math.Copysign(magnitude, components.time)
	return direction.scaled(whole / signed)
}

// Falling is a body let go from rest at a radius on the x axis, as a
// camera falling into the hole: its geodesic from there, timelike with
// unit mass, marched by Advance.
//
// The radius is R, the kerr-schild one; round a spinning hole that point's
// x is the root of R squared plus A squared.
func (hole Hole) Falling(initialRadius float64) Ray {
	x := math.Sqrt(initialRadius*initialRadius + hole.spin*hole.spin)
	return hole.Dropped(Vec3{x: x})
}

// ringDistance is how far a point is from the singularity, the ring of
// radius A in the equator. At nought spin the ring is the centre and
// the distance is R.
func (hole Hole) ringDistance(position Vec3) float64 {
	fromRingInPlane := math.Hypot(position.x, position.y) - hole.spin
	return math.Hypot(fromRingInPlane, position.z)
}

// Advance carries a body's geodesic on by a proper time and says
// whether it is inside the horizon. Its steps are half the pace at the
// nearer of its radius and its distance from the ring: close to the
// ring the hole changes over that distance, which is much the shorter.
//
// Inside means inside the horizon itself, to the last digit: the margin
// a march of light stops at is for rays and is not used here.
//
// A body is carried no further than the chart goes.
//
// It stops, inside, with the rest of its proper time unspent, within
// choices.json's singularity margin of the ring or of the disc the ring
// spans, and at the inner horizon of a spinning hole.
//
// Past the inner horizon a body off the equator turns back outward, and
// its way out is not in these coordinates. The pace goes to nought at
// the singularity, and a step taken any nearer than the margin lands in
// the flat space the kernel falls back to at nought radius.
//
// Its steps are choices.json's advance share of the pace: a body's proper
// time is short beside a ray's parameter near the hole, so its steps are
// the finer.
//
// Advance reads the march's choices, so it is moving to slayer and will
// change; see the package doc.
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

// Arriving is the ray of light arriving at a point from the direction
// direction as the observer with tetrad frame sees it, set up to be
// marched forward in its parameter, which runs backward in time: a
// picture is traced from the eye.
//
// Its momentum is the arriving photon's negated, so the energies an
// observer measures along it, Dot(P, u), come out positive. Round a
// still hole it runs the path of the ray of Null leaving toward direction;
// round a spinning one the two part, since the dragging turns them.
func (hole Hole) Arriving(position Vec3, frame Frame, direction Vec3) Ray {
	ray := hole.Null(position, frame, direction.scaled(-whole))
	negated := ray.momentum.scaled(-whole)
	return Ray{position: ray.position, momentum: negated}
}

// Shift is the ratio of a photon's energy at the eye to its energy at
// the emitter, along a ray: its momentum at the eye on the observer's
// four-velocity there, over its momentum at the emitter on the emitter's
// four-velocity there.
//
// Over one the light arrives bluer, under one redder; doppler beaming and
// the hole's own redshift are both in it.
//
// The momentum is not the same at the two ends: a ray keeps only its
// time part. One momentum serves at both ends only where that end's
// observer is still, with nothing but a time part to its velocity.
func Shift(momentumAtEye, observer, momentumAtEmitter, emitter Vec4) float64 {
	return Dot(momentumAtEye, observer) / Dot(momentumAtEmitter, emitter)
}

// Dropped is a body let go from rest at a point, at time nought, as a
// camera is that falls in from anywhere; Falling is the same on the x
// axis.
func (hole Hole) Dropped(position Vec3) Ray {
	return Ray{
		position: pointAt(position),
		momentum: hole.Lower(position, hole.Static(position)),
	}
}

// RayClassification is where a ray or a body ends up, as ClassifyRay
// says it.
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

// ClassifyRay says whether a ray or a body escapes, plunges into the hole
// or circulates, by marching it and seeing where it goes.
//
// It escapes on getting choices.json's far factor times its starting
// radius out, or its far floor in M if that is further; it circulates if
// it has done neither after its classify steps.
//
// A bound body whose orbit reaches past that radius is classed as
// escaping, and a ray that is not a number as circulating.
//
// ClassifyRay marches, so it is moving to slayer and will change; see
// the package doc.
func (hole Hole) ClassifyRay(ray Ray) RayClassification {
	chosen := choices()
	startRadius := hole.Radius(ray.Space())
	farRadius := math.Max(
		chosen.farFactor*startRadius, chosen.farFloor*hole.mass,
	)
	_, fate := hole.March(ray, farRadius, chosen.steps)
	return classOf[fate]
}
