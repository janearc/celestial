package hole_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/hole"
)

// unitMass is the mass that is the unit of length.
const unitMass = 1 // defined: a hole's mass is its own unit of length

// NewHole makes the hole it was given, mass and spin as said.
func TestNewHoleMakesTheHoleItWasGiven(t *testing.T) {
	holes := pickedNumbers(t).Holes
	for _, from := range []spec{
		holes.Still, holes.Kerr, holes.Heavy, holes.NearlyExtremal,
		holes.Flat,
	} {
		made, err := hole.NewHole(from.Mass, from.Spin)
		if err != nil {
			t.Fatalf("no hole of mass %v and spin %v: %v", from.Mass,
				from.Spin, err)
		}
		if made.Mass() != from.Mass || made.Spin() != from.Spin {
			t.Errorf("a hole of %v and %v answers mass %v and spin %v",
				from.Mass, from.Spin, made.Mass(), made.Spin())
		}
	}
}

// The test: the masses just inside hole's limits, and the limits
// themselves, are holes that hold their own horizon, so nothing
// overflowed or lost its digits; and a spin just under the mass is a
// hole.
func TestNewHoleTakesTheLimits(t *testing.T) {
	numbers := pickedNumbers(t)
	limits := numbers.Limits
	heaviest, lightest := limitsOf(numbers)
	for _, mass := range []float64{limits.Heaviest, limits.Lightest,
		heaviest, lightest} {
		made, err := hole.NewHole(mass, none)
		if err != nil {
			t.Errorf("a mass of %v was refused: %v", mass, err)
			continue
		}
		holdsHorizon(t, *made, limits.Tolerance)
	}
	mass := numbers.Holes.Still.Mass
	if _, err := hole.NewHole(mass, math.Nextafter(mass, none)); err != nil {
		t.Errorf("a spin the next number below the mass was refused: %v",
			err)
	}
}

// The anti-test: the next float64 past each of hole's limits is
// refused.
func TestNewHoleRefusesJustPastTheLimits(t *testing.T) {
	heaviest, lightest := limitsOf(pickedNumbers(t))
	upward := math.Inf(positive)
	refusedAs(t, "just past the heaviest", math.Nextafter(heaviest, upward),
		none)
	refusedAs(t, "just under the lightest", math.Nextafter(lightest, none),
		none)
}

// limitsOf is the heaviest and lightest masses hole holds, worked from
// their definitions in the test data.
func limitsOf(numbers *picked) (heaviest, lightest float64) {
	limits := numbers.Limits
	heaviest = math.Pow(math.MaxFloat64, limits.FourthRoot) /
		limits.HorizonInMasses
	lightest = math.Pow(limits.LeastNormal, limits.FourthRoot)
	return heaviest, lightest
}

// holdsHorizon fails the test unless a still hole's horizon is twice its
// mass, both by Horizon and by Radius at a point on it.
func holdsHorizon(t *testing.T, made hole.Hole, tolerance float64) {
	t.Helper()
	mass := made.Mass()
	twice := mass + mass
	horizon := made.Horizon()
	if !(math.Abs(horizon-twice) <= tolerance*twice) {
		t.Errorf("a mass of %v has a horizon at %v", mass, horizon)
	}
	onIt := made.Radius(hole.NewVec3(horizon, none, none))
	if !(math.Abs(onIt-twice) <= tolerance*twice) {
		t.Errorf("a mass of %v has Radius %v on its horizon", mass, onIt)
	}
}

// The anti-test: NewHole returns nil and an error for a mass and a spin
// that no black hole has, or that hole's arithmetic does not hold.
func TestNewHoleRefusesWhatNoHoleIs(t *testing.T) {
	for _, refused := range pickedNumbers(t).Refused.Cases {
		refusedAs(t, refused.Why, refused.Mass, refused.Spin)
	}
}

// The anti-test: a mass or a spin that is not a number, and an infinite
// mass or spin, are no hole's.
func TestNewHoleRefusesWhatIsNotANumber(t *testing.T) {
	mass := pickedNumbers(t).Holes.Still.Mass
	notANumber, infinite := math.NaN(), math.Inf(positive)
	refusedAs(t, "a mass that is not a number", notANumber, none)
	refusedAs(t, "a spin that is not a number", mass, notANumber)
	refusedAs(t, "an infinite mass", infinite, none)
	refusedAs(t, "an infinite spin", mass, infinite)
}

// refusedAs fails the test unless NewHole returns nil and an error for a
// mass and a spin.
func refusedAs(t *testing.T, why string, mass, spin float64) {
	t.Helper()
	made, err := hole.NewHole(mass, spin)
	if err == nil {
		t.Errorf("%s: mass %v and spin %v were not refused: %v", why, mass,
			spin, err)
	}
	if made != nil {
		t.Errorf("%s: mass %v and spin %v made a hole", why, mass, spin)
	}
}

// The property and its anti-test: any mass and any spin under it make a
// hole, and a spin at the mass or over it does not.
func TestNewHoleIsRightExactlyForSpinsUnderTheMass(t *testing.T) {
	rapid.Check(t, newHoleIsRightExactlyForSpinsUnderTheMass)
}

// newHoleIsRightExactlyForSpinsUnderTheMass is the property that test
// checks.
func newHoleIsRightExactlyForSpinsUnderTheMass(test *rapid.T) {
	draws := pickedNumbers(test)
	mass := rapid.Float64Range(draws.Draws.MassLow,
		draws.Draws.MassHigh).Draw(test, "mass")
	under := rapid.Float64Range(draws.Draws.SpinLow,
		draws.Draws.SpinHigh).Draw(test, "spin as a part of the mass")
	if _, err := hole.NewHole(mass, mass*under); err != nil {
		test.Fatalf("mass %v, spin %v: refused: %v", mass, mass*under, err)
	}
	if made, err := hole.NewHole(mass, mass); err == nil || made != nil {
		test.Fatalf("mass %v, spin at the mass: made a hole", mass)
	}
}

// A table of radii holds for each hole the data names: the values are the
// textbook's for a still hole and the formulas' limits for a nearly
// extremal one.
func TestTheRadiiAreTheTextbooks(t *testing.T) {
	numbers := pickedNumbers(t)
	for name, want := range map[string]struct {
		spec  spec
		radii radiiOf
	}{
		"still":           {numbers.Holes.Still, numbers.Radii.Still},
		"nearly extremal": {numbers.Holes.NearlyExtremal, numbers.Radii.NearlyExtremal},
		"half":            {numbers.Holes.Half, numbers.Radii.Half},
	} {
		black := build(t, want.spec)
		got := map[string]float64{
			"horizon":           black.Horizon(),
			"photon prograde":   black.PhotonOrbit(true),
			"photon retrograde": black.PhotonOrbit(false),
			"isco prograde":     black.ISCO(true),
			"isco retrograde":   black.ISCO(false),
		}
		wanted := map[string]quantity{
			"horizon":           want.radii.Horizon,
			"photon prograde":   want.radii.PhotonPrograde,
			"photon retrograde": want.radii.PhotonRetrograde,
			"isco prograde":     want.radii.ISCOPrograde,
			"isco retrograde":   want.radii.ISCORetrograde,
		}
		for what, quantity := range wanted {
			if quantity.Tolerance == none {
				continue // the data names no value for this hole
			}
			if !within(got[what], quantity.Want, quantity.Tolerance) {
				t.Errorf("%s hole: %s is %v, want %v", name, what,
					got[what], quantity.Want)
			}
		}
	}
}

// A still hole's photon sphere and shadow are the textbooks' radii. That
// they stay a still hole's whatever the spin is held by
// TestSpinDoesNotMoveThePhotonSphereOrShadow.
func TestPhotonSphereAndShadowAreAStillHolesRadii(t *testing.T) {
	numbers := pickedNumbers(t)
	black := build(t, numbers.Holes.Still)
	if got, want := black.PhotonSphere(),
		numbers.Radii.Still.PhotonSphere; !within(got, want.Want,
		want.Tolerance) {
		t.Errorf("photon sphere %v, want %v", got, want.Want)
	}
	if got, want := black.Shadow(), numbers.Radii.Still.Shadow; !within(got,
		want.Want, want.Tolerance) {
		t.Errorf("shadow %v, want %v", got, want.Want)
	}
}

// Between zero and full spin, the prograde innermost stable orbit is where
// kerrgeodesic_gw, a program apart from celestial, documents it.
//
// The tests above hold it only at zero and nearly one, where a wrong in the
// middle of Bardeen, press and teukolsky's formula can still come out right;
// the data's about says what these numbers are and are not.
func TestTheISCOAtMidSpinsIsWhereAnotherProgramPutsIt(t *testing.T) {
	numbers := pickedNumbers(t)
	mid := numbers.IscoMidSpins
	mass := numbers.Holes.Still.Mass
	for _, each := range mid.Radii {
		black := build(t, spec{Mass: mass, Spin: each.Spin})
		if got := black.ISCO(true); !within(got, each.Radius,
			mid.Tolerance) {
			t.Errorf("at spin %v the prograde ISCO is %v, not %v",
				each.Spin, got, each.Radius)
		}
	}
}

// Spin moves the horizon in and splits the orbits: the prograde photon
// orbit and innermost stable orbit lie inside the retrograde ones, and the
// horizon is nearer than a still hole's. This is what a still hole's
// numbers would get wrong.
func TestSpinMovesTheRadii(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	fast := build(t, numbers.Holes.Fast)
	if !(fast.Horizon() < still.Horizon()) {
		t.Errorf("a spinning hole's horizon %v is not inside a still one's %v",
			fast.Horizon(), still.Horizon())
	}
	if !(fast.PhotonOrbit(true) < fast.PhotonOrbit(false)) {
		t.Errorf("photon orbits %v and %v not prograde inside",
			fast.PhotonOrbit(true), fast.PhotonOrbit(false))
	}
	if !(fast.ISCO(true) < fast.ISCO(false)) {
		t.Errorf("innermost stable orbits %v and %v not prograde inside",
			fast.ISCO(true), fast.ISCO(false))
	}
	if still.ISCO(true) != still.ISCO(false) {
		t.Errorf("a still hole's innermost stable orbits part: %v, %v",
			still.ISCO(true), still.ISCO(false))
	}
}

// The equatorial photon orbit is a root of r^2 - 3 M r +- 2 A sqrt(M r),
// plus for prograde, for any hole.
func TestPhotonOrbitSolvesItsEquation(t *testing.T) {
	rapid.Check(t, photonOrbitSolvesItsEquation)
}

// photonOrbitSolvesItsEquation is the property that test checks.
func photonOrbitSolvesItsEquation(test *rapid.T) {
	draws := pickedNumbers(test)
	black := drawnHole(test, draws)
	mass, spin := black.Mass(), black.Spin()
	for _, prograde := range []bool{true, false} {
		sense := float64(retrogradeSign)
		if prograde {
			sense = progradeSign
		}
		radius := black.PhotonOrbit(prograde)
		residue := radius*radius - orbitThree*mass*radius +
			sense*orbitTwo*spin*math.Sqrt(mass*radius)
		allowed := draws.Draws.OrbitTolerance * mass * mass
		if !within(residue, none, allowed) {
			test.Fatalf("mass %v, spin %v, prograde %v: orbit %v leaves %v",
				mass, spin, prograde, radius, residue)
		}
	}
}

// The anti-test: the mass is only the unit of the radii, and moves none of
// them beyond that. A hole of any mass has the radii, over its mass, of the
// hole of one mass with the same spin over mass. A radius that left out the
// mass, or took it twice, agrees with this only at a mass of one, which is
// where the textbook radii are held.
func TestTheRadiiScaleWithTheMass(t *testing.T) {
	rapid.Check(t, theRadiiScaleWithTheMass)
}

// radiusPair is a radius of a hole and the same radius of the hole of one
// mass.
type radiusPair struct{ ofHole, ofUnit float64 }

// theRadiiScaleWithTheMass is the property that test checks.
func theRadiiScaleWithTheMass(test *rapid.T) {
	draws := pickedNumbers(test)
	black := drawnHole(test, draws)
	mass := black.Mass()
	unit, err := hole.NewHole(unitMass, black.Spin()/mass)
	if err != nil {
		test.Fatalf("no hole of unit mass and spin %v: %v",
			black.Spin()/mass, err)
	}
	allowed := draws.Draws.OrbitTolerance
	for what, radius := range map[string]radiusPair{
		"horizon":           {black.Horizon(), unit.Horizon()},
		"photon sphere":     {black.PhotonSphere(), unit.PhotonSphere()},
		"photon prograde":   {black.PhotonOrbit(true), unit.PhotonOrbit(true)},
		"photon retrograde": {black.PhotonOrbit(false), unit.PhotonOrbit(false)},
		"shadow":            {black.Shadow(), unit.Shadow()},
		"isco prograde":     {black.ISCO(true), unit.ISCO(true)},
		"isco retrograde":   {black.ISCO(false), unit.ISCO(false)},
	} {
		if !within(radius.ofHole/mass, radius.ofUnit, allowed) {
			test.Fatalf("mass %v: %s is %v M, and %v M at one mass",
				mass, what, radius.ofHole/mass, radius.ofUnit)
		}
	}
}

// The anti-test: spin does not move the photon sphere or the shadow. They
// are a still hole's, so a hole of the same mass and any spin has the same
// two, to the last digit.
func TestSpinDoesNotMoveThePhotonSphereOrShadow(t *testing.T) {
	numbers := pickedNumbers(t)
	still := build(t, numbers.Holes.Still)
	for _, from := range []spec{
		numbers.Holes.Half, numbers.Holes.Kerr, numbers.Holes.Fast,
		numbers.Holes.NearlyExtremal,
	} {
		turning := build(t, from)
		if turning.PhotonSphere() != still.PhotonSphere() {
			t.Errorf("spin %v moved the photon sphere to %v from %v",
				from.Spin, turning.PhotonSphere(), still.PhotonSphere())
		}
		if turning.Shadow() != still.Shadow() {
			t.Errorf("spin %v moved the shadow to %v from %v",
				from.Spin, turning.Shadow(), still.Shadow())
		}
	}
}
