package hole

import "math"

// Normal is the four-velocity of the observer normal to the chart's
// slices of constant time at a point: it carries no angular momentum, is
// carried round by the dragging and nothing else, and exists everywhere
// outside the horizon, ergosphere included, where Static does not.
//
// It is not still: in this chart it falls inward, slowly far out and
// faster close in, even round a still hole. Where Static exists, it is
// the one to prefer for a thing held in place.
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
