package mesh_test

import (
	"errors"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/puzzlemouth/numbers"
	"pgregory.net/rapid"
)

// The counts and angles the tests work with.
const (
	faces    = 20  // defined: an icosahedron's faces
	children = 4   // defined: each triangle splits into four
	sphere   = 4   // defined: the sphere is 4 pi steradians
	pole     = 90  // defined: the latitude of a pole, in degrees
	fullTurn = 360 // defined: a turn, in degrees
	halfTurn = 180 // defined: half a turn, in degrees
	none     = 0   // defined: no depth
	shallow  = 2   // defined: the depth every cell of is checked, 320 cells
	deepest  = 12  // defined: the deepest the properties draw, for speed
	past     = 1   // defined: a step past what is allowed
)

// picked is a picked number from the tests' numbers file.
func picked(t *testing.T, name string) float64 {
	t.Helper()
	raw, err := os.ReadFile("testdata/numbers.yaml")
	if err != nil {
		t.Fatal(err)
	}
	held, err := numbers.Read("numbers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	number, err := held.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// The twenty faces are alike, and together they are the whole sphere.
func TestTheFacesAreTheWholeSphere(t *testing.T) {
	within := picked(t, "area-rounding")
	all := mesh.Faces()
	if len(all) != faces {
		t.Fatalf("%d faces", len(all))
	}
	var total float64
	for _, face := range all {
		total += face.Area()
		if !(math.Abs(face.Area()-sphere*math.Pi/faces) <= within) {
			t.Errorf("a face of %v", face.Area())
		}
	}
	if !(math.Abs(total-sphere*math.Pi) <= within) {
		t.Errorf("the faces make %v", total)
	}
}

// Every cell at a shallow depth is itself when found again by its centre.
func TestEveryCellIsFoundByItsCentre(t *testing.T) {
	cells := mesh.Faces()
	for range shallow {
		var next []*mesh.Cell
		for _, cell := range cells {
			split, err := cell.Children()
			if err != nil {
				t.Fatal(err)
			}
			next = append(next, split...)
		}
		cells = next
	}
	for _, cell := range cells {
		centre := cell.Centre()
		found, err := mesh.Locate(&centre, shallow)
		if err != nil || found.Key() != cell.Key() {
			t.Errorf("%#x found as %v, %v", cell.Key(), found, err)
		}
	}
}

// drawPoint draws a place on the sphere.
func drawPoint(t *rapid.T) *mesh.Point {
	lon := rapid.Float64Range(none, fullTurn).Draw(t, "longitude")
	lat := rapid.Float64Range(-pole, pole).Draw(t, "latitude")
	place, err := mesh.NewPoint(lon, lat)
	if err != nil {
		t.Fatal(err)
	}
	return place
}

// angle is the angle between two places, in radians.
func angle(one, other mesh.Point) float64 {
	toRadians := math.Pi / halfTurn
	sinLat, cosLat := math.Sincos(one.Lat() * toRadians)
	sinOther, cosOther := math.Sincos(other.Lat() * toRadians)
	sinApart, cosApart := math.Sincos((other.Lon() - one.Lon()) * toRadians)
	east := cosOther * sinApart
	north := cosLat*sinOther - sinLat*cosOther*cosApart
	toward := sinLat*sinOther + cosLat*cosOther*cosApart
	return math.Atan2(math.Hypot(east, north), toward)
}

// fallsIn is the property of TestEveryPlaceFallsInItsCell, for one draw.
// The place is no farther from its cell's centre than the farthest
// corner. Its cell one step deeper has its cell for a parent.
func fallsIn(t *rapid.T) {
	place := drawPoint(t)
	depth := rapid.IntRange(none, deepest).Draw(t, "depth")
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		t.Fatal(err)
	}
	centre := cell.Centre()
	one, two, three := cell.Corners()
	reach := math.Max(angle(centre, one), angle(centre, two))
	reach = math.Max(reach, angle(centre, three))
	if !(angle(centre, *place) <= reach+slack) {
		t.Fatalf("%v lies outside %#x", *place, cell.Key())
	}
	deeper, err := mesh.Locate(place, depth+past)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := deeper.Parent()
	if err != nil || parent.Key() != cell.Key() {
		t.Fatalf("%#x's parent is %v, %v", deeper.Key(), parent, err)
	}
}

// slack is the angle rounding, for the properties. Each test that runs
// one fills it.
var slack float64

// Property: every place falls in its cell, and its cell one step deeper
// lies inside that one.
func TestEveryPlaceFallsInItsCell(t *testing.T) {
	slack = picked(t, "angle-rounding")
	rapid.Check(t, fallsIn)
}

// splitsWhole is the property of TestChildrenAreTheirParent, for one draw.
func splitsWhole(t *rapid.T) {
	place := drawPoint(t)
	depth := rapid.IntRange(none, deepest).Draw(t, "depth")
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		t.Fatal(err)
	}
	split, err := cell.Children()
	if err != nil || len(split) != children {
		t.Fatalf("%v, %v", split, err)
	}
	var total float64
	for _, child := range split {
		total += child.Area()
	}
	if !(math.Abs(total-cell.Area()) <= share*cell.Area()) {
		t.Fatalf("%#x's children make %v of %v", cell.Key(), total,
			cell.Area())
	}
}

// Property: a cell's four children cover just the cell.
func TestChildrenAreTheirParent(t *testing.T) {
	share = picked(t, "area-share")
	rapid.Check(t, splitsWhole)
}

// keysBack is the property of TestEveryKeyComesBack, for one draw.
func keysBack(t *rapid.T) {
	place := drawPoint(t)
	depth := rapid.IntRange(none, mesh.Deepest).Draw(t, "depth")
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		t.Fatal(err)
	}
	back, err := mesh.FromKey(cell.Key())
	if err != nil || back.Key() != cell.Key() ||
		back.Depth() != cell.Depth() {
		t.Fatalf("%#x came back %v, %v", cell.Key(), back, err)
	}
}

// Property: every cell's key names it again, to the deepest depth.
func TestEveryKeyComesBack(t *testing.T) {
	rapid.Check(t, keysBack)
}

// Anti-tests: places off the sphere are refused. So are depths a key has
// no room for and keys no cell has. So are a face's parent and the
// children of a deepest cell.
func TestWhatIsNotACellIsRefused(t *testing.T) {
	for _, each := range []struct{ lon, lat float64 }{
		{none, pole + past}, {none, math.NaN()}, {math.Inf(past), none},
	} {
		if _, err := mesh.NewPoint(each.lon, each.lat); !errors.Is(err,
			mesh.Refused()) {
			t.Errorf("%v: %v", each, err)
		}
	}
	place, err := mesh.NewPoint(none, none)
	if err != nil {
		t.Fatal(err)
	}
	for _, depth := range []int{-past, mesh.Deepest + past} {
		if _, err := mesh.Locate(place, depth); !errors.Is(err,
			mesh.Refused()) {
			t.Errorf("depth %d: %v", depth, err)
		}
	}
	if _, err := mesh.Locate(nil, none); !errors.Is(err, mesh.Refused()) {
		t.Errorf("no place: %v", err)
	}
	refusedKeys(t)
	face := mesh.Faces()[none]
	if _, err := face.Parent(); !errors.Is(err, mesh.Refused()) {
		t.Errorf("a face's parent: %v", err)
	}
	bottom, err := mesh.Locate(place, mesh.Deepest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bottom.Children(); !errors.Is(err, mesh.Refused()) {
		t.Errorf("the deepest's children: %v", err)
	}
}

// The parts of a key, as the tests build crooked ones. Each is defined by
// Key's layout: the depth in the top five bits, the face in the next five.
const (
	depthAt = 59 // defined: 64 less the depth's five bits
	faceAt  = 54 // defined: 59 less the face's five bits
)

// refusedKeys checks that keys no cell has are refused. Each is one of
// three: a depth past the deepest; a face past the twentieth; a path
// longer than its depth.
func refusedKeys(t *testing.T) {
	t.Helper()
	for why, key := range map[string]uint64{
		"too deep":    uint64(mesh.Deepest+past) << depthAt,
		"no face":     uint64(faces) << faceAt,
		"a long path": uint64(past),
	} {
		if _, err := mesh.FromKey(key); !errors.Is(err, mesh.Refused()) {
			t.Errorf("%s: %v", why, err)
		}
	}
}

// share is the area share, for the property. TestChildrenAreTheirParent
// fills it.
var share float64

// The bits a key is read with in the pin. Each is defined by Key's
// layout.
const (
	faceMask = 31 // defined: the face's five bits
	aside    = 10 // defined: a longitude ten degrees past a turn's ends
)

// The layout is pinned. Sgr A*'s cell at the pinned depth has the face
// and path it had when the page was first given keys. Changing the
// icosahedron built or the order of its faces moves every key. So does a
// change to how a key is laid out. This test says so.
func TestTheLayoutIsPinned(t *testing.T) {
	place, err := mesh.NewPoint(picked(t, "sgra-ra"), picked(t, "sgra-dec"))
	if err != nil {
		t.Fatal(err)
	}
	depth := int(picked(t, "pinned-depth"))
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		t.Fatal(err)
	}
	key := cell.Key()
	face := key >> faceAt & faceMask
	path := key & (uint64(past)<<faceAt - uint64(past))
	if face != uint64(picked(t, "pinned-face")) ||
		path != uint64(picked(t, "pinned-path")) {
		t.Errorf("face %d, path %d", face, path)
	}
}

// Longitudes are brought into one turn from none to under 360 however
// they are given. So is a cell's centre.
func TestLongitudesAreWithinOneTurn(t *testing.T) {
	for given, want := range map[float64]float64{fullTurn + aside: aside,
		-aside: fullTurn - aside, none: none} {
		place, err := mesh.NewPoint(given, none)
		if err != nil || place.Lon() != want {
			t.Errorf("%v: %v, %v", given, place, err)
		}
	}
	for _, face := range mesh.Faces() {
		centre := face.Centre()
		if centre.Lon() < none || centre.Lon() >= fullTurn {
			t.Errorf("a centre at %v", centre.Lon())
		}
	}
}

// Places exactly on an edge fall in a cell that holds them. Some edges of
// the icosahedron lie on the equator, where a place's height is none to
// the last bit. Places at every quarter turn along it are found, and at
// the poles, each inside each edge of its cell.
func TestPlacesOnEdgesFallInACell(t *testing.T) {
	slack = picked(t, "angle-rounding")
	var places []*mesh.Point
	for lon := range quarterTurns {
		place, err := mesh.NewPoint(float64(lon)*pole, none)
		if err != nil {
			t.Fatal(err)
		}
		places = append(places, place)
	}
	for _, lat := range []float64{-pole, pole} {
		place, err := mesh.NewPoint(none, lat)
		if err != nil {
			t.Fatal(err)
		}
		places = append(places, place)
	}
	for _, place := range places {
		for depth := range mesh.Deepest + past {
			liesIn(t, place, depth)
		}
	}
}

// quarterTurns is how many quarter turns make one. It's defined.
const quarterTurns = 4

// whole is the whole way from one place to another. It's defined.
const whole = 1

// firstCut is the first cut along an edge past its end, a corner. It's
// defined.
const firstCut = 1

// fatal is what a check needs of a test, rapid's or testing's.
type fatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

// liesIn checks that a place is found at a depth, and lies in the cell it
// is found in: on the inner side of each of its edges, taken from corner
// to corner counterclockwise, or past one by no more than slack.
func liesIn(check fatal, place *mesh.Point, depth int) {
	check.Helper()
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		check.Fatalf("%v at %d: %v", *place, depth, err)
	}
	one, two, three := cell.Corners()
	for _, side := range []edge{{one, two}, {two, three}, {three, one}} {
		if past := beyond(*place, side); past > slack {
			check.Fatalf("%v lies %g radians outside %#x at %d", *place,
				past, cell.Key(), depth)
		}
	}
}

// edge is a cell's edge, from one corner to the next counterclockwise.
type edge struct{ from, to mesh.Point }

// heading is a place as a direction from the sphere's centre: x toward
// longitude and latitude none, y toward longitude 90, z toward the north
// pole.
type heading struct{ x, y, z float64 }

// headingOf is the direction to a place.
func headingOf(place mesh.Point) heading {
	toRadians := math.Pi / halfTurn
	sinLon, cosLon := math.Sincos(place.Lon() * toRadians)
	sinLat, cosLat := math.Sincos(place.Lat() * toRadians)
	return heading{x: cosLat * cosLon, y: cosLat * sinLon, z: sinLat}
}

// plus is two headings added.
func (way heading) plus(other heading) heading {
	return heading{x: way.x + other.x, y: way.y + other.y,
		z: way.z + other.z}
}

// minus is one heading less another.
func (way heading) minus(other heading) heading {
	return heading{x: way.x - other.x, y: way.y - other.y,
		z: way.z - other.z}
}

// times is a heading scaled.
func (way heading) times(share float64) heading {
	return heading{x: way.x * share, y: way.y * share, z: way.z * share}
}

// cross is the heading square to two others, by the right hand.
func (way heading) cross(other heading) heading {
	return heading{x: way.y*other.z - way.z*other.y,
		y: way.z*other.x - way.x*other.z,
		z: way.x*other.y - way.y*other.x}
}

// dot is how far one heading goes along another.
func (way heading) dot(other heading) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// beyond is how far a place lies past the great circle along an edge,
// outward of a triangle turned counterclockwise: the sine of its angle
// past it, none or less inside. The circle's normal is (from+to) cross
// (to-from), which keeps its digits when the corners are close.
func beyond(place mesh.Point, side edge) float64 {
	from, to := headingOf(side.from), headingOf(side.to)
	square := from.plus(to).cross(to.minus(from))
	size := math.Sqrt(square.dot(square))
	return -headingOf(place).dot(square) / size
}

// foundDeep is the property of TestEveryPlaceIsFoundAtEveryDepth, for one
// draw: a place at a depth from deepest down to the deepest a key holds.
func foundDeep(t *rapid.T) {
	place := drawPoint(t)
	depth := rapid.IntRange(deepest, mesh.Deepest).Draw(t, "depth")
	liesIn(t, place, depth)
}

// Every place is found at every depth a key holds, and lies in its cell.
// The properties above stop at deepest for speed. Past it, cells are a
// few hundred-millionths of a radian across and rounding is felt.
func TestEveryPlaceIsFoundAtEveryDepth(t *testing.T) {
	slack = picked(t, "angle-rounding")
	rapid.Check(t, foundDeep)
}

// between is the place a share of the way along an edge, on the great
// circle through its corners: their headings weighed and summed.
func between(t *testing.T, side edge, share float64) *mesh.Point {
	t.Helper()
	from := headingOf(side.from).times(whole - share)
	to := headingOf(side.to).times(share)
	sum := from.plus(to)
	toDegrees := halfTurn / math.Pi
	lon := math.Atan2(sum.y, sum.x) * toDegrees
	lat := math.Atan2(sum.z, math.Hypot(sum.x, sum.y)) * toDegrees
	place, err := mesh.NewPoint(lon, lat)
	if err != nil {
		t.Fatal(err)
	}
	return place
}

// Places along the faces' edges are found at every depth. Such a place
// can pass its face's test and then fall outside all four of a cell's
// children, whose corners are rounded apart from their parent's; Locate
// then takes the child it is nearest to falling in.
func TestPlacesAlongEdgesAreFoundAtEveryDepth(t *testing.T) {
	slack = picked(t, "angle-rounding")
	steps := int(picked(t, "edge-steps"))
	for _, face := range mesh.Faces() {
		one, two, three := face.Corners()
		for _, side := range []edge{{one, two}, {two, three}, {three, one}} {
			for step := firstCut; step < steps; step++ {
				share := float64(step) / float64(steps)
				place := between(t, side, share)
				for depth := range mesh.Deepest + past {
					liesIn(t, place, depth)
				}
			}
		}
	}
}
