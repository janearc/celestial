package orbit_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"

	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/mocks"
	"github.com/janearc/celestial/orbit"
	"github.com/janearc/celestial/orbit/orbittest"
	"github.com/janearc/celestial/si"
)

// picked is the numbers these tests use, from testdata/orbit.json.
type picked struct {
	CentreSuns  float64 `json:"centre suns"`
	LighterSuns float64 `json:"lighter suns"`
	Escaping    struct {
		AlongX     float64 `json:"x"`
		AlongY     float64 `json:"y"`
		AlongZ     float64 `json:"z"`
		VX, VY, VZ float64
		Span       float64
		Steps      int
		Share      float64
	}
	Circle struct {
		Radius     float64
		Steps      int
		Room       float64
		NearRoom   float64 `json:"near room"`
		MovedRoom  float64 `json:"moved room"`
		PulledRoom float64 `json:"pulled room"`
	}
	Inner struct {
		Radius     float64
		Steps      int
		Room       float64
		NearRoom   float64 `json:"near room"`
		MovedRoom  float64 `json:"moved room"`
		PulledRoom float64 `json:"pulled room"`
	} `json:"inner circle"`
	TestMassKg float64 `json:"test mass kg"`
	Ellipse    struct {
		Axis         float64
		Eccentricity float64
		Steps        int
		NearRoom     float64 `json:"near room"`
		AgoSteps     float64 `json:"ago steps"`
		EnergyRoom   float64 `json:"energy room"`
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
	}
	Asked struct {
		Span  float64
		Steps int
		Asks  int
	}
	RunawayPull float64 `json:"runaway pull"`
	Beyond      struct {
		Speed float64
		Span  float64
		Steps int
	}
	Patience float64 `json:"patience seconds"`
	One      float64
	State    struct {
		AlongX     float64 `json:"x"`
		AlongY     float64 `json:"y"`
		AlongZ     float64 `json:"z"`
		VX, VY, VZ float64
		Room       float64
	}
	Pass struct{ Near, Ago, Moved, Pulled float64 }
	Some struct {
		Span  float64
		Steps int
	}
	RefusedSteps []int     `json:"refused steps"`
	RefusedSpans []float64 `json:"refused spans"`
	None         float64
	Drawn        struct {
		Farthest     float64
		Nearest      float64
		Fastest      float64
		LeastSpeed   float64 `json:"least speed"`
		ShortestSpan float64 `json:"shortest span"`
		LongestSpan  float64 `json:"longest span"`
		FewestSteps  int     `json:"fewest steps"`
		MostSteps    int     `json:"most steps"`
	}
	Resolved struct{ Factor, Floor, Limit, Slack float64 }
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
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

// centre is the point mass the tests orbit.
func centre(numbers picked) orbit.Kepler {
	return orbit.NewKepler(orbittest.Centre(numbers.CentreSuns))
}

// free is no pull at all: a mock field that pulls nothing, however
// often it is asked.
func free(test *testing.T, numbers picked) *mocks.MockPotential {
	field := mocks.NewMockPotential(gomock.NewController(test))
	none := orbit.NewPull(numbers.None, numbers.None, numbers.None)
	field.EXPECT().Pull(gomock.Any()).Return(none).AnyTimes()
	return field
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

// energy is a star's energy for each unit of its mass in a point mass's
// field.
func energy(field orbit.Kepler, star orbit.State) float64 {
	return orbittest.Energy(field.GM(), speed(star), distance(star))
}

// energies is the size of that energy's two parts together: what a
// drift in the energy is held against, since the parts can nearly
// cancel while each is large.
func energies(field orbit.Kepler, star orbit.State) float64 {
	moving, pulled := orbittest.Energies(field.GM(), speed(star),
		distance(star))
	return moving + pulled
}

// circling runs a circle of a radius back one period in so many steps
// and holds its end to its start within a room, and its nearest pass to
// its radius within another: the leapfrog's own errors, a few parts in
// a million at four thousand steps, and the nearest exact to rounding.
// The pass's shares are a circle's, each within its own room: in the
// motion, the step's share of a turn; in the pull, that share squared,
// since a circle's pull is its speed squared over its radius. It logs
// what it measured.
func circling(test *testing.T, field orbit.Kepler, radius float64,
	steps int, room, nearRoom, movedRoom, pulledRoom float64,
) {
	test.Helper()
	fast := orbittest.Circular(field.GM(), radius)
	start := orbit.StateOf(radius, nothing, nothing, nothing, fast, nothing)
	period := orbittest.Period(field.GM(), radius)
	pass, end, err := orbit.Back(field, start, orbit.Units(period), steps)
	if err != nil {
		test.Fatal(err)
	}
	if off := apart(start, *end); off > room {
		test.Errorf("after one period the %g kpc circle is %g kpc from its"+
			" start", radius, off)
	}
	if near, _ := pass.Numbers(); !orbittest.Within(near, radius, nearRoom) {
		test.Errorf("a circle of %g kpc came %g kpc from the centre",
			radius, near)
	}
	moved, pulled := pass.Steps()
	stepOfTurn := turn / float64(steps)
	test.Logf("circle of %g kpc: moved off %g, pulled off %g", radius,
		moved-stepOfTurn, pulled-moved*moved)
	if !orbittest.Within(moved, stepOfTurn, movedRoom) ||
		!orbittest.Within(pulled, moved*moved, pulledRoom) {
		test.Errorf("a circle of %g kpc has shares %g and %g, want %g and"+
			" its square", radius, moved, pulled, stepOfTurn)
	}
}

// nothing is no distance and no speed. Defined.
const nothing = 0.0

// turn is a whole turn in radians. Defined.
const turn = 2 * math.Pi

// a circular orbit run back one period comes back where it started, and
// never nearer the centre than its radius.
func TestACircleComesBackToItsStart(test *testing.T) {
	numbers := pickedNumbers(test)
	circling(test, centre(numbers), numbers.Circle.Radius,
		numbers.Circle.Steps, numbers.Circle.Room, numbers.Circle.NearRoom,
		numbers.Circle.MovedRoom, numbers.Circle.PulledRoom)
}

// a circle a parsec from the centre, the search's own distances, comes
// back to its start as the kiloparsec one does, its errors scaled down
// with it.
func TestACircleAParsecOutComesBackToItsStart(test *testing.T) {
	numbers := pickedNumbers(test)
	circling(test, centre(numbers), numbers.Inner.Radius,
		numbers.Inner.Steps, numbers.Inner.Room, numbers.Inner.NearRoom,
		numbers.Inner.MovedRoom, numbers.Inner.PulledRoom)
}

// the test and the anti-test: an ellipse begun at apocentre and run back
// one period passes pericentre half a period ago, at its own distance
// from the centre, and keeps its energy. the pass is sampled, so it is
// as near as the steps allow. its shares are the pericentre's, where
// the star is fastest and nearest, and not the start's: a pass that
// kept the apocentre's would be nine times too small in the motion.
func TestAnEllipsePassesPericentreHalfAPeriodAgo(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	axis, eccentricity := numbers.Ellipse.Axis, numbers.Ellipse.Eccentricity
	far, fast := orbittest.Apocentre(field.GM(), axis, eccentricity)
	start := orbit.StateOf(far, numbers.None, numbers.None, numbers.None,
		fast, numbers.None)
	period := orbittest.Period(field.GM(), axis)
	pass, end, err := orbit.Back(field, start, orbit.Units(period),
		numbers.Ellipse.Steps)
	if err != nil {
		test.Fatal(err)
	}
	near, ago := pass.Numbers()
	want, fastest := orbittest.Pericentre(field.GM(), axis, eccentricity)
	if !orbittest.Within(near, want, numbers.Ellipse.NearRoom) {
		test.Errorf("pericentre at %g kpc, want %g", near, want)
	}
	dt := period / float64(numbers.Ellipse.Steps)
	moved, pulled := pass.Steps()
	wantMoved, wantPulled := orbittest.Shares(field.GM(), fastest, want, dt)
	test.Logf("ellipse: moved off %g of %g, pulled off %g of %g",
		moved-wantMoved, wantMoved, pulled-wantPulled, wantPulled)
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
	before, after := energy(field, start), energy(field, *end)
	if !orbittest.Within(after, before,
		numbers.Ellipse.EnergyRoom*math.Abs(before)) {
		test.Errorf("energy went from %g to %g over a period", before, after)
	}
	if off := apart(start, *end); off > numbers.Ellipse.EndRoom {
		test.Errorf("after one period the star is %g kpc from its start",
			off)
	}
}

// with no pull a star moves in a straight line: run back, its nearest
// pass is where that line comes closest to the centre, its share in the
// motion is the step's length over that distance, and its share in the
// pull is nothing.
func TestAFreeStarCameInAStraightLine(test *testing.T) {
	numbers := pickedNumbers(test)
	line := numbers.Line
	start := orbit.StateOf(line.AlongX, line.AlongY, line.AlongZ, line.VX, line.VY,
		line.VZ)
	pass, end, err := orbit.Back(free(test, numbers), start,
		orbit.Units(line.Span), line.Steps)
	if err != nil {
		test.Fatal(err)
	}
	near, ago := pass.Numbers()
	if !orbittest.Within(near, line.Near, line.Room) ||
		!orbittest.Within(ago, line.Ago, line.Room) {
		test.Errorf("the line passed %g kpc from the centre %g units ago,"+
			" want %g and %g", near, ago, line.Near, line.Ago)
	}
	freeShares(test, numbers, *pass, speed(start), line.Near, line.Room)
	then := orbit.StateOf(line.AlongX, line.EndY, line.AlongZ, line.VX, line.VY,
		line.VZ)
	if off := apart(*end, then); off > line.Room {
		test.Errorf("at the start of the span the star was %g kpc off", off)
	}
}

// freeShares holds a free star's pass to its shares: in the motion, the
// step's length at the star's speed over the pass's distance, within a
// room; in the pull, nothing at all.
func freeShares(test *testing.T, numbers picked, pass orbit.Pass,
	fast, near, room float64,
) {
	test.Helper()
	dt := numbers.Line.Span / float64(numbers.Line.Steps)
	moved, pulled := pass.Steps()
	wantMoved, wantPulled := orbittest.Shares(numbers.None, fast, near, dt)
	if !orbittest.Within(moved, wantMoved, room) || pulled != wantPulled {
		test.Errorf("a free star's shares are %g and %g, want %g and %g",
			moved, pulled, wantMoved, wantPulled)
	}
}

// still is a star at the state's place, not moving.
func still(numbers picked) orbit.State {
	return orbit.StateOf(numbers.State.AlongX, numbers.State.AlongY,
		numbers.State.AlongZ, numbers.None, numbers.None, numbers.None)
}

// the field is asked once at the start and once a step; and a star
// that does not move passed nearest now, not at any later step as near,
// with no share in the motion and none in the pull.
func TestTheFieldIsAskedOnceAStep(test *testing.T) {
	numbers := pickedNumbers(test)
	field := mocks.NewMockPotential(gomock.NewController(test))
	none := orbit.NewPull(numbers.None, numbers.None, numbers.None)
	field.EXPECT().Pull(gomock.Any()).Return(none).Times(numbers.Asked.Asks)
	star := still(numbers)
	pass, _, err := orbit.Back(field, star, orbit.Units(numbers.Asked.Span),
		numbers.Asked.Steps)
	if err != nil {
		test.Fatal(err)
	}
	if near, ago := pass.Numbers(); near != distance(star) ||
		ago != numbers.None {
		test.Errorf("a still star passed %g kpc out %g units ago", near, ago)
	}
	if moved, pulled := pass.Steps(); moved != numbers.None ||
		pulled != numbers.None {
		test.Errorf("a still star's shares are %g and %g", moved, pulled)
	}
}

// the anti-test: a pass through the centre itself has no finite shares,
// since no step resolves it: a free star at the centre, moving, passed
// now at no distance, and its shares are not numbers to hold anything
// to.
func TestAPassThroughTheCentreHasNoFiniteShares(test *testing.T) {
	numbers := pickedNumbers(test)
	line := numbers.Line
	start := orbit.StateOf(numbers.None, numbers.None, numbers.None,
		line.VX, line.VY, line.VZ)
	pass, _, err := orbit.Back(free(test, numbers), start,
		orbit.Units(line.Span), line.Steps)
	if err != nil {
		test.Fatal(err)
	}
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

// the unit of time is a kiloparsec per kilometre per second: so many
// seconds as a kiloparsec has metres over a kilometre's.
func TestTheUnitOfTimeIsAKiloparsecPerKilometrePerSecond(test *testing.T) {
	numbers := pickedNumbers(test)
	kiloparsec := orbit.Kiloparsecs(numbers.One).InMetres()
	want := kiloparsec / kilometresInMetres
	got := orbit.Units(numbers.One).InSeconds()
	if !orbittest.Within(got, want, numbers.State.Room*want) {
		test.Errorf("the unit of time is %g s, want %g", got, want)
	}
}

// the anti-test: a star beyond a float runs away.
// a star faster than a float can add to itself runs away: its place
// is infinite, and infinite either way is no place, however finite its
// motion stays.
func TestAStarBeyondAFloatRunsAway(test *testing.T) {
	numbers := pickedNumbers(test)
	star := orbit.StateOf(numbers.State.AlongX, numbers.State.AlongY,
		numbers.State.AlongZ, numbers.Beyond.Speed, numbers.None,
		numbers.None)
	_, _, err := orbit.Back(free(test, numbers), star,
		orbit.Units(numbers.Beyond.Span), numbers.Beyond.Steps)
	if err == nil {
		test.Error("a star beyond a float has an orbit")
	}
}

// the anti-test: a pull that is not a number runs the orbit away.
// a pull that is not a number along one axis alone runs the orbit
// away: one axis is enough.
func TestAPullThatIsNotANumberRunsAway(test *testing.T) {
	numbers := pickedNumbers(test)
	field := mocks.NewMockPotential(gomock.NewController(test))
	unsaid := orbit.NewPull(math.NaN(), numbers.None, numbers.None)
	field.EXPECT().Pull(gomock.Any()).Return(unsaid).AnyTimes()
	_, _, err := orbit.Back(field, still(numbers),
		orbit.Units(numbers.Some.Span), numbers.Some.Steps)
	if err == nil {
		test.Error("a star pulled by no number has an orbit")
	}
}

// ending is a run of Back under way, for a test that it ends.
type ending struct {
	field orbit.Potential
	star  orbit.State
	span  measure.Duration
	steps int
	done  chan error
}

// run runs Back and says how it ended.
func (run ending) run() {
	_, _, err := orbit.Back(run.field, run.star, run.span, run.steps)
	run.done <- err
}

// a run of so many steps ends, within the test's patience.
func TestARunEnds(test *testing.T) {
	numbers := pickedNumbers(test)
	run := ending{
		field: free(test, numbers), star: still(numbers),
		span: orbit.Units(numbers.Some.Span), steps: numbers.Some.Steps,
		done: make(chan error),
	}
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

// a kepler field's gm is the body's, in the package's units: newton's
// constant times a mass given in kilograms, over a kiloparsec from
// length's parsec and a kilometre squared, a route that shares with the
// package neither the sun's gm nor its kiloparsec.
func TestKeplerGMIsTheBodysInKiloparsecs(test *testing.T) {
	numbers := pickedNumbers(test)
	body, err := grav.Point(measure.Kilograms(numbers.TestMassKg))
	if err != nil {
		test.Fatal(err)
	}
	kiloparsec := si.Kilo(length.Parsec)
	want := si.Gravitation() * numbers.TestMassKg /
		(kiloparsec * kilometresInMetres * kilometresInMetres)
	got := orbit.NewKepler(*body).GM()
	if !orbittest.Within(got, want, numbers.State.Room*want) {
		test.Errorf("gm is %g, want %g", got, want)
	}
}

// kilometresInMetres is a kilometre. Defined by the SI's prefix.
const kilometresInMetres = 1000

// a state gives back the place and motion it was made from.
func TestAStateIsWhatItWasMadeFrom(test *testing.T) {
	numbers := pickedNumbers(test)
	given := numbers.State
	start := orbit.StateOf(given.AlongX, given.AlongY, given.AlongZ, given.VX, given.VY,
		given.VZ)
	place := measure.At(orbit.Kiloparsecs(given.AlongX),
		orbit.Kiloparsecs(given.AlongY), orbit.Kiloparsecs(given.AlongZ))
	moving, err := measure.KilometresPerSecond(given.VX, given.VY, given.VZ)
	if err != nil {
		test.Fatal(err)
	}
	if again := orbit.NewState(place, *moving); apart(start, again) >
		given.Room || speed(again) != speed(start) {
		test.Error("a state from a place and a motion is not the state")
	}
	off := start.Place().Minus(place).Length().InMetres()
	if off > orbit.Kiloparsecs(given.Room).InMetres() {
		test.Errorf("the place is %g m off", off)
	}
	motion, err := start.Motion()
	if err != nil {
		test.Fatal(err)
	}
	if got := motion.AlongZ().InKilometresPerSecond(); !orbittest.Within(
		got, given.VZ, given.Room) {
		test.Errorf("the motion along z is %g km/s, want %g", got, given.VZ)
	}
}

// a pass gives back its numbers and its shares, and a place its
// numbers; the place a potential is asked at is a coordinate.
func TestAPassAndAPlaceAreWhatTheyWereMadeFrom(test *testing.T) {
	numbers := pickedNumbers(test)
	given := numbers.State
	pass := orbit.NewPass(numbers.Pass.Near, numbers.Pass.Ago,
		numbers.Pass.Moved, numbers.Pass.Pulled)
	near := pass.Near().Over(orbit.Kiloparsecs(numbers.Pass.Near))
	ago := pass.Ago().Over(orbit.Units(numbers.Pass.Ago))
	if !orbittest.Within(near, whole, given.Room) ||
		!orbittest.Within(ago, whole, given.Room) {
		test.Errorf("the pass is %g of its near and %g of its ago", near,
			ago)
	}
	if moved, pulled := pass.Steps(); moved != numbers.Pass.Moved ||
		pulled != numbers.Pass.Pulled {
		test.Errorf("the pass's shares are %g and %g, want %g and %g", moved,
			pulled, numbers.Pass.Moved, numbers.Pass.Pulled)
	}
	if x, y, z := orbit.PlaceOf(given.AlongX, given.AlongY,
		given.AlongZ).Numbers(); x != given.AlongX || y != given.AlongY ||
		z != given.AlongZ {
		test.Errorf("a place of %g %g %g reads back as %g %g %g",
			given.AlongX, given.AlongY, given.AlongZ, x, y, z)
	}
	empty := (orbit.Place{}).Coord().Minus(measure.Origin()).Length()
	if off := empty.InMetres(); off != numbers.None {
		test.Errorf("the empty place is %g m from the centre", off)
	}
}

// whole is a ratio of a number to itself. Defined.
const whole = 1.0

// the anti-tests: what Back refuses.

// the anti-test: no steps are refused.
func TestNoStepsAreRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	star := orbit.StateOf(numbers.State.AlongX, numbers.State.AlongY, numbers.State.AlongZ,
		numbers.None, numbers.None, numbers.None)
	for _, steps := range numbers.RefusedSteps {
		_, _, err := orbit.Back(free(test, numbers), star,
			orbit.Units(numbers.Some.Span), steps)
		if err == nil {
			test.Errorf("%d steps were taken", steps)
		}
	}
	if _, err := orbit.Step(orbit.Units(numbers.Some.Span),
		numbers.RefusedSteps[int(numbers.None)]); err == nil {
		test.Error("a step of no steps")
	}
}

// the anti-test: a span that is not positive is refused.
func TestASpanThatIsNotPositiveIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	star := orbit.StateOf(numbers.State.AlongX, numbers.State.AlongY, numbers.State.AlongZ,
		numbers.None, numbers.None, numbers.None)
	spans := append(numbers.RefusedSpans, math.NaN())
	for _, span := range spans {
		_, _, err := orbit.Back(free(test, numbers), star,
			orbit.Units(span), numbers.Some.Steps)
		if err == nil {
			test.Errorf("a span of %g units was run", span)
		}
	}
}

// the anti-test: the pull at the centre itself is not a number, along
// any axis, so no orbit can be stepped from there.
func TestThePullAtTheCentreIsNotANumber(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	pull := field.Pull(orbit.Place{})
	x, y, z := pull.Along()
	if !math.IsNaN(x) || !math.IsNaN(y) || !math.IsNaN(z) {
		test.Errorf("the pull at the centre is (%g, %g, %g)", x, y, z)
	}
}

// the anti-test: a star at the centre runs away.
func TestAStarAtTheCentreRunsAway(test *testing.T) {
	numbers := pickedNumbers(test)
	star := orbit.StateOf(numbers.None, numbers.None, numbers.None,
		numbers.None, numbers.None, numbers.None)
	_, _, err := orbit.Back(centre(numbers), star,
		orbit.Units(numbers.Some.Span), numbers.Some.Steps)
	if err == nil {
		test.Error("a star at the centre has an orbit")
	}
}

// the anti-test: a star pulled past light has no motion.
func TestAStarPulledPastLightHasNoMotion(test *testing.T) {
	numbers := pickedNumbers(test)
	field := mocks.NewMockPotential(gomock.NewController(test))
	hard := orbit.NewPull(numbers.RunawayPull, numbers.None, numbers.None)
	field.EXPECT().Pull(gomock.Any()).Return(hard).AnyTimes()
	star := orbit.StateOf(numbers.State.AlongX, numbers.State.AlongY, numbers.State.AlongZ,
		numbers.None, numbers.None, numbers.None)
	_, end, err := orbit.Back(field, star, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps)
	if err != nil {
		test.Fatal(err)
	}
	if _, err := end.Motion(); err == nil {
		test.Error("a star past light's speed has a motion")
	}
}

// the anti-test: a heavier centre pulls harder at the same place and
// a lighter one less, and the pull is the inverse square toward the
// centre along each axis, not only in size.
func TestAHeavierCentrePullsHarder(test *testing.T) {
	numbers := pickedNumbers(test)
	heavy := centre(numbers)
	light := orbit.NewKepler(orbittest.Centre(numbers.LighterSuns))
	given := numbers.State
	where := orbit.PlaceOf(given.AlongX, given.AlongY, given.AlongZ)
	distance := orbittest.Size(given.AlongX, given.AlongY, given.AlongZ)
	cubed := distance * distance * distance
	x, y, z := heavy.Pull(where).Along()
	if !orbittest.Within(x, -heavy.GM()*given.AlongX/cubed, given.Room) ||
		!orbittest.Within(y, -heavy.GM()*given.AlongY/cubed, given.Room) ||
		!orbittest.Within(z, -heavy.GM()*given.AlongZ/cubed, given.Room) {
		test.Errorf("the pull at (%g, %g, %g) is (%g, %g, %g)",
			given.AlongX, given.AlongY, given.AlongZ, x, y, z)
	}
	harder := orbittest.Size(x, y, z)
	softer := orbittest.Size(light.Pull(where).Along())
	if !(harder > softer) || heavy.GM() <= light.GM() {
		test.Errorf("a centre of %g suns pulls %g, one of %g suns %g",
			numbers.CentreSuns, harder, numbers.LighterSuns, softer)
	}
}

// the anti-test: a free star moving away from the centre passed
// nearest now: run back it only recedes, so its pass is where it is,
// with the start's shares.
func TestAFreeStarMovingAwayPassedNow(test *testing.T) {
	numbers := pickedNumbers(test)
	line := numbers.Line
	start := orbit.StateOf(line.AlongX, line.AlongY, line.AlongZ, line.VX,
		-line.VY, line.VZ)
	pass, _, err := orbit.Back(free(test, numbers), start,
		orbit.Units(line.Span), line.Steps)
	if err != nil {
		test.Fatal(err)
	}
	near, ago := pass.Numbers()
	if !orbittest.Within(near, distance(start), line.Room) ||
		ago != numbers.None {
		test.Errorf("a star moving away passed %g kpc out %g units ago,"+
			" want %g and now", near, ago, distance(start))
	}
	freeShares(test, numbers, *pass, speed(start), distance(start),
		line.Room)
}

// the anti-test: an ellipse is not a circle; it comes nearer.
func TestAnEllipseIsNotACircle(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	far, fast := orbittest.Apocentre(field.GM(), numbers.Ellipse.Axis,
		numbers.Ellipse.Eccentricity)
	start := orbit.StateOf(far, numbers.None, numbers.None, numbers.None,
		fast, numbers.None)
	pass, _, err := orbit.Back(field, start, orbit.Units(numbers.Some.Span),
		numbers.Some.Steps)
	if err != nil {
		test.Fatal(err)
	}
	if near, _ := pass.Numbers(); near >= far {
		test.Errorf("an ellipse begun at apocentre came no nearer than %g",
			near)
	}
}

// the properties: over any span, a star's nearest pass is no farther
// than where it is now and no earlier than the span; in a point mass's
// field it keeps its energy where the steps resolve the orbit; and with
// no pull it comes no nearer than the line it moved along does.

// properties is what any star's orbit must satisfy, under a test.
type properties struct {
	test    *testing.T
	numbers picked
	field   orbit.Kepler
}

// the anti-test: a star whose energies nearly cancel keeps its energy
// too, held against the parts and not against their near-nothing
// difference. firang's run found this star on 2026-11-05: a drift of a
// hundred-thousandth of either part was three hundredths of the sum.
func TestAStarNearEscapeKeepsItsEnergy(test *testing.T) {
	numbers := pickedNumbers(test)
	field := centre(numbers)
	found := numbers.Escaping
	start := orbit.StateOf(found.AlongX, found.AlongY, found.AlongZ,
		found.VX, found.VY, found.VZ)
	_, end, err := orbit.Back(field, start, orbit.Units(found.Span),
		found.Steps)
	if err != nil {
		test.Fatal(err)
	}
	before, after := energy(field, start), energy(field, *end)
	parts := energies(field, start)
	if math.Abs(before) > found.Share*parts {
		test.Fatalf("the energy %g is not near nothing against %g, so"+
			" the star shows nothing", before, parts)
	}
	room := orbittest.Resolved(field.GM(), speed(start), distance(start),
		found.Span/float64(found.Steps), numbers.Resolved.Factor,
		numbers.Resolved.Floor)
	if !orbittest.Within(after, before, room*parts) {
		test.Errorf("energy went from %g to %g, past %g of %g", before,
			after, room, parts)
	}
}

// the property: any pass is within the span, its share in the pull
// follows from its distance, and energy is kept.
func TestAnyPassIsWithinTheSpan(test *testing.T) {
	numbers := pickedNumbers(test)
	holds := properties{test, numbers, centre(numbers)}
	rapid.Check(test, holds.withinTheSpan)
}

// the property: a free star comes no nearer than its line.
func TestAFreeStarComesNoNearerThanItsLine(test *testing.T) {
	numbers := pickedNumbers(test)
	holds := properties{test, numbers, centre(numbers)}
	rapid.Check(test, holds.noNearerThanItsLine)
}

// withinTheSpan is the pass within the span, its share in the pull what
// a point mass's pull at its distance gives, to rounding, and the energy
// kept where the steps resolve the orbit.
func (holds properties) withinTheSpan(draw *rapid.T) {
	start, span, steps := holds.anyStar(draw)
	pass, end, err := orbit.Back(holds.field, start, orbit.Units(span),
		steps)
	if err != nil {
		draw.Fatal(err)
	}
	near, ago := pass.Numbers()
	slack := holds.numbers.Resolved.Slack
	if now := distance(start); near > now+slack {
		draw.Errorf("the pass at %g kpc is farther than now, %g", near, now)
	}
	if ago < holds.numbers.None || ago > span+slack {
		draw.Errorf("the pass %g units ago is outside %g", ago, span)
	}
	dt := span / float64(steps)
	_, pulled := pass.Steps()
	_, wantPulled := orbittest.Shares(holds.field.GM(), speed(start), near,
		dt)
	if !orbittest.Within(pulled, wantPulled, slack*wantPulled) {
		draw.Errorf("the pull's share at %g kpc is %g, want %g", near,
			pulled, wantPulled)
	}
	room := orbittest.Resolved(holds.field.GM(), speed(start), near, dt,
		holds.numbers.Resolved.Factor, holds.numbers.Resolved.Floor)
	if room >= holds.numbers.Resolved.Limit {
		return
	}
	before, after := energy(holds.field, start), energy(holds.field, *end)
	if !orbittest.Within(after, before, room*energies(holds.field, start)) {
		draw.Errorf("energy went from %g to %g, past %g of its parts'"+
			" %g", before, after, room, energies(holds.field, start))
	}
}

// noNearerThanItsLine is a free star's pass against the line it moved
// along: back in time the star is at its place less its motion times
// the time, nearest the centre where the line is, held within the span;
// the sampled pass is no nearer than that and no farther than a step.
// its shares are the step's length at the star's speed over the pass's
// distance, to rounding, and nothing in the pull.
func (holds properties) noNearerThanItsLine(draw *rapid.T) {
	start, span, steps := holds.anyStar(draw)
	pass, _, err := orbit.Back(free(holds.test, holds.numbers), start,
		orbit.Units(span), steps)
	if err != nil {
		draw.Fatal(err)
	}
	x, y, z := start.Numbers().At()
	vx, vy, vz := start.Numbers().Moving()
	fast := speed(start)
	when := (x*vx + y*vy + z*vz) / (fast * fast)
	when = math.Max(holds.numbers.None, math.Min(span, when))
	line := orbittest.Size(x-vx*when, y-vy*when, z-vz*when)
	near, _ := pass.Numbers()
	slack := holds.numbers.Resolved.Slack
	dt := span / float64(steps)
	if near < line-slack || near > line+fast*dt+slack {
		draw.Errorf("the pass at %g kpc is not within a step of the"+
			" line's %g", near, line)
	}
	moved, pulled := pass.Steps()
	wantMoved, wantPulled := orbittest.Shares(holds.numbers.None, fast,
		near, dt)
	if !orbittest.Within(moved, wantMoved, slack*wantMoved) ||
		pulled != wantPulled {
		draw.Errorf("a free star's shares are %g and %g, want %g and %g",
			moved, pulled, wantMoved, wantPulled)
	}
}

// anyStar is a star drawn from the picked ranges: no nearer the centre
// than the nearest, moving at least the least speed, with a span and a
// count of steps.
func (holds properties) anyStar(draw *rapid.T) (orbit.State, float64,
	int,
) {
	drawn := holds.numbers.Drawn
	x, y, z := holds.triple(draw, drawn.Farthest, "at")
	if orbittest.Size(x, y, z) < drawn.Nearest {
		x += math.Copysign(drawn.Nearest, x)
	}
	vx, vy, vz := holds.triple(draw, drawn.Fastest, "moving")
	if orbittest.Size(vx, vy, vz) == holds.numbers.None {
		vx = drawn.LeastSpeed
	}
	span := rapid.Float64Range(drawn.ShortestSpan,
		drawn.LongestSpan).Draw(draw, "span")
	steps := rapid.IntRange(drawn.FewestSteps, drawn.MostSteps).Draw(draw,
		"steps")
	return orbit.StateOf(x, y, z, vx, vy, vz), span, steps
}

// triple is three numbers drawn within a size of zero either way.
func (holds properties) triple(draw *rapid.T, size float64,
	label string,
) (float64, float64, float64) {
	return rapid.Float64Range(-size, size).Draw(draw, label),
		rapid.Float64Range(-size, size).Draw(draw, label),
		rapid.Float64Range(-size, size).Draw(draw, label)
}
