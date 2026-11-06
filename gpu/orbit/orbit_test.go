//go:build darwin

package orbit_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"

	"github.com/janearc/celestial/gpu"
	gpuorbit "github.com/janearc/celestial/gpu/orbit"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/mocks"
	"github.com/janearc/celestial/orbit"
	"github.com/janearc/celestial/orbit/orbittest"
	"github.com/janearc/celestial/research"
)

// picked is the numbers these tests use, from testdata/orbit.json.
type picked struct {
	CentreSuns float64 `json:"centre suns"`
	Room       int
	Circle     struct {
		Radius     float64
		Steps      int
		Room       float64
		NearRoom   float64 `json:"near room"`
		MovedRoom  float64 `json:"moved room"`
		PulledRoom float64 `json:"pulled room"`
	}
	Ellipse struct {
		Axis         float64
		Eccentricity float64
		Steps        int
		NearRoom     float64 `json:"near room"`
		AgoSteps     float64 `json:"ago steps"`
		EndRoom      float64 `json:"end room"`
		ShareRoom    float64 `json:"share room"`
	}
	Line struct {
		AlongX     float64 `json:"x"`
		AlongY     float64 `json:"y"`
		AlongZ     float64 `json:"z"`
		VX, VY, VZ float64
		Span       float64
		Steps      int
		Near, Ago  float64
		EndY       float64 `json:"end y"`
		Room       float64
		EndRoom    float64 `json:"end room"`
	}
	Agree  struct{ Room, Resolved, Compared, Worst float64 }
	Parity struct{ Room float64 }
	Galaxy struct {
		HoleSuns float64 `json:"hole suns"`
	}
	Inner struct {
		Radius     float64
		Steps      int
		Room       float64
		NearRoom   float64 `json:"near room"`
		MovedRoom  float64 `json:"moved room"`
		PulledRoom float64 `json:"pulled room"`
	} `json:"inner circle"`
	Layout struct{ Star, Pass, Place, Bytes int }
	Beyond struct {
		Speed float64
		Span  float64
		Steps int
	}
	Commands int
	Patience float64 `json:"patience seconds"`
	Heavy    load
	Heavier  load
	Some     struct {
		Span  float64
		Steps int
	}
	None  float64
	One   int
	Two   int
	Huge  int
	Drawn struct {
		Farthest     float64
		Nearest      float64
		Fastest      float64
		ShortestSpan float64 `json:"shortest span"`
		LongestSpan  float64 `json:"longest span"`
		FewestSteps  int     `json:"fewest steps"`
		MostSteps    int     `json:"most steps"`
		FewestStars  int     `json:"fewest stars"`
	}
	Resolved struct{ Factor, Floor float64 }
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test testing.TB) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/orbit.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// no pull, in Metal: a star moves in a straight line
const freePull = `
static inline float3 pull(float3 at, constant float *field) {
	return float3(field[gm]);
}
`

// centre is the point mass the tests orbit, as a field.
func centre(numbers picked) gpuorbit.Field {
	return gpuorbit.Kepler(orbittest.Centre(numbers.CentreSuns))
}

// gm is a kepler field's gm.
func gm(field gpuorbit.Field) float64 {
	return field.Reference().(orbit.Kepler).GM()
}

// free is no pull at all: the free pull in Metal, whose one number is
// nothing, and a mock in Go that pulls nothing.
func free(test *testing.T, numbers picked) gpuorbit.Field {
	reference := mocks.NewMockPotential(gomock.NewController(test))
	none := orbit.NewPull(numbers.None, numbers.None, numbers.None)
	reference.EXPECT().Pull(gomock.Any()).Return(none).AnyTimes()
	return gpuorbit.NewField(freePull, []float32{float32(numbers.None)},
		reference)
}

// solver opens a solver over a field on the gpu with room for so many
// stars, or skips with no gpu.
func solver(test *testing.T, field gpuorbit.Field,
	most int) *gpuorbit.Solver {
	test.Helper()
	made, err := gpuorbit.Open(gpu.Metal(), field, most)
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(made.Close)
	return made
}

// apart is the distance between two stars' places, in kiloparsecs.
func apart(from, to orbit.State) float64 {
	fx, fy, fz := from.Numbers().At()
	tx, ty, tz := to.Numbers().At()
	return orbittest.Size(tx-fx, ty-fy, tz-fz)
}

// distance is how far a star is from the centre, in kiloparsecs.
func distance(star orbit.State) float64 {
	return orbittest.Size(star.Numbers().At())
}

// speed is how fast a star moves, in kilometres per second.
func speed(star orbit.State) float64 {
	return orbittest.Size(star.Numbers().Moving())
}

// speedApart is the difference between two stars' motions, in
// kilometres per second.
func speedApart(from, to orbit.State) float64 {
	fx, fy, fz := from.Numbers().Moving()
	tx, ty, tz := to.Numbers().Moving()
	return orbittest.Size(tx-fx, ty-fy, tz-fz)
}

// still is a star at the line's place, not moving.
func still(numbers picked) orbit.State {
	return orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY,
		numbers.Line.AlongZ, numbers.None, numbers.None, numbers.None)
}

// only is the one star of a batch of one.
func only(test *testing.T, numbers picked, made *gpuorbit.Solver,
	star orbit.State, span float64, steps int) (orbit.Pass, orbit.State) {
	test.Helper()
	passes, ends, err := made.Back([]orbit.State{star}, orbit.Units(span),
		steps)
	if err != nil {
		test.Fatal(err)
	}
	first := int(numbers.None)
	return passes[first], ends[first]
}

// circling runs a circle of a radius back one period on the gpu in so
// many steps and holds its end to its start within a room and its
// nearest pass to its radius within another: 32-bit rounding over the
// steps, a few hundred times the leapfrog's own error, measured in
// testdata/measured.json. The pass's shares are a circle's, each
// within its own room: in the motion, the step's share of a turn; in
// the pull, that share squared. It logs what it measured.
func circling(test *testing.T, numbers picked, made *gpuorbit.Solver,
	field gpuorbit.Field, radius float64, steps int, room, nearRoom,
	movedRoom, pulledRoom float64) {
	test.Helper()
	fast := orbittest.Circular(gm(field), radius)
	start := orbit.StateOf(radius, numbers.None, numbers.None, numbers.None,
		fast, numbers.None)
	period := orbittest.Period(gm(field), radius)
	pass, end := only(test, numbers, made, start, period, steps)
	off := apart(start, end)
	near, _ := pass.Numbers()
	moved, pulled := pass.Steps()
	stepOfTurn := turn / float64(steps)
	test.Logf("circle of %g kpc: end off %g kpc, nearest off %g kpc, moved"+
		" off %g, pulled off %g", radius, off, near-radius, moved-stepOfTurn,
		pulled-moved*moved)
	if off > room {
		test.Errorf("after one period the %g kpc circle is %g kpc from its"+
			" start", radius, off)
	}
	if !orbittest.Within(near, radius, nearRoom) {
		test.Errorf("a circle of %g kpc came %g kpc from the centre",
			radius, near)
	}
	if !orbittest.Within(moved, stepOfTurn, movedRoom) ||
		!orbittest.Within(pulled, moved*moved, pulledRoom) {
		test.Errorf("a circle of %g kpc has shares %g and %g, want %g and"+
			" its square", radius, moved, pulled, stepOfTurn)
	}
}

// turn is a whole turn in radians. Defined.
const turn = 2 * math.Pi

// firstAway is the first of the stars a Runaway names: the place in the
// batch of the one star these tests send through the centre.
const firstAway = 0 // defined: the first named

// a circular orbit run back one period comes back where it started.
func TestACircleComesBackToItsStart(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	made := solver(test, field, numbers.One)
	circling(test, numbers, made, field, numbers.Circle.Radius,
		numbers.Circle.Steps, numbers.Circle.Room, numbers.Circle.NearRoom,
		numbers.Circle.MovedRoom, numbers.Circle.PulledRoom)
	if made.GPUTime() <= time.Duration(numbers.None) {
		test.Error("the gpu spent no time on the orbit")
	}
}

// a circle a parsec from the centre, the search's own distances, comes
// back to its start in 32 bits as the kiloparsec one does, its errors
// scaled down with it.
func TestACircleAParsecOutComesBackToItsStart(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	made := solver(test, field, numbers.One)
	circling(test, numbers, made, field, numbers.Inner.Radius,
		numbers.Inner.Steps, numbers.Inner.Room, numbers.Inner.NearRoom,
		numbers.Inner.MovedRoom, numbers.Inner.PulledRoom)
}

// the test and the anti-test: an ellipse begun at apocentre and run back
// one period passes pericentre half a period ago, at its own distance
// from the centre, and comes back to its start. its shares are the
// pericentre's, where the star is fastest and nearest, and not the
// start's: a pass that kept the apocentre's would be nine times too
// small in the motion. It logs what it measured.
func TestAnEllipsePassesPericentreHalfAPeriodAgo(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	made := solver(test, field, numbers.One)
	axis, eccentricity := numbers.Ellipse.Axis, numbers.Ellipse.Eccentricity
	far, fast := orbittest.Apocentre(gm(field), axis, eccentricity)
	start := orbit.StateOf(far, numbers.None, numbers.None, numbers.None,
		fast, numbers.None)
	period := orbittest.Period(gm(field), axis)
	pass, end := only(test, numbers, made, start, period,
		numbers.Ellipse.Steps)
	near, ago := pass.Numbers()
	want, fastest := orbittest.Pericentre(gm(field), axis, eccentricity)
	dt := period / float64(numbers.Ellipse.Steps)
	off := apart(start, end)
	moved, pulled := pass.Steps()
	wantMoved, wantPulled := orbittest.Shares(gm(field), fastest, want, dt)
	test.Logf("ellipse: pericentre off %g kpc and %g steps, end off %g kpc,"+
		" moved off %g of %g, pulled off %g of %g", near-want,
		(ago-orbittest.Half(period))/dt, off, moved-wantMoved, wantMoved,
		pulled-wantPulled, wantPulled)
	if !orbittest.Within(near, want, numbers.Ellipse.NearRoom) {
		test.Errorf("pericentre at %g kpc, want %g", near, want)
	}
	if !orbittest.Within(moved, wantMoved,
		numbers.Ellipse.ShareRoom*wantMoved) || !orbittest.Within(pulled,
		wantPulled, numbers.Ellipse.ShareRoom*wantPulled) {
		test.Errorf("the pericentre's shares are %g and %g, want %g and %g",
			moved, pulled, wantMoved, wantPulled)
	}
	if !orbittest.Within(ago, orbittest.Half(period),
		numbers.Ellipse.AgoSteps*dt) {
		test.Errorf("pericentre %g units ago, want %g", ago,
			orbittest.Half(period))
	}
	if off > numbers.Ellipse.EndRoom {
		test.Errorf("after one period the ellipse is %g kpc from its start",
			off)
	}
}

// with no pull a star moves in a straight line: a field of the test's
// own, through NewField. its share in the motion is the step's length
// over the pass's distance, and its share in the pull is nothing.
func TestAFreeStarCameInAStraightLine(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, free(test, numbers), numbers.One)
	line := numbers.Line
	start := orbit.StateOf(line.AlongX, line.AlongY, line.AlongZ, line.VX, line.VY,
		line.VZ)
	pass, end := only(test, numbers, made, start, line.Span, line.Steps)
	near, ago := pass.Numbers()
	if !orbittest.Within(near, line.Near, line.Room) ||
		!orbittest.Within(ago, line.Ago, line.Room) {
		test.Errorf("the line passed %g kpc from the centre %g units ago,"+
			" want %g and %g", near, ago, line.Near, line.Ago)
	}
	freeShares(test, numbers, pass, speed(start), line.Near, line.Room)
	then := orbit.StateOf(line.AlongX, line.EndY, line.AlongZ, line.VX, line.VY,
		line.VZ)
	if off := apart(end, then); off > line.EndRoom {
		test.Errorf("at the start of the span the star was %g kpc off", off)
	}
}

// freeShares holds a free star's pass to its shares: in the motion, the
// step's length at the star's speed over the pass's distance, within a
// room; in the pull, nothing at all.
func freeShares(test *testing.T, numbers picked, pass orbit.Pass,
	fast, near, room float64) {
	test.Helper()
	dt := numbers.Line.Span / float64(numbers.Line.Steps)
	moved, pulled := pass.Steps()
	wantMoved, wantPulled := orbittest.Shares(numbers.None, fast, near, dt)
	if !orbittest.Within(moved, wantMoved, room) || pulled != wantPulled {
		test.Errorf("a free star's shares are %g and %g, want %g and %g",
			moved, pulled, wantMoved, wantPulled)
	}
}

// the anti-test: a free star moving away from the centre passed
// nearest now: run back it only recedes, so its pass is where it is,
// with the start's shares. before its first step the kernel has no
// pass, and this is the start found as the first.
func TestAFreeStarMovingAwayPassedNow(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, free(test, numbers), numbers.One)
	line := numbers.Line
	start := orbit.StateOf(line.AlongX, line.AlongY, line.AlongZ, line.VX,
		-line.VY, line.VZ)
	pass, _ := only(test, numbers, made, start, line.Span, line.Steps)
	near, ago := pass.Numbers()
	if !orbittest.Within(near, distance(start), line.Room) ||
		ago != numbers.None {
		test.Errorf("a star moving away passed %g kpc out %g units ago,"+
			" want %g and now", near, ago, distance(start))
	}
	freeShares(test, numbers, pass, speed(start), distance(start),
		line.Room)
}

// the anti-test: a pass through the centre itself has no finite shares,
// since no step resolves it: a free star at the centre, moving, passed
// now at no distance, and its shares are not numbers to hold anything
// to, in 32 bits as in 64.
func TestAPassThroughTheCentreHasNoFiniteShares(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, free(test, numbers), numbers.One)
	line := numbers.Line
	start := orbit.StateOf(numbers.None, numbers.None, numbers.None,
		line.VX, line.VY, line.VZ)
	pass, _ := only(test, numbers, made, start, line.Span, line.Steps)
	near, ago := pass.Numbers()
	if near != numbers.None || ago != numbers.None {
		test.Errorf("a star at the centre passed %g kpc out %g units ago",
			near, ago)
	}
	if moved, pulled := pass.Steps(); finite(moved) || finite(pulled) {
		test.Errorf("a pass through the centre has shares %g and %g", moved,
			pulled)
	}
}

// finite says whether a number is one, and not infinite.
func finite(number float64) bool {
	return math.Abs(number) <= math.MaxFloat64
}

// a field carries its reference, and a kepler field the body's gm.
func TestAFieldCarriesItsReference(test *testing.T) {
	numbers := pickedNumbers(test)
	body := orbittest.Centre(numbers.CentreSuns)
	field := gpuorbit.Kepler(body)
	if got, want := gm(field), orbit.NewKepler(body).GM(); got != want {
		test.Errorf("the field's gm is %g, the body's %g", got, want)
	}
	made := solver(test, field, numbers.One)
	if made.Field().Reference() != field.Reference() {
		test.Error("the solver's field is not the one it was opened over")
	}
}

// the anti-tests: what the solver refuses.

// the anti-test: room for no stars is refused, and says so.
func TestRoomForNoStarsIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	_, err := gpuorbit.Open(gpu.Metal(), centre(numbers),
		int(numbers.None))
	if err == nil || !strings.Contains(err.Error(), "room") {
		test.Errorf("a solver with room for no stars: %v", err)
	}
}

// the anti-test: a star beyond a float runs away.
// a star faster than a 32-bit float can add to itself runs away: its
// place is infinite, and infinite either way is no place.
func TestAStarBeyondAFloatRunsAway(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, free(test, numbers), numbers.One)
	stars := []orbit.State{orbit.StateOf(numbers.Line.AlongX,
		numbers.Line.AlongY, numbers.Line.AlongZ, numbers.Beyond.Speed,
		numbers.None, numbers.None)}
	_, _, err := made.Back(stars, orbit.Units(numbers.Beyond.Span),
		numbers.Beyond.Steps)
	away, ran := gpuorbit.RanAway(err)
	if err == nil || !ran || len(away) != numbers.One ||
		!strings.Contains(err.Error(), "star 0 ran away") {
		test.Errorf("a star beyond a float gave %v, %v", err, away)
	}
	if away, ran := gpuorbit.RanAway(errors.New("not a runaway")); ran ||
		away != nil {
		test.Errorf("a plain error is a runaway: %v", away)
	}
}

// the anti-test: room for more stars than the gpu holds is refused.
func TestRoomForMoreStarsThanTheGPUHoldsIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	_, err := gpuorbit.Open(gpu.Metal(), centre(numbers),
		numbers.Huge)
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err == nil {
		test.Error("a solver with room for terabytes of stars opened")
	}
}

// the anti-test: a pull that does not compile is refused.
func TestAPullThatDoesNotCompileIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	broken := gpuorbit.NewField("float3 pull(", nil, nil)
	_, err := gpuorbit.Open(gpu.Metal(), broken, numbers.One)
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err == nil {
		test.Error("a pull that does not compile opened")
	}
}

// the anti-test: no steps and no span are refused.
func TestNoStepsAndNoSpanAreRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.One)
	stars := []orbit.State{orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY,
		numbers.Line.AlongZ, numbers.None, numbers.None, numbers.None)}
	if _, _, err := made.Back(stars, orbit.Units(numbers.Some.Span),
		int(numbers.None)); err == nil {
		test.Error("no steps were taken")
	}
	if _, _, err := made.Back(stars, orbit.Units(numbers.None),
		numbers.Some.Steps); err == nil {
		test.Error("a span of nothing was run")
	}
}

// the anti-test: more stars than there is room for are refused.
func TestMoreStarsThanRoomAreRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.One)
	star := orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY, numbers.Line.AlongZ,
		numbers.None, numbers.None, numbers.None)
	stars := []orbit.State{star, star}
	if _, _, err := made.Back(stars, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps); err == nil {
		test.Error("two stars ran in a solver with room for one")
	}
}

// the anti-test: no stars is no work, and no error.
func TestNoStarsIsNoWork(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.One)
	passes, ends, err := made.Back(nil, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps)
	if err != nil || len(passes) != int(numbers.None) ||
		len(ends) != int(numbers.None) {
		test.Errorf("no stars gave %v, %v, %v", passes, ends, err)
	}
}

// the anti-test: a star at the centre runs away.
func TestAStarAtTheCentreRunsAway(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.Two)
	stars := []orbit.State{
		orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY, numbers.Line.AlongZ,
			numbers.None, numbers.None, numbers.None),
		orbit.StateOf(numbers.None, numbers.None, numbers.None,
			numbers.None, numbers.None, numbers.None),
	}
	_, _, err := made.Back(stars, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps)
	away, ran := gpuorbit.RanAway(err)
	if err == nil || !ran || len(away) != numbers.One ||
		away[firstAway] != numbers.One {
		test.Errorf("a star at the centre gave %v, %v", err, away)
	}
	both := gpuorbit.NewRunaway([]int{firstAway, numbers.One})
	if !strings.Contains(both.Error(), "2 stars ran away, the first star 0") {
		test.Errorf("two runaways say %q", both.Error())
	}
}

// the anti-test: a closed solver refuses to run.
func TestAClosedSolverRefusesToRun(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.One)
	made.Close()
	made.Close()
	stars := []orbit.State{orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY,
		numbers.Line.AlongZ, numbers.None, numbers.None, numbers.None)}
	if _, _, err := made.Back(stars, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps); err == nil {
		test.Error("a closed solver ran")
	}
}

// what the device says when it fails, the solver says: the mock device
// stands in for a gpu that will not.

// opened is a mock device that opens, and closes once.
func opened(test *testing.T, numbers picked) *mocks.MockDevice {
	device := mocks.NewMockDevice(gomock.NewController(test))
	device.EXPECT().Open(gomock.Any(), gomock.Any()).Return(false, "")
	device.EXPECT().Close().Times(numbers.One)
	return device
}

// bytes is a device's buffers in go: each is its bytes, numbered none.
type bytes struct{ numbers picked }

// buffer is a buffer of so many bytes.
func (made bytes) buffer(count int) (int, []byte, string) {
	return int(made.numbers.None), make([]byte, count), ""
}

// kept is a device's buffers in go, kept for a test to read after.
type kept struct {
	numbers picked
	made    [][]byte
}

// buffer is a buffer of so many bytes, kept.
func (held *kept) buffer(count int) (int, []byte, string) {
	held.made = append(held.made, make([]byte, count))
	return int(held.numbers.None), held.made[len(held.made)-held.numbers.One],
		""
}

// the anti-test: a closed solver writes nothing into its buffers and
// asks nothing of its device. the refusal comes first, not after a
// star has been written where the device no longer is.
func TestAClosedSolverWritesNothing(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test, numbers)
	held := &kept{numbers: numbers}
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(held.buffer).AnyTimes()
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	made.Close()
	_, _, err = made.Back([]orbit.State{still(numbers)},
		orbit.Units(numbers.Some.Span), numbers.Some.Steps)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		test.Errorf("a closed solver ran: %v", err)
	}
	stars, found := held.made[int(numbers.None)], held.made[numbers.One]
	for at, value := range append(stars, found...) {
		if value != uint8(numbers.None) {
			test.Fatalf("byte %d of the buffers is %d after a closed run",
				at, value)
		}
	}
}

// the anti-test: buffers the device cannot make close the solver.
func TestBuffersTheDeviceCannotMakeCloseTheSolver(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test, numbers)
	device.EXPECT().Buffer(gomock.Any()).Return(int(numbers.None), nil,
		"could not make a buffer")
	_, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err == nil {
		test.Error("a solver opened with no buffers")
	}
}

// ready is a mock device opened with the kernel's three buffers, each
// of the bytes the layout says: a star's numbers and a pass's for each
// star it has room for, and the field's one number.
func ready(test *testing.T, numbers picked) *mocks.MockDevice {
	device := opened(test, numbers)
	layout := numbers.Layout
	device.EXPECT().Buffer(numbers.One * layout.Star * layout.Bytes).
		DoAndReturn(bytes{numbers}.buffer)
	device.EXPECT().Buffer(numbers.One * layout.Pass * layout.Bytes).
		DoAndReturn(bytes{numbers}.buffer)
	device.EXPECT().Buffer(layout.Bytes).DoAndReturn(bytes{numbers}.buffer)
	return device
}

// the anti-test: no stars ask nothing of the device.
// the solver asks its device for the kernel's layout, and for nothing
// when it has no stars.
func TestNoStarsAskNothingOfTheDevice(test *testing.T) {
	numbers := pickedNumbers(test)
	made, err := gpuorbit.Open(ready(test, numbers), centre(numbers),
		numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	passes, _, err := made.Back(nil, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps)
	if err != nil || len(passes) != int(numbers.None) {
		test.Errorf("no stars gave %v, %v", passes, err)
	}
}

// a run of exactly so many commands' steps is that many commands, each
// dispatched and committed, and not one more.
func TestARunIsSoManyCommands(test *testing.T) {
	numbers := pickedNumbers(test)
	device := ready(test, numbers)
	device.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("").Times(numbers.Commands)
	device.EXPECT().Commit().Return("").Times(numbers.Commands)
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	each := int(research.Value("orbit-steps-a-command"))
	steps := numbers.Commands * each
	passes, _, err := made.Back([]orbit.State{still(numbers)},
		orbit.Units(numbers.Some.Span), steps)
	if err != nil || len(passes) != numbers.One {
		test.Errorf("a run of %d steps gave %v, %v", steps, passes, err)
	}
}

// ending is a run of Back under way, for a test that it ends.
type ending struct {
	made  *gpuorbit.Solver
	stars []orbit.State
	span  measure.Duration
	steps int
	done  chan error
}

// run runs Back and says how it ended.
func (run ending) run() {
	_, _, err := run.made.Back(run.stars, run.span, run.steps)
	run.done <- err
}

// a run of so many steps ends, within the test's patience.
func TestARunEnds(test *testing.T) {
	numbers := pickedNumbers(test)
	device := ready(test, numbers)
	device.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("").AnyTimes()
	device.EXPECT().Commit().Return("").AnyTimes()
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	run := ending{made: made, stars: []orbit.State{still(numbers)},
		span: orbit.Units(numbers.Some.Span), steps: numbers.Some.Steps,
		done: make(chan error)}
	go run.run()
	patience := time.Duration(numbers.Patience * float64(time.Second))
	select {
	case err := <-run.done:
		if err != nil {
			test.Error(err)
		}
	case <-time.After(patience):
		test.Fatal("the run did not end")
	}
}

// the anti-test: a dispatch the device refuses stops the run.
// a dispatch the device refuses stops the run before its commit.
func TestADispatchTheDeviceRefusesStopsTheRun(test *testing.T) {
	numbers := pickedNumbers(test)
	device := ready(test, numbers)
	device.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("no such kernel")
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	_, _, err = made.Back([]orbit.State{still(numbers)},
		orbit.Units(numbers.Some.Span), numbers.Some.Steps)
	if err == nil {
		test.Error("a dispatch the device refused went on to a commit")
	}
}

// the anti-test: a run the device stops is said.
func TestARunTheDeviceStopsIsSaid(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test, numbers)
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(bytes{numbers}.buffer).
		AnyTimes()
	device.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("")
	device.EXPECT().Commit().Return("run: the command was stopped")
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	stars := []orbit.State{orbit.StateOf(numbers.Line.AlongX, numbers.Line.AlongY,
		numbers.Line.AlongZ, numbers.None, numbers.None, numbers.None)}
	if _, _, err := made.Back(stars, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps); err == nil {
		test.Error("a run the device stopped went on")
	}
}

// the kernel agrees with the reference on any batch of stars whose
// orbits the steps resolve: each pass as near, to a thousandth of a
// kiloparsec in each, and each star ending as near the reference's end.
// through an unresolved pass 32-bit rounding grows past any bound, so
// there only the span holds the pass. how long ago a pass was is the
// step the floats found nearest, which where the orbit is flat is not
// the reference's step; so the reference is asked where the star was
// at the kernel's step, which must be as near as its own, and the
// pass's shares are held to the reference's at that step, since the
// star's speed can change between the two steps by more than the
// shares' room.

// agreement is the kernel against the reference, under a test: how
// many stars were drawn and how many compared, the worst compared pass
// as a share of its room, and the worst of their shares as a share of
// theirs.
type agreement struct {
	test       *testing.T
	numbers    picked
	field      gpuorbit.Field
	made       *gpuorbit.Solver
	drawn      int
	compared   int
	worst      float64
	worstShare float64
}

// the property: the kernel agrees with the reference on any batch; and
// a run that compared too few of its stars, or found one near the edge
// of its room, says so rather than passing quietly. the gates are
// written so that a worst that is not a number fails them.
func TestTheKernelAgreesWithTheReference(test *testing.T) {
	numbers := pickedNumbers(test)
	agreeing(test, numbers, centre(numbers))
}

// agreeing runs the agreement property over a field, and holds the run
// to the gates on what it compared and how near the edge it came.
func agreeing(test *testing.T, numbers picked, field gpuorbit.Field) {
	test.Helper()
	holds := &agreement{test: test, numbers: numbers, field: field,
		made: solver(test, field, numbers.Room)}
	rapid.Check(test, holds.onAnyBatch)
	share := float64(holds.compared) / float64(holds.drawn)
	test.Logf("compared %d of %d stars drawn; the worst pass was %.2g of"+
		" its room, and the worst share %.2g of its", holds.compared,
		holds.drawn, holds.worst, holds.worstShare)
	if share < numbers.Agree.Compared {
		test.Errorf("only %d of %d stars were compared; the gate asks %g",
			holds.compared, holds.drawn, numbers.Agree.Compared)
	}
	if !(holds.worst <= numbers.Agree.Worst) {
		test.Errorf("the worst pass was %.2f of its room, past %g",
			holds.worst, numbers.Agree.Worst)
	}
	if !(holds.worstShare <= numbers.Agree.Worst) {
		test.Errorf("the worst share was %.2f of its room, past %g",
			holds.worstShare, numbers.Agree.Worst)
	}
}

// onAnyBatch is the agreement on one drawn batch.
func (holds *agreement) onAnyBatch(draw *rapid.T) {
	stars, span, steps := holds.anyStars(draw)
	passes, ends, err := holds.made.Back(stars, orbit.Units(span), steps)
	if err != nil {
		draw.Fatal(err)
	}
	for index, start := range stars {
		pass, end, err := orbit.Back(holds.field.Reference(), start,
			orbit.Units(span), steps)
		if err != nil {
			draw.Fatal(err)
		}
		holds.onOneStar(draw, index, start, span, steps, passes[index],
			*pass, ends[index], *end)
	}
}

// onOneStar is the agreement on one star of a batch.
func (holds *agreement) onOneStar(draw *rapid.T, index int,
	start orbit.State, span float64, steps int, got, want orbit.Pass,
	end, wantEnd orbit.State) {
	near, ago := got.Numbers()
	wantNear, _ := want.Numbers()
	slack := holds.numbers.Agree.Room
	if near > distance(start)*(whole+slack) || ago > span*(whole+slack) {
		draw.Errorf("star %d passed %g kpc out %g units ago, outside its"+
			" span", index, near, ago)
	}
	dt := span / float64(steps)
	wantMoved, wantPulled := want.Steps()
	resolved := orbittest.Room(wantMoved, wantPulled,
		holds.numbers.Resolved.Factor, holds.numbers.Resolved.Floor)
	holds.drawn++
	if resolved >= holds.numbers.Agree.Resolved {
		return
	}
	holds.compared++
	room := holds.numbers.Agree.Room * (whole + wantNear)
	holds.worst = math.Max(holds.worst, math.Abs(near-wantNear)/room)
	if !orbittest.Within(near, wantNear, room) {
		draw.Errorf("star %d passed at %g kpc, the reference at %g", index,
			near, wantNear)
	}
	then := holds.then(start, ago, dt)
	if !orbittest.Within(distance(then), wantNear, room) {
		draw.Errorf("star %d passed %g units ago, where the reference has"+
			" it %g kpc out, not %g", index, ago, distance(then), wantNear)
	}
	holds.onItsShares(draw, index, got, then, dt)
	if off := apart(end, wantEnd); off > room {
		draw.Errorf("star %d ended %g kpc from the reference", index, off)
	}
	slow := holds.numbers.Agree.Room * (whole + speed(wantEnd))
	if off := speedApart(end, wantEnd); off > slow {
		draw.Errorf("star %d ended %g km/s from the reference", index, off)
	}
}

// whole is the whole of a kiloparsec, under which a pass's room is a
// share of one. Defined.
const whole = 1.0

// onItsShares is the agreement on one star's pass's two shares, held to
// the reference's at the kernel's step, where the reference has the
// star at then.
func (holds *agreement) onItsShares(draw *rapid.T, index int,
	got orbit.Pass, then orbit.State, dt float64) {
	moved, pulled := got.Steps()
	wantMoved, wantPulled := holds.sharesAt(then, dt)
	movedRoom, pulledRoom := holds.shareRooms(then, dt, wantMoved,
		wantPulled)
	movedOff := math.Abs(moved-wantMoved) / movedRoom
	pulledOff := math.Abs(pulled-wantPulled) / pulledRoom
	holds.worstShare = math.Max(holds.worstShare, math.Max(movedOff,
		pulledOff))
	if !orbittest.Within(moved, wantMoved, movedRoom) ||
		!orbittest.Within(pulled, wantPulled, pulledRoom) {
		draw.Errorf("star %d's shares are %g and %g at its step, the"+
			" reference's there %g and %g", index, moved, pulled, wantMoved,
			wantPulled)
	}
}

// sharesAt is the step's shares of a pass at a state in the reference's
// field: the speed times the step over the distance, and the pull there
// times the step squared over the distance, as orbit's Pass carries
// them, for any field.
func (holds *agreement) sharesAt(then orbit.State, dt float64) (moved,
	pulled float64) {
	near := distance(then)
	x, y, z := then.Numbers().At()
	pull := holds.field.Reference().Pull(orbit.PlaceOf(x, y, z)).Along
	return speed(then) * dt / near, orbittest.Size(pull()) * dt * dt / near
}

// shareRooms is the room a pass's two shares have to agree with the
// reference's within, from the rooms the pass's distance and the star's
// speed are held to, a thousandth and a thousandth more for each unit
// of either.
//
// The share in the motion is the speed times the step over the
// distance, so it may be off by the speed's room times the step over
// the distance, which is finite for a star at rest, plus the share
// times the distance's share of room; the share in the pull is a point
// mass's pull over a distance, so by the cube of the distance's share.
func (holds *agreement) shareRooms(then orbit.State, dt, moved,
	pulled float64) (movedRoom, pulledRoom float64) {
	near, fast := distance(then), speed(then)
	slack := holds.numbers.Agree.Room
	byNear := slack * (whole + near) / near
	bySpeed := slack * (whole + fast) * dt / near
	return bySpeed + moved*byNear, pulled * cube * byNear
}

// cube is the most a pass's share in the pull changes with the
// distance, as a power: a pull that falls no faster than the inverse
// square, over the distance, falls no faster than the cube. A point
// mass's pull is the inverse square exactly; the galaxy's falls slower
// everywhere, the disc's nearing the square from above; so for every
// field here the share's room is three of the distance's. Defined by
// the inverse square law.
const cube = 3

// then is where the reference has a star so many units ago, in steps
// of the given length, and how it moves there.
func (holds *agreement) then(start orbit.State, ago, dt float64) orbit.State {
	steps := int(math.Round(ago / dt))
	if steps < holds.numbers.Drawn.FewestSteps {
		return start
	}
	span := orbit.Units(float64(steps) * dt)
	_, end, err := orbit.Back(holds.field.Reference(), start, span, steps)
	if err != nil {
		holds.test.Fatal(err)
	}
	return *end
}

// anyStars is a batch of stars drawn from the picked ranges, with a
// span and a count of steps: no nearer the centre than the nearest,
// at any speed, a star at rest included.
func (holds *agreement) anyStars(draw *rapid.T) ([]orbit.State, float64,
	int) {
	drawn := holds.numbers.Drawn
	count := rapid.IntRange(drawn.FewestStars, holds.numbers.Room).Draw(
		draw, "count")
	stars := make([]orbit.State, count)
	for index := range stars {
		x, y, z := holds.triple(draw, drawn.Farthest, "at")
		if orbittest.Size(x, y, z) < drawn.Nearest {
			x += math.Copysign(drawn.Nearest, x)
		}
		vx, vy, vz := holds.triple(draw, drawn.Fastest, "moving")
		stars[index] = orbit.StateOf(x, y, z, vx, vy, vz)
	}
	span := rapid.Float64Range(drawn.ShortestSpan,
		drawn.LongestSpan).Draw(draw, "span")
	steps := rapid.IntRange(drawn.FewestSteps, drawn.MostSteps).Draw(draw,
		"steps")
	return stars, span, steps
}

// triple is three numbers drawn within a size of zero either way.
func (holds *agreement) triple(draw *rapid.T, size float64,
	label string) (float64, float64, float64) {
	return drawnTriple(draw, size, label)
}

// drawnTriple is three numbers drawn within a size of zero either way.
func drawnTriple(draw *rapid.T, size float64, label string) (float64,
	float64, float64) {
	return rapid.Float64Range(-size, size).Draw(draw, label),
		rapid.Float64Range(-size, size).Draw(draw, label),
		rapid.Float64Range(-size, size).Draw(draw, label)
}
