package mesh

import (
	"math"
	"slices"
	"sync"
)

// vector is a direction from the sphere's centre. x points to longitude
// and latitude none, y to longitude 90 and z to the north pole.
type vector struct {
	x, y, z float64
}

// vectorOf is the direction to a place.
func vectorOf(at Point) vector {
	sinLon, cosLon := math.Sincos(radians(at.lon))
	sinLat, cosLat := math.Sincos(radians(at.lat))
	return vector{x: cosLat * cosLon, y: cosLat * sinLon, z: sinLat}
}

// point is the place a direction points to.
func (way vector) point() Point {
	lon := degrees(math.Atan2(way.y, way.x))
	across := math.Hypot(way.x, way.y)
	lat := degrees(math.Atan2(way.z, across))
	return Point{lon: onceRound(lon), lat: lat}
}

// minus is one vector less another.
func (way vector) minus(other vector) vector {
	return vector{x: way.x - other.x, y: way.y - other.y,
		z: way.z - other.z}
}

// plus is two vectors added.
func (way vector) plus(other vector) vector {
	return vector{x: way.x + other.x, y: way.y + other.y,
		z: way.z + other.z}
}

// dot is how far one vector goes along another.
func (way vector) dot(other vector) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// cross is the vector square to two others, by the right hand.
func (way vector) cross(other vector) vector {
	return vector{x: way.y*other.z - way.z*other.y,
		y: way.z*other.x - way.x*other.z,
		z: way.x*other.y - way.y*other.x}
}

// unit is the vector's direction, one long.
func (way vector) unit() vector {
	size := math.Sqrt(way.dot(way))
	return vector{x: way.x / size, y: way.y / size, z: way.z / size}
}

// triangle is three corners on the sphere, counterclockwise as seen from
// outside it.
type triangle struct {
	first, second, third vector
}

// normal is square to the great circle from one corner to the next, by the
// right hand: the inner side of an edge taken counterclockwise. It is
// (from+to) cross (to-from), which is twice from cross to.
//
// Two close corners' plain cross product loses its digits: at the deepest
// cells, a hundred-millionth of a radian, as wide as the cells. Their
// difference keeps them.
func normal(from, to vector) vector {
	return from.plus(to).cross(to.minus(from))
}

// within is whether a direction falls in a triangle: on the inner side of
// each of its three edges' great circles, or on one.
func (way vector) within(shape triangle) bool {
	return way.dot(normal(shape.first, shape.second)) >= none &&
		way.dot(normal(shape.second, shape.third)) >= none &&
		way.dot(normal(shape.third, shape.first)) >= none
}

// margin is how far inside a triangle a direction lies: the least, over
// its three edges, of the sine of its angle past the edge's great circle.
// It is none or more inside the triangle and less outside it.
func (way vector) margin(shape triangle) float64 {
	return min(way.inside(shape.first, shape.second),
		way.inside(shape.second, shape.third),
		way.inside(shape.third, shape.first))
}

// inside is how far a direction lies on the inner side of the great
// circle from one corner to the next: the sine of its angle from it.
func (way vector) inside(from, to vector) float64 {
	side := normal(from, to).unit()
	return way.dot(side)
}

// nearest is which of some triangles a direction falls in. A direction
// within rounding of an edge can fall in none of them, since a parent's
// corners and its children's are rounded apart; then it is the one it is
// nearest to falling in.
func (way vector) nearest(shapes []triangle) int {
	if found := slices.IndexFunc(shapes, way.within); found >= none {
		return found
	}
	best := none
	for i := range shapes {
		if way.margin(shapes[i]) > way.margin(shapes[best]) {
			best = i
		}
	}
	return best
}

// split is the four triangles a triangle splits into at its edges'
// midpoints. One is at each corner in the corners' order. Then comes the
// middle.
func (shape triangle) split() [children]triangle {
	ab := shape.first.plus(shape.second).unit()
	bc := shape.second.plus(shape.third).unit()
	ca := shape.third.plus(shape.first).unit()
	return [children]triangle{{shape.first, ab, ca},
		{ab, shape.second, bc},
		{ca, bc, shape.third}, {ab, bc, ca}}
}

// excess is the triangle's area on the unit sphere: its spherical excess,
// by the formula of Van Oosterom and Strackee. Its triple product is taken
// over the edges from one corner, which keeps more of its digits for a
// small triangle than over the corners themselves.
func (shape triangle) excess() float64 {
	along := shape.second.minus(shape.first)
	across := shape.third.minus(shape.first)
	triple := shape.first.dot(along.cross(across))
	below := unitArea + shape.first.dot(shape.second) +
		shape.second.dot(shape.third) +
		shape.third.dot(shape.first)
	return twice * math.Atan2(triple, below)
}

// The numbers the excess is reckoned with. Each is defined by Van
// Oosterom and Strackee 1983.
const (
	unitArea = 1 // defined: the one in the formula's denominator
	twice    = 2 // defined: the excess is twice the angle found
)

// golden is the golden ratio: an icosahedron's corners are its cyclic
// placings with one. It's defined.
var golden = (unitArea + math.Sqrt(goldenRoot)) / twice

// goldenRoot is the square root's argument in the golden ratio. It's
// defined.
const goldenRoot = 5

// The icosahedron, built once.
var (
	built   sync.Once
	laidOut []triangle
)

// icosahedron is the twenty faces, each counterclockwise from outside. They
// are in a fixed order, by their corners in turn.
func icosahedron() []triangle {
	built.Do(build)
	return laidOut
}

// build lays the icosahedron out. Each of its twelve corners holds none,
// one either way and the golden ratio either way in its three parts in
// turn. Its faces are the threes of corners each nearest the others.
func build() {
	var tips []vector
	for _, first := range []float64{-unitArea, unitArea} {
		for _, second := range []float64{-golden, golden} {
			tips = append(tips,
				vector{x: none, y: first, z: second},
				vector{x: first, y: second, z: none},
				vector{x: second, y: none, z: first})
		}
	}
	for i := range tips {
		tips[i] = tips[i].unit()
	}
	nearest := math.Pi
	for _, other := range tips[firstTip+unitArea:] {
		apart := math.Acos(tips[firstTip].dot(other))
		nearest = math.Min(nearest, apart)
	}
	laidOut = facesOf(tips, nearest)
}

// firstTip is the first corner, against which the edge is measured. It's
// defined.
const firstTip = 0

// edgeSlack is how far an edge may differ from the nearest and still be
// one, in radians. It's defined: the next nearest corner is half again as
// far, so any slack far below that holds.
const edgeSlack = 1e-9

// facesOf is the triples of corners each an edge from the others. Each is
// turned counterclockwise from outside, and they come in order.
func facesOf(tips []vector, edge float64) []triangle {
	var found []triangle
	for i, one := range tips {
		later := tips[i+unitArea:]
		for next, two := range later {
			for _, three := range later[next+unitArea:] {
				shape := triangle{one, two, three}
				if shape.isFace(edge) {
					found = append(found, shape.outward())
				}
			}
		}
	}
	return found
}

// isFace is whether each of a triangle's corners is an edge from the
// others.
func (shape triangle) isFace(edge float64) bool {
	return isEdge(shape.first, shape.second, edge) &&
		isEdge(shape.second, shape.third, edge) &&
		isEdge(shape.third, shape.first, edge)
}

// isEdge is whether two corners are an edge apart.
func isEdge(one, other vector, edge float64) bool {
	apart := math.Acos(one.dot(other))
	return math.Abs(apart-edge) <= edgeSlack
}

// outward is the triangle with its corners counterclockwise from outside.
func (shape triangle) outward() triangle {
	if shape.first.dot(shape.second.cross(shape.third)) < none {
		return triangle{shape.first, shape.third, shape.second}
	}
	return shape
}

// radians is an angle in degrees, in radians.
func radians(angle float64) float64 { return angle * math.Pi / halfTurn }

// degrees is an angle in radians, in degrees.
func degrees(angle float64) float64 { return angle * halfTurn / math.Pi }
