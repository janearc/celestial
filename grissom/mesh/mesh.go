// Package mesh lays a sphere out in triangles. It begins from an
// icosahedron's twenty faces pushed out onto the sphere. Each triangle is
// split into four at its edges' midpoints, as deep as asked.
//
// Every edge is an arc of a great circle. Seen from the sphere's centre a
// great circle is a straight line, so each triangle looks flat from there:
// the sky as grissom draws it. The mesh says nothing of which sphere it
// lays out; its longitudes and latitudes are the caller's frame.
package mesh

import (
	"errors"
	"fmt"
	"math"
)

// The counts the mesh is built with.
const (
	// faces is how many faces an icosahedron has. It's defined.
	faces = 20
	// children is how many triangles each splits into. It's defined.
	children = 4
	// pathBits is the bits a step down takes in a path: one of four.
	// It's defined.
	pathBits = 2
	// depthBits is the bits a key gives the depth. It's defined: up to 31.
	depthBits = 5
	// faceBits is the bits a key gives the face. It's defined: up to 31,
	// past the twenty there are.
	faceBits = 5
	// keyBits is the width of a key. It's defined: a uint64.
	keyBits = 64
	// Deepest is the deepest a cell may be: as many steps as a key has
	// room for after its depth and face. It's defined by the key.
	Deepest = (keyBits - depthBits - faceBits) / pathBits
)

// The angles the mesh works with, in degrees.
const (
	halfTurn = 180 // defined: half a turn
	fullTurn = 360 // defined: a turn
	pole     = 90  // defined: the latitude of a pole
	none     = 0   // defined: no length, no depth, nothing
)

// errRefused is what Refused gives.
var errRefused = errors.New("refused")

// Refused is wrapped by each refusal here. Places off the sphere are
// refused. So are depths a key has no room for and keys no cell has.
func Refused() error { return errRefused }

// Point is a place on the sphere, by longitude and latitude in degrees in
// the caller's frame.
type Point struct {
	lon, lat float64
}

// NewPoint is the place at a longitude and latitude. It refuses a latitude
// past a pole, and numbers that are not finite.
func NewPoint(lon, lat float64) (*Point, error) {
	if math.IsNaN(lon) || math.IsInf(lon, none) || math.IsNaN(lat) ||
		lat < -pole || lat > pole {
		return nil, fmt.Errorf("%w: a place at %v, %v", errRefused, lon,
			lat)
	}
	return &Point{lon: onceRound(lon), lat: lat}, nil
}

// Lon is the point's longitude, from 0 to under 360 degrees.
func (place Point) Lon() float64 { return place.lon }

// Lat is the point's latitude, in degrees.
func (place Point) Lat() float64 { return place.lat }

// Cell is one triangle of the mesh. It is its depth and the face it began
// on, and the steps down from that face to it.
type Cell struct {
	depth, face int
	path        uint64
}

// Locate is the cell at a depth that a place falls in. Places exactly on
// an edge fall in one of the cells that share it, never in none.
func Locate(at *Point, depth int) (*Cell, error) {
	if at == nil || depth < none || depth > Deepest {
		return nil, fmt.Errorf("%w: a depth of %d, from none to %d",
			errRefused, depth, Deepest)
	}
	toward := vectorOf(*at)
	mesh := icosahedron()
	face := toward.nearest(mesh)
	cell := &Cell{face: face}
	triangle := mesh[face]
	for range depth {
		split := triangle.split()
		step := toward.nearest(split[:])
		cell.path = cell.path<<pathBits | uint64(step)
		cell.depth++
		triangle = split[step]
	}
	return cell, nil
}

// Depth is how many times the cell's face was split to reach it.
func (cell Cell) Depth() int { return cell.depth }

// triangle is the cell's corners, found by walking its path down from its
// face.
func (cell Cell) triangle() triangle {
	shape := icosahedron()[cell.face]
	for taken := range cell.depth {
		step := cell.depth - firstStep - taken
		which := cell.path >> (pathBits * step) & lastTwo
		shape = shape.split()[which]
	}
	return shape
}

// The bits a path is read with.
const (
	firstStep = 1 // defined: the last step sits one less than the depth
	lastTwo   = 3 // defined: the two low bits, one step of a path
)

// Corners are the cell's three corners, counterclockwise as seen from
// outside the sphere.
func (cell Cell) Corners() (Point, Point, Point) {
	shape := cell.triangle()
	return shape.first.point(), shape.second.point(), shape.third.point()
}

// Centre is the place the cell's corners are balanced about.
func (cell Cell) Centre() Point {
	shape := cell.triangle()
	middle := shape.first.plus(shape.second).plus(shape.third)
	return middle.unit().point()
}

// Area is the cell's area on the unit sphere in steradians. It is its
// corners' angles less half a turn.
func (cell Cell) Area() float64 {
	shape := cell.triangle()
	return shape.excess()
}

// Children are the four cells the cell splits into, in order.
func (cell Cell) Children() ([]*Cell, error) {
	if cell.depth >= Deepest {
		return nil, fmt.Errorf("%w: a cell at the deepest has no "+
			"children",
			errRefused)
	}
	split := make([]*Cell, children)
	for i := range split {
		split[i] = &Cell{depth: cell.depth + firstStep, face: cell.face,
			path: cell.path<<pathBits | uint64(i)}
	}
	return split, nil
}

// Faces are the twenty cells of depth none, in order.
func Faces() []*Cell {
	all := make([]*Cell, faces)
	for i := range all {
		all[i] = &Cell{face: i}
	}
	return all
}

// Key is the cell as one number. Its depth comes first in the high bits.
// Its face comes next, and its path last.
func (cell Cell) Key() uint64 {
	depthAt := keyBits - depthBits
	faceAt := depthAt - faceBits
	return uint64(cell.depth)<<depthAt | uint64(cell.face)<<faceAt |
		cell.path
}

// FromKey is the cell a key names. It refuses a key no cell has.
func FromKey(key uint64) (*Cell, error) {
	depthAt := keyBits - depthBits
	faceAt := depthAt - faceBits
	depth := int(key >> depthAt)
	face := int(key >> faceAt & (one<<faceBits - one))
	path := key & (one<<faceAt - one)
	if depth > Deepest || face >= faces || path>>(pathBits*depth) != none {
		return nil, fmt.Errorf("%w: no cell has the key %#x",
			errRefused,
			key)
	}
	return &Cell{depth: depth, face: face, path: path}, nil
}

// one is the low bit, for the masks a key is read with. It's defined.
const one = 1

// onceRound is a longitude brought from 0 to under 360.
func onceRound(lon float64) float64 {
	within := math.Mod(lon, fullTurn) + fullTurn
	return math.Mod(within, fullTurn)
}

// Parent is the cell the cell was split from. Faces have none, and are
// refused.
func (cell Cell) Parent() (*Cell, error) {
	if cell.depth == none {
		return nil, fmt.Errorf("%w: a face has no parent", errRefused)
	}
	return &Cell{depth: cell.depth - firstStep, face: cell.face,
		path: cell.path >> pathBits}, nil
}
