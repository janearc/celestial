package orbittest_test

import (
	"encoding/json"
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/orbit/orbittest"
)

// picked is the numbers these tests use, from testdata/orbittest.json.
type picked struct {
	Suns         float64
	GM           float64 `json:"gm"`
	Axis         float64
	PeriodInPis  float64 `json:"period in pis"`
	Eccentricity float64
	Triple       struct {
		AlongX float64 `json:"x"`
		AlongY float64 `json:"y"`
		AlongZ float64 `json:"z"`
		Size   float64
	}
	Resolved struct {
		Speed, Near, Dt, Factor, Floor, Want, Moved, Pulled float64
	}
	Energy struct{ Speed, Distance, Want float64 }
	Room   float64
	None   float64
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/orbittest.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// a centre of one sun weighs one sun.
func TestACentreWeighsItsSuns(test *testing.T) {
	numbers := pickedNumbers(test)
	body := orbittest.Centre(numbers.Suns)
	suns := body.Mass().Over(measure.SolarMasses(numbers.Suns))
	if !orbittest.Within(suns, numbers.Suns/numbers.Suns, numbers.Room) {
		test.Errorf("a centre of %g suns weighs %g of them", numbers.Suns,
			suns)
	}
}

// the anti-test: a centre of no suns is no centre, and panics with
// grav's refusal, not with a nil body.
func TestACentreOfNoSunsPanics(test *testing.T) {
	numbers := pickedNumbers(test)
	defer recovered(test)
	orbittest.Centre(numbers.Energy.Want)
	test.Error("a centre of no suns was made")
}

// recovered is a panic that was wanted; it lets the test go on.
func recovered(test *testing.T) {
	why := recover()
	if why == nil {
		test.Error("nothing panicked")
	}
	if _, crashed := why.(runtime.Error); crashed {
		test.Errorf("the panic was a crash, not a refusal: %v", why)
	}
}

// the closed forms agree with each other and with what they define: a
// circular orbit's period is two pi over its angular speed, an
// ellipse's apocentre and pericentre average to its semi-major axis,
// and the apocentre's speed is the circular speed there scaled by the
// ellipse's eccentricity.
func TestTheClosedFormsAgree(test *testing.T) {
	numbers := pickedNumbers(test)
	gm, axis := numbers.GM, numbers.Axis
	period := orbittest.Period(gm, axis)
	if want := numbers.PeriodInPis * math.Pi; !orbittest.Within(period,
		want, numbers.Room) {
		test.Errorf("the period is %g, want %g", period, want)
	}
	far, fast := orbittest.Apocentre(gm, axis, numbers.Eccentricity)
	near, fastest := orbittest.Pericentre(gm, axis, numbers.Eccentricity)
	if !orbittest.Within(orbittest.Half(far+near), axis, numbers.Room) {
		test.Errorf("apocentre %g and pericentre %g average to %g, not %g",
			far, near, orbittest.Half(far+near), axis)
	}
	if !orbittest.Within(fastest*near, fast*far, numbers.Room) {
		test.Errorf("the pericentre's %g at %g and the apocentre's %g at %g"+
			" do not keep angular momentum", fastest, near, fast, far)
	}
	slow := orbittest.Circular(gm, far) * math.Sqrt(near/axis)
	if !orbittest.Within(fast, slow, numbers.Room) {
		test.Errorf("the apocentre's speed is %g, want %g", fast, slow)
	}
}

// a triple's size, a star's energy, the step's shares of a pass and how
// well steps resolve an orbit are what their definitions give.
func TestTheNumbersAreTheirDefinitions(test *testing.T) {
	numbers := pickedNumbers(test)
	triple := numbers.Triple
	size := orbittest.Size(triple.AlongX, triple.AlongY, triple.AlongZ)
	if size != triple.Size {
		test.Errorf("the size of the triple is %g, want %g", size,
			triple.Size)
	}
	resolved := numbers.Resolved
	moved, pulled := orbittest.Shares(numbers.GM, resolved.Speed,
		resolved.Near, resolved.Dt)
	if moved != resolved.Moved || pulled != resolved.Pulled {
		test.Errorf("the shares are %g and %g, want %g and %g", moved,
			pulled, resolved.Moved, resolved.Pulled)
	}
	got := orbittest.Resolved(numbers.GM, resolved.Speed, resolved.Near,
		resolved.Dt, resolved.Factor, resolved.Floor)
	if !orbittest.Within(got, resolved.Want, numbers.Room) {
		test.Errorf("resolved is %g, want %g", got, resolved.Want)
	}
	energy := numbers.Energy
	if got := orbittest.Energy(numbers.GM, energy.Speed,
		energy.Distance); got != energy.Want {
		test.Errorf("the energy is %g, want %g", got, energy.Want)
	}
}

// the anti-test: an ellipse of no eccentricity is a circle, whose
// apocentre and pericentre are both the axis, at the circular speed.
func TestAnEllipseOfNoEccentricityIsACircle(test *testing.T) {
	numbers := pickedNumbers(test)
	gm, axis := numbers.GM, numbers.Axis
	far, fast := orbittest.Apocentre(gm, axis, numbers.None)
	near, fastest := orbittest.Pericentre(gm, axis, numbers.None)
	if far != axis || near != axis {
		test.Errorf("a circle of %g reaches %g and %g", axis, far, near)
	}
	if round := orbittest.Circular(gm, axis); fast != round ||
		fastest != round {
		test.Errorf("a circle of %g moves at %g and %g, not %g", axis, fast,
			fastest, round)
	}
}

// the test and the anti-test: the energies are the energy's two parts,
// whose difference is the energy; and their sum is never less than the
// energy's size, which is nothing where they cancel.
func TestTheEnergiesAreTheEnergysParts(test *testing.T) {
	numbers := pickedNumbers(test)
	energy := numbers.Energy
	moving, pulled := orbittest.Energies(numbers.GM, energy.Speed,
		energy.Distance)
	total := orbittest.Energy(numbers.GM, energy.Speed, energy.Distance)
	if moving-pulled != total {
		test.Errorf("the energies %g and %g differ by %g, not %g", moving,
			pulled, moving-pulled, total)
	}
	if moving+pulled < math.Abs(total) || moving+pulled == numbers.None {
		test.Errorf("the energies %g and %g sum to %g against an energy"+
			" of %g", moving, pulled, moving+pulled, total)
	}
}

// the anti-test: within is not without, and the room's edge is within.
func TestWithinIsNotWithout(test *testing.T) {
	numbers := pickedNumbers(test)
	triple := numbers.Triple
	if orbittest.Within(triple.AlongX, triple.AlongY, numbers.Room) {
		test.Errorf("%g is within %g of %g", triple.AlongX, numbers.Room,
			triple.AlongY)
	}
	edge := triple.AlongY - triple.AlongX
	if !orbittest.Within(triple.AlongX, triple.AlongY, edge) {
		test.Errorf("%g is not within %g of %g", triple.AlongX, edge,
			triple.AlongY)
	}
}
