package hole

// kerrTerms caches quantities shared by the velocity and momentum derivatives,
// avoiding repeated divisions in Hamilton’s equations.
type kerrTerms struct {
	radius            float64
	radiusSpinSum     float64
	overRadius        float64
	overRadiusSpinSum float64
	overDenominator   float64
	reach             float64 // the null vector on the momentum
	reachInPlane      float64 // the part of reach from x and y
	reachAlongAxis    float64 // the part of reach from z
	pull              float64 // the scalar times reach
}

// rateAt is Hamilton's equations for a point and a momentum whose radius is
// already in hand.
//
// The inverse metric is flat minus scalar * vector * vector, so with reach the
// vector on the momentum, the hamiltonian is the flat one less half of scalar *
// reach squared, and
//
//	velocity        = flat momentum - scalar * reach * vector
//	momentum change = reach * (reach/2 * grad scalar + scalar * grad reach)
//
// Both gradients are a multiple of the radius's gradient plus a short vector of
// their own, so the whole of it is three divisions and no loop, written out in
// a line as a shader would have it. The kernel is worked here again and not
// called for, to share those divisions.
func (hole Hole) rateAt(
	point, momentum Vec4, radius float64,
) (velocity, momentumChange Vec4) {
	spin := hole.spin
	radiusSquared, spinSquared := radius*radius, spin*spin
	radiusSpinSum := radiusSquared + spinSquared
	// r^4 + A^2 z^2, which is zero only on the ring and the disc
	denominator := radiusSquared*radiusSquared + spinSquared*point.z*point.z
	if denominator == nothing || radius == nothing {
		// flat space, as kernel answers there
		return Vec4{time: -momentum.time, x: momentum.x, y: momentum.y,
			z: momentum.z}, Vec4{}
	}
	terms := kerrTerms{
		radius:            radius,
		radiusSpinSum:     radiusSpinSum,
		overRadius:        whole / radius,
		overRadiusSpinSum: whole / radiusSpinSum,
		overDenominator:   whole / denominator,
	}
	scalar := scalarFactor * hole.mass * radius * radiusSquared *
		terms.overDenominator
	vector := Vec4{
		x: (radius*point.x + spin*point.y) * terms.overRadiusSpinSum,
		y: (radius*point.y - spin*point.x) * terms.overRadiusSpinSum,
		z: point.z * terms.overRadius,
	}
	terms.reachInPlane = vector.x*momentum.x + vector.y*momentum.y
	terms.reachAlongAxis = vector.z * momentum.z
	terms.reach = terms.reachInPlane + terms.reachAlongAxis - momentum.time
	terms.pull = scalar * terms.reach
	velocity = Vec4{
		time: terms.pull - momentum.time,
		x:    momentum.x - terms.pull*vector.x,
		y:    momentum.y - terms.pull*vector.y,
		z:    momentum.z - terms.pull*vector.z,
	}
	return velocity, hole.momentumChange(point, momentum, terms)
}

// momentumChange is the momentum's change along the geodesic, minus half
// the gradient of the inverse metric on the momentum twice, from the
// terms rateAt has worked out.
//
// The radius's gradient, from the quartic that defines it, is
// (x r^3, y r^3, z r (r^2 + A^2)) / (r^4 + A^2 z^2).
func (hole Hole) momentumChange(
	point, momentum Vec4, terms kerrTerms,
) Vec4 {
	radius, spin := terms.radius, hole.spin
	radiusSquared, spinSquared := radius*radius, spin*spin
	radiusGradInPlane := radius * radiusSquared * terms.overDenominator
	radiusGradZ := point.z * radius * terms.radiusSpinSum *
		terms.overDenominator

	// grad scalar / scalar = scalarAlongRadius * grad r -
	// scalarAlongAxis * zhat
	scalarAlongRadius := (scalarGradientThree*spinSquared*point.z*point.z -
		radiusSquared*radiusSquared) * terms.overRadius *
		terms.overDenominator
	scalarAlongAxis := gradientFactor * spinSquared * point.z *
		terms.overDenominator
	// grad reach = reachAlongRadius * grad r + the vector's own turn with
	// the point
	overSum := terms.overRadiusSpinSum
	reachAlongRadius := (point.x*momentum.x+point.y*momentum.y-
		gradientFactor*radius*terms.reachInPlane)*overSum -
		terms.reachAlongAxis*terms.overRadius
	alongRadius := gradientHalf*terms.reach*scalarAlongRadius +
		reachAlongRadius

	pull := terms.pull
	alongAxis := alongRadius*radiusGradZ + momentum.z*terms.overRadius -
		gradientHalf*terms.reach*scalarAlongAxis
	return Vec4{
		x: pull * (alongRadius*point.x*radiusGradInPlane +
			(radius*momentum.x-spin*momentum.y)*overSum),
		y: pull * (alongRadius*point.y*radiusGradInPlane +
			(radius*momentum.y+spin*momentum.x)*overSum),
		z: pull * alongAxis,
	}
}

// rateOf is rateAt for a point whose radius is not in hand.
func (hole Hole) rateOf(
	point, momentum Vec4,
) (velocity, momentumChange Vec4) {
	return hole.rateAt(
		point, momentum, hole.radius(point.x, point.y, point.z),
	)
}

// Step advances by one fourth-order Runge–Kutta interval in the affine
// parameter, using the caller’s step size.
func (hole Hole) Step(ray Ray, stepSize float64) Ray {
	velocity, momentumChange := hole.rateOf(ray.position, ray.momentum)
	point, momentum := hole.stepFrom(
		ray.position, ray.momentum, velocity, momentumChange, stepSize,
	)
	return Ray{position: point, momentum: momentum}
}

// stepFrom takes a Runge–Kutta step with the initial derivative already
// available from the marcher’s step-size calculation.
func (hole Hole) stepFrom(
	point, momentum, velocity1, momentumChange1 Vec4, stepSize float64,
) (Vec4, Vec4) {
	halfStep := stepSize * midpoint
	velocity2, momentumChange2 := hole.rateOf(
		point.moved(velocity1, halfStep),
		momentum.moved(momentumChange1, halfStep),
	)
	velocity3, momentumChange3 := hole.rateOf(
		point.moved(velocity2, halfStep),
		momentum.moved(momentumChange2, halfStep),
	)
	velocity4, momentumChange4 := hole.rateOf(
		point.moved(velocity3, stepSize),
		momentum.moved(momentumChange3, stepSize),
	)
	pointStep := rungeKutta(velocity1, velocity2, velocity3, velocity4)
	momentumStep := rungeKutta(
		momentumChange1, momentumChange2,
		momentumChange3, momentumChange4,
	)
	weighed := stepSize / rungeKuttaWeights
	return point.moved(pointStep, weighed),
		momentum.moved(momentumStep, weighed)
}
