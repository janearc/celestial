package hole

import "math"

// Vec3 stores a spatial position or direction with named component accessors.
// Construction permits NaN so invalid or undefined positions can propagate
// explicitly.
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

// Vec4 stores a spacetime point or vector, covariant or contravariant as
// specified by the caller. Component order is (t, x, y, z).
//
// The four-field struct lets the compiler keep values in registers within and
// between calls. In the measured array version, memory traffic accounted for
// more than half the cost of a step.
//
// Named accessors expose private components. NaN values are allowed to
// represent undefined states.
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

// Mat4 stores a metric or its inverse in (t, x, y, z) row and column order.
type Mat4 [dimensions][dimensions]float64

// Dot contracts a covector with a contravariant vector. It applies no
// additional metric sign; lowering supplies that beforehand.
func Dot(covector, vector Vec4) float64 {
	return covector.time*vector.time + covector.x*vector.x +
		covector.y*vector.y + covector.z*vector.z
}

// rungeKutta combines four derivatives with weights 1, 2, 2, 1, returning six
// times their weighted mean.
func rungeKutta(first, second, third, fourth Vec4) Vec4 {
	return Vec4{
		time: first.time + rungeKuttaDouble*(second.time+third.time) +
			fourth.time,
		x: first.x + rungeKuttaDouble*(second.x+third.x) + fourth.x,
		y: first.y + rungeKuttaDouble*(second.y+third.y) + fourth.y,
		z: first.z + rungeKuttaDouble*(second.z+third.z) + fourth.z,
	}
}
