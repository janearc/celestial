//ninja:vow dojo going inward is held to the radius's gradient worked from Radius, and the march's give-up rules to their edges, through goingInward, limitsOf, pacedStep and runsAway, which hole does not export
package hole

import (
	"math"
	"testing"
)

// The numbers of a central difference and of a unit axis.
const (
	// bothWays is the 2 of a central difference, (f(x+h) - f(x-h)) / 2h.
	// Defined by the arithmetic.
	bothWays = 2
	// outward and inward are the signs of a velocity's part along the
	// radius's gradient. Defined by the arithmetic.
	outward, inward = 1, -1
	// belowAll is the sign math.Inf takes for minus infinity. Defined by
	// package math.
	belowAll = -1
)

// gradientOf is the radius's gradient at a point, worked by central
// differences of Radius, a step either way along each axis.
func gradientOf(black Hole, point Vec3, step float64) Vec3 {
	return Vec3{x: centralDifference(black, point, Vec3{x: step}, step),
		y: centralDifference(black, point, Vec3{y: step}, step),
		z: centralDifference(black, point, Vec3{z: step}, step)}
}

// centralDifference is the radius's rate along one axis at a point: the
// radius a shift ahead less the radius a shift behind, over twice the step.
func centralDifference(black Hole, point, shift Vec3, step float64) float64 {
	ahead := Vec3{x: point.x + shift.x, y: point.y + shift.y,
		z: point.z + shift.z}
	behind := Vec3{x: point.x - shift.x, y: point.y - shift.y,
		z: point.z - shift.z}
	return (black.Radius(ahead) - black.Radius(behind)) / (bothWays * step)
}

// offTheSphere is velocities at a point that leave the sphere of its
// radius by the share off of the gradient, the way the sign says: along
// the sphere in three ways, the gradient crossed with each axis, each
// worked out here and not by cross, which is held by tests of its own.
func offTheSphere(black Hole, point Vec3, step, off, sign float64) []Vec4 {
	gradient := gradientOf(black, point, step)
	tangents := []Vec3{
		{y: gradient.z, z: -gradient.y},
		{x: -gradient.z, z: gradient.x},
		{x: gradient.y, y: -gradient.x},
	}
	var velocities []Vec4
	for _, tangent := range tangents {
		share := sign * off * tangent.length() / gradient.length()
		velocities = append(velocities, Vec4{
			x: tangent.x + share*gradient.x,
			y: tangent.y + share*gradient.y,
			z: tangent.z + share*gradient.z,
		})
	}
	return velocities
}

// A velocity that leaves the sphere of its radius outward, by a ten
// millionth of the gradient, is not going inward, however it runs along
// the sphere, off the equator and round a spinning hole.
func TestALittleOutwardIsNotGoingInward(t *testing.T) {
	given := innerNumbers(t).Inward
	black := madeHole(t, given.Hole.Mass, given.Hole.Spin)
	for _, at := range given.Points {
		point := at.space()
		radius := black.Radius(point)
		for _, velocity := range offTheSphere(black, point,
			given.GradientStep, given.Off, outward) {
			if black.goingInward(pointAt(point), velocity, radius) {
				t.Errorf("at %v, %v a little outward is going inward",
					point, velocity)
			}
		}
	}
}

// The anti-test: the same velocities a little inward are going inward.
func TestALittleInwardIsGoingInward(t *testing.T) {
	given := innerNumbers(t).Inward
	black := madeHole(t, given.Hole.Mass, given.Hole.Spin)
	for _, at := range given.Points {
		point := at.space()
		radius := black.Radius(point)
		for _, velocity := range offTheSphere(black, point,
			given.GradientStep, given.Off, inward) {
			if !black.goingInward(pointAt(point), velocity, radius) {
				t.Errorf("at %v, %v a little inward is not going inward",
					point, velocity)
			}
		}
	}
}

// A point on the equator going round the axis and along it keeps its
// radius, to the last bit of the sum: it is not going inward.
func TestAlongTheSphereIsNotGoingInward(t *testing.T) {
	given := innerNumbers(t).Inward
	black := madeHole(t, given.Hole.Mass, given.Hole.Spin)
	point := given.Round.space()
	radius := black.Radius(point)
	if black.goingInward(pointAt(point), given.Along.spacetime(), radius) {
		t.Errorf("at %v, %v along the sphere is going inward", point,
			given.Along)
	}
}

// The anti-test: a ten millionth of the point taken away, toward the
// axis, it is.
func TestJustInwardOfTheSphereIsGoingInward(t *testing.T) {
	given := innerNumbers(t).Inward
	black := madeHole(t, given.Hole.Mass, given.Hole.Spin)
	point := given.Round.space()
	radius := black.Radius(point)
	velocity := given.Along.spacetime().minus(pointAt(point), given.Off)
	if !black.goingInward(pointAt(point), velocity, radius) {
		t.Errorf("at %v, %v just inward is not going inward", point,
			velocity)
	}
}

// sentInward is a ray of light sent toward the hole by a still observer
// on the x axis, as a march's point and momentum, with its limits.
func sentInward(black Hole, x float64, toward Vec3) (Vec4, Vec4, limits) {
	position := Vec3{x: x}
	frame := black.FrameOf(position, black.Static(position))
	ray := black.Null(position, frame, toward)
	return ray.position, ray.momentum,
		black.limitsOf(ray.position, ray.momentum)
}

// noReturnOnTheAxis is the radius a ray sent inward along the x axis of a
// still hole cannot come back from, found from a ray at the photon sphere;
// round a still hole every ray's is the same, which the test checks.
func noReturnOnTheAxis(test *testing.T, black Hole, toward Vec3) float64 {
	test.Helper()
	_, _, found := sentInward(black, black.PhotonSphere(), toward)
	_, _, again := sentInward(black, found.noReturn, toward)
	if again.noReturn != found.noReturn {
		test.Fatalf("the radius of no return moved with the ray: %v, %v",
			found.noReturn, again.noReturn)
	}
	return found.noReturn
}

// A ray going inward at exactly the radius it cannot come back from is not
// yet inside it, and is stepped.
func TestAtTheRadiusOfNoReturnARayIsStepped(t *testing.T) {
	given := innerNumbers(t).Edge
	still := madeHole(t, given.Still.Mass, given.Still.Spin)
	toward := given.TowardTheHole.space()
	edge := noReturnOnTheAxis(t, still, toward)
	point, momentum, bounds := sentInward(still, edge, toward)
	radius := still.radius(point.x, point.y, point.z)
	if radius != edge {
		t.Fatalf("the ray is at %v, not %v", radius, edge)
	}
	if _, _, givenUp := still.pacedStep(point, momentum, radius,
		bounds); givenUp {
		t.Errorf("a ray going inward at %v is given up there", radius)
	}
}

// The anti-test: a hair inside it, the same ray is given up.
func TestAHairInsideTheRadiusOfNoReturnARayIsGivenUp(t *testing.T) {
	given := innerNumbers(t).Edge
	still := madeHole(t, given.Still.Mass, given.Still.Spin)
	toward := given.TowardTheHole.space()
	inside := math.Nextafter(noReturnOnTheAxis(t, still, toward), nothing)
	point, momentum, bounds := sentInward(still, inside, toward)
	radius := still.radius(point.x, point.y, point.z)
	if _, _, givenUp := still.pacedStep(point, momentum, radius,
		bounds); !givenUp {
		t.Errorf("a ray going inward at %v, inside %v, is stepped",
			radius, bounds.noReturn)
	}
}

// A ray whose time runs backward exactly the hover ratio times its energy
// is not running away: only faster than that.
func TestAtTheHoverRatioTimeDoesNotRunAway(t *testing.T) {
	energy := innerNumbers(t).Edge.Energy
	timeVelocity := -choices().hover * energy
	if runsAway(timeVelocity, -energy) {
		t.Errorf("time running back at %v for energy %v runs away",
			timeVelocity, energy)
	}
}

// The anti-test: a hair faster backward, it runs away.
func TestAHairPastTheHoverRatioTimeRunsAway(t *testing.T) {
	energy := innerNumbers(t).Edge.Energy
	faster := math.Nextafter(-choices().hover*energy, math.Inf(belowAll))
	if !runsAway(faster, -energy) {
		t.Errorf("time running back at %v for energy %v does not run away",
			faster, energy)
	}
}
