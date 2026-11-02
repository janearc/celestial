package hole

import "math"

// Normal returns the four-velocity normal to the chart’s constant-time slices.
// It has zero angular momentum, follows frame dragging, and exists outside the
// horizon, including the ergosphere.
//
// It is not stationary: even without spin it falls inward, slowly far away and
// faster nearby. Use Static where available for an object held at fixed
// coordinates.
func (hole Hole) Normal(position Vec3) Vec4 {
	scalar, null := hole.kernel(position)
	scale := whole / math.Sqrt(whole+scalar)
	return Vec4{
		time: scale * (whole + scalar),
		x:    -scale * scalar * null.x,
		y:    -scale * scalar * null.y,
		z:    -scale * scalar * null.z,
	}
}
