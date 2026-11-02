//ninja:vow dojo the radius a ray cannot come back from is held to the ratio its doc gives, through noReturn, which hole does not export
package hole

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// spot is a point or a direction in space as the data gives it.
type spot struct {
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// space is the spot as a Vec3.
func (given spot) space() Vec3 {
	return Vec3{x: given.AlongX, y: given.AlongY, z: given.AlongZ}
}

// event is a point or a vector in spacetime as the data gives it.
type event struct {
	Time   float64 `json:"time"`
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// spacetime is the event as a Vec4.
func (given event) spacetime() Vec4 {
	return Vec4{time: given.Time, x: given.AlongX, y: given.AlongY,
		z: given.AlongZ}
}

// massAndSpin is a hole as the data gives it.
type massAndSpin struct{ Mass, Spin float64 }

// noReturnNumbers are the rays noReturn is held to.
type noReturnNumbers struct {
	Holes []struct {
		Mass, Spin    float64
		Radii, Inside []float64
	}
	Position   spot
	Energy     float64
	MomentumX  float64 `json:"momentum x"`
	MomentumZ  float64 `json:"momentum z"`
	Relative   float64
	NoEnergy   []event `json:"no energy"`
	WithEnergy event   `json:"with energy"`
}

// unexportedNumbers are testdata/hole.json's numbers for the tests in
// package hole.
type unexportedNumbers struct {
	NoReturn noReturnNumbers `json:"no return"`
	Inward   struct {
		Hole         massAndSpin
		Points       []spot
		GradientStep float64 `json:"gradient step"`
		Off          float64
		Round        spot
		Along        event
	}
	Edge struct {
		Still         massAndSpin
		TowardTheHole spot `json:"toward the hole"`
		Energy        float64
	}
	Cross struct {
		First, Second, Product spot
		Dot                    float64
		LengthsSquared         float64 `json:"lengths squared"`
		Tolerance              float64
	}
	Choices struct{ Broken, Empty string }
}

// innerNumbers reads the numbers, or the test stops.
func innerNumbers(test *testing.T) unexportedNumbers {
	test.Helper()
	raw, err := os.ReadFile("testdata/hole.json")
	if err != nil {
		test.Fatal(err)
	}
	var file struct {
		Unexported unexportedNumbers `json:"unexported"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		test.Fatal(err)
	}
	return file.Unexported
}

// madeHole is the hole of a mass and a spin, or the test stops.
func madeHole(test *testing.T, mass, spin float64) Hole {
	test.Helper()
	made, err := NewHole(mass, spin)
	if err != nil {
		test.Fatal(err)
	}
	return *made
}

// The numbers of the ratio of Lz to E of a spherical photon orbit. Each is
// defined by the formula noReturn's doc gives,
// -(r^3 - 3 M r^2 + A^2 r + A^2 M) / (A (r - M)).
const (
	// ratioThree is the 3 of 3 M r^2. Defined by that formula.
	ratioThree = 3
)

// photonRatio is Lz over E of the spherical photon orbit at a radius, by
// the formula noReturn's doc gives.
func photonRatio(black Hole, radius float64) float64 {
	mass, spinSquared := black.mass, black.spin*black.spin
	cubic := radius*radius*radius - ratioThree*mass*radius*radius +
		spinSquared*radius + spinSquared*mass
	return -cubic / (black.spin * (radius - mass))
}

// ratioRay is a ray at the data's point with the data's energy and the
// ratio of Lz to E of the orbit at a radius: its y momentum is worked so
// that x p_y - y p_x is the ratio times the energy.
func ratioRay(black Hole, given noReturnNumbers, radius float64) (Vec4, Vec4) {
	at := given.Position
	lz := photonRatio(black, radius) * given.Energy
	momentumY := (lz + at.AlongY*given.MomentumX) / at.AlongX
	return Vec4{x: at.AlongX, y: at.AlongY, z: at.AlongZ}, Vec4{
		time: -given.Energy, x: given.MomentumX, y: momentumY,
		z: given.MomentumZ,
	}
}

// A ray inward of the radius noReturn gives it has no turn ahead: that
// radius is the spherical photon orbit with the ray's ratio of Lz to E.
// Made from a radius by that ratio, a ray is given the radius back, round
// three spins and two masses, with an energy that is not one.
func TestNoReturnIsTheOrbitWithTheRaysRatio(t *testing.T) {
	given := innerNumbers(t).NoReturn
	for _, from := range given.Holes {
		black := madeHole(t, from.Mass, from.Spin)
		for _, inMasses := range from.Radii {
			radius := inMasses * from.Mass
			got := black.noReturn(ratioRay(black, given, radius))
			if !(math.Abs(got-radius) <= given.Relative*radius) {
				t.Errorf("spin %v, mass %v: the ray of the orbit at %v "+
					"is given %v", from.Spin, from.Mass, radius, got)
			}
		}
	}
}

// The anti-test: a ratio no orbit has, from inside the prograde orbit, is
// not given its radius from the cubic: the prograde orbit, inside all the
// others, bounds the answer below, and is the answer.
func TestNoReturnIsNeverInsideTheProgradeOrbit(t *testing.T) {
	given := innerNumbers(t).NoReturn
	for _, from := range given.Holes {
		black := madeHole(t, from.Mass, from.Spin)
		innermost := black.PhotonOrbit(true)
		for _, inMasses := range from.Inside {
			radius := inMasses * from.Mass
			got := black.noReturn(ratioRay(black, given, radius))
			if got != innermost {
				t.Errorf("spin %v, mass %v: the ratio from %v is given %v, "+
					"not the prograde orbit %v", from.Spin, from.Mass,
					radius, got, innermost)
			}
		}
	}
}

// A ray with no energy has no ratio, and is given the innermost photon
// orbit, whichever way it turns about the axis or if it does not: Lz over
// minus nought is minus infinity, infinity, or not a number.
func TestNoReturnOfARayWithNoEnergyIsTheInnermost(t *testing.T) {
	given := innerNumbers(t).NoReturn
	for _, from := range given.Holes {
		black := madeHole(t, from.Mass, from.Spin)
		point := given.Position.space()
		for _, momentum := range given.NoEnergy {
			got := black.noReturn(pointAt(point), momentum.spacetime())
			if want := black.PhotonOrbit(true); got != want {
				t.Errorf("spin %v: a ray with no energy and momentum %v is "+
					"given %v, not %v", from.Spin, momentum, got, want)
			}
		}
	}
}

// The anti-test: the same ray with energy has a ratio, and a radius of its
// own outside the innermost orbit.
func TestNoReturnOfARayWithEnergyIsItsOwn(t *testing.T) {
	given := innerNumbers(t).NoReturn
	for _, from := range given.Holes {
		black := madeHole(t, from.Mass, from.Spin)
		point := pointAt(given.Position.space())
		got := black.noReturn(point, given.WithEnergy.spacetime())
		if !(got > black.PhotonOrbit(true)) {
			t.Errorf("spin %v: a ray with energy is given %v, the "+
				"innermost orbit", from.Spin, got)
		}
	}
}
