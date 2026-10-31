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
		made := hole.NewHole(from.Mass, from.Spin)
		if made == nil {
			t.Fatalf("no hole of mass %v and spin %v", from.Mass, from.Spin)
		}
		if made.Mass() != from.Mass || made.Spin() != from.Spin {
			t.Errorf("a hole of %v and %v answers mass %v and spin %v",
				from.Mass, from.Spin, made.Mass(), made.Spin())
		}
	}
}

// The largest mass a float holds is still a hole, as is a spin of nought
// and one just under the mass.
func TestNewHoleTakesTheLimits(t *testing.T) {
	if hole.NewHole(math.MaxFloat64, none) == nil {
		t.Error("the largest mass was refused")
	}
	mass := pickedNumbers(t).Holes.Still.Mass
	if hole.NewHole(mass, math.Nextafter(mass, none)) == nil {
		t.Error("a spin the next number below the mass was refused")
	}
}

// The anti-test: NewHole returns nil, and no hole, for a mass and a spin that no black
// hole has.
func TestNewHoleRefusesWhatNoHoleIs(t *testing.T) {
	for _, refused := range pickedNumbers(t).Refused.Cases {
		if made := hole.NewHole(refused.Mass, refused.Spin); made != nil {
			t.Errorf("%s: mass %v and spin %v made a hole",
				refused.Why, refused.Mass, refused.Spin)
		}
	}
}

// The anti-test: a mass or a spin that is not a number, and an infinite mass, are no
// hole's.
func TestNewHoleRefusesWhatIsNotANumber(t *testing.T) {
	mass := pickedNumbers(t).Holes.Still.Mass
	notANumber := math.NaN()
	for name, made := range map[string]*hole.Hole{
		"a mass that is not a number": hole.NewHole(notANumber, none),
		"a spin that is not a number": hole.NewHole(mass, notANumber),
		"an infinite mass":            hole.NewHole(math.Inf(positive), none),
	} {
		if made != nil {
			t.Errorf("%s made a hole", name)
		}
	}
}

// The property and its anti-test: any mass and any spin under it make a hole, and any spin at the mass or
// over it do not.
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
	if hole.NewHole(mass, mass*under) == nil {
		test.Fatalf("mass %v, spin %v: refused", mass, mass*under)
	}
	if hole.NewHole(mass, mass) != nil {
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

// Between nought and full spin, the prograde innermost stable orbit is
// where kerrgeodesic_gw, a program apart from celestial, documents it.
// The tests above hold it only at nought and nearly one, where a wrong in
// the middle of bardeen, press and teukolsky's formula can still come
// out right; the data's about says what these numbers are and are not.
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
	unit := hole.NewHole(unitMass, black.Spin()/mass)
	if unit == nil {
		test.Fatalf("no hole of unit mass and spin %v", black.Spin()/mass)
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
