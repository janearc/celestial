package hole

import "math"

// Vec3 is a point or a direction in space.
//
// Its parts are private so that its places are said by name, AlongX,
// AlongY and AlongZ, and not by number.
//
// Any three numbers make one, a number that is not a number among them:
// that is how a place with no value is carried.
type Vec3 struct{ x, y, z float64 }

// NewVec3 is the point or direction with the parts x, y and z.
func NewVec3(x, y, z float64) Vec3 { return Vec3{x: x, y: y, z: z} }

// AlongX is the part along the x axis.
func (vec Vec3) AlongX() float64 { return vec.x }

// AlongY is the part along the y axis.
func (vec Vec3) AlongY() float64 { return vec.y }

// AlongZ is the part along the z axis, the spin axis of a hole.
func (vec Vec3) AlongZ() float64 { return vec.z }

// dot is the flat inner product of two directions.
func (vec Vec3) dot(other Vec3) float64 {
	return vec.x*other.x + vec.y*other.y + vec.z*other.z
}

// length is the flat length of a direction.
func (vec Vec3) length() float64 { return math.Sqrt(vec.dot(vec)) }

// cross is the cross product of two directions.
func (vec Vec3) cross(other Vec3) Vec3 {
	return Vec3{
		x: vec.y*other.z - vec.z*other.y,
		y: vec.z*other.x - vec.x*other.z,
		z: vec.x*other.y - vec.y*other.x,
	}
}

// scaled is the direction with each part times a factor.
func (vec Vec3) scaled(factor float64) Vec3 {
	return Vec3{x: vec.x * factor, y: vec.y * factor, z: vec.z * factor}
}

// Vec4 is a point in spacetime, or a covariant or contravariant vector
// there; which one is said where it is used. Its places are Time, AlongX,
// AlongY and AlongZ, in that order.
//
// It is a struct of four numbers and not an array because the compiler
// keeps a struct of up to four numbers in registers, in a function and
// between functions, and sends an array through memory; with arrays that
// traffic was more than half the cost of a step.
//
// Its parts are private so that its places are said by name. Any four
// numbers make one, a number that is not a number among them: that is how
// a place with no value is carried.
type Vec4 struct{ time, x, y, z float64 }

// NewVec4 is the vector with the parts time, x, y and z.
func NewVec4(time, x, y, z float64) Vec4 {
	return Vec4{time: time, x: x, y: y, z: z}
}

// Time is the part along time.
func (vec Vec4) Time() float64 { return vec.time }

// AlongX is the part along the x axis.
func (vec Vec4) AlongX() float64 { return vec.x }

// AlongY is the part along the y axis.
func (vec Vec4) AlongY() float64 { return vec.y }

// AlongZ is the part along the z axis, the spin axis of a hole.
func (vec Vec4) AlongZ() float64 { return vec.z }

// Space is the part in space of a point or a vector in spacetime.
func (vec Vec4) Space() Vec3 { return Vec3{x: vec.x, y: vec.y, z: vec.z} }

// parts are the four parts, in the order t, x, y, z, for loops that go
// over them.
func (vec Vec4) parts() [dimensions]float64 {
	return [dimensions]float64{vec.time, vec.x, vec.y, vec.z}
}

// scaled is the vector with each part times a factor.
func (vec Vec4) scaled(factor float64) Vec4 {
	return Vec4{
		time: vec.time * factor, x: vec.x * factor,
		y: vec.y * factor, z: vec.z * factor,
	}
}

// minus is the vector less another times a factor.
func (vec Vec4) minus(other Vec4, factor float64) Vec4 {
	return Vec4{
		time: vec.time - other.time*factor, x: vec.x - other.x*factor,
		y: vec.y - other.y*factor, z: vec.z - other.z*factor,
	}
}

// moved is a vector carried along a rate by a step.
func (vec Vec4) moved(rate Vec4, stepSize float64) Vec4 {
	return Vec4{
		time: vec.time + rate.time*stepSize, x: vec.x + rate.x*stepSize,
		y: vec.y + rate.y*stepSize, z: vec.z + rate.z*stepSize,
	}
}

// Mat4 is a metric or its inverse, by row and column in the order
// t, x, y, z.
type Mat4 [dimensions][dimensions]float64

// Dot is the inner product of a covariant and a contravariant vector.
func Dot(covector, vector Vec4) float64 {
	return covector.time*vector.time + covector.x*vector.x +
		covector.y*vector.y + covector.z*vector.z
}

// rungeKutta is the four rates of a runge-kutta step weighed together:
// six times the step's mean rate.
func rungeKutta(first, second, third, fourth Vec4) Vec4 {
	return Vec4{
		time: first.time + rungeKuttaDouble*(second.time+third.time) +
			fourth.time,
		x: first.x + rungeKuttaDouble*(second.x+third.x) + fourth.x,
		y: first.y + rungeKuttaDouble*(second.y+third.y) + fourth.y,
		z: first.z + rungeKuttaDouble*(second.z+third.z) + fourth.z,
	}
}
