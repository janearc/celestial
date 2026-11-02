package hole_test

import (
	"encoding/json"
	"math"
	"os"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/hole"
)

// spec is a hole's mass and spin as the data gives them.
type spec struct{ Mass, Spin float64 }

// place is a point in space as the data gives it.
type place struct {
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// quantity is a number a test wants, and how near it holds it.
type quantity struct{ Want, Tolerance float64 }

// radiiOf are the radii wanted of one hole.
type radiiOf struct {
	Horizon          quantity
	PhotonSphere     quantity `json:"photon sphere"`
	PhotonPrograde   quantity `json:"photon prograde"`
	PhotonRetrograde quantity `json:"photon retrograde"`
	Shadow           quantity
	ISCOPrograde     quantity `json:"isco prograde"`
	ISCORetrograde   quantity `json:"isco retrograde"`
}

// picked is the numbers these tests pick, from testdata/hole.json.
type picked struct {
	Holes struct {
		Still          spec
		Half           spec
		Kerr           spec
		Spinning       spec
		Fast           spec
		NearlyExtremal spec `json:"nearly extremal"`
		Heavy          spec
		Flat           spec
	}
	Refused struct {
		Cases []struct {
			Why        string
			Mass, Spin float64
		}
	}
	Limits struct {
		Heaviest        float64
		Lightest        float64
		Tolerance       float64
		FourthRoot      float64 `json:"fourth root"`
		HorizonInMasses float64 `json:"horizon in masses"`
		LeastNormal     float64 `json:"least normal"`
	}
	Radii struct {
		Still          radiiOf
		Half           radiiOf
		NearlyExtremal radiiOf `json:"nearly extremal"`
	}
	Places struct {
		Skew, Slant, Tilt, Equator, Pole, Far, Origin, Near, Eye place
	}
	Metric struct {
		IdentityTolerance float64 `json:"identity tolerance"`
		FarTolerance      float64 `json:"far tolerance"`
		RoundTrip         float64 `json:"round trip tolerance"`
	}
	Frame struct {
		Tolerance         float64
		PropertyTolerance float64 `json:"property tolerance"`
		DopplerTolerance  float64 `json:"doppler tolerance"`
		InsideHorizon     place   `json:"inside horizon"`
		Ergosphere        place
		Emitter           place
		Aim               place
		Outward           place
		CircleRadius      float64 `json:"circle radius"`
		InsideShare       float64 `json:"inside share"`
		OutsideShare      float64 `json:"outside share"`
		Stretch           float64
		Speed             float64
		FarObserver       four `json:"far observer"`
		Photon            four
		Toward            four
		Away              four
	}
	Geodesic struct {
		Position         four
		Momentum         four
		SpaceChanged     four `json:"space changed"`
		TimeChanged      four `json:"time changed"`
		Energy           float64
		Lz               float64
		Step             float64
		FlatTolerance    float64 `json:"flat tolerance"`
		HalvingTolerance float64 `json:"halving tolerance"`
		Start            place
		Aim              place
	}
	Keeps struct {
		FarRadius float64 `json:"far radius"`
		Steps     int
		StartH    float64 `json:"start h"`
		EndH      float64 `json:"end h"`
		Relative  float64
		BigStep   float64 `json:"big step"`
	}
	March struct {
		Far           float64
		Steps         int
		EyeRadii      []float64 `json:"eye radii"`
		TowardHole    place     `json:"toward the hole"`
		AwayFromHole  place     `json:"away from the hole"`
		EdgeEye       float64   `json:"edge eye"`
		Inside        place
		Outside       place
		JustOutside   place   `json:"just outside"`
		FallEye       place   `json:"fall eye"`
		FallAim       place   `json:"fall aim"`
		FallGraze     place   `json:"fall graze"`
		HoverSteps    int     `json:"hover steps"`
		HoverShare    float64 `json:"hover share"`
		HoverH        float64 `json:"hover h"`
		PaceShare     float64 `json:"pace share"`
		PaceTolerance float64 `json:"pace tolerance"`
		EdgeSteps     int     `json:"edge steps"`
		Shrinks       float64
	}
	Fall struct {
		Radius      float64
		Tolerance   float64
		BodyH       float64 `json:"body h"`
		Off         place
		Seconds     int
		Second      float64
		Early       float64
		EarlyRadius float64 `json:"early radius"`
		Near        float64
		BeforeLow   float64 `json:"before low"`
		BeforeHigh  float64 `json:"before high"`
		StopRadius  float64 `json:"stop radius"`
		Starts      []place
		Times       []float64
		Drift       float64
		AtTheMargin place `json:"at the margin"`
	}
	GPU struct {
		Step    float64
		Steps   int
		Reached float64
		Count   int
		Layout  struct {
			Size         int
			Float        int
			Mass         int
			Spin         int
			Horizon      int
			PhotonSphere int `json:"photon sphere"`
			Shadow       int
			Camera       int
			Frame        int
		}
	}
	Approach struct {
		Impact          float64
		From            float64
		Steps           int
		Turning         float64
		Tolerance       float64
		ExactTolerance  float64 `json:"exact tolerance"`
		OutFrom         place   `json:"out from"`
		InFrom          place   `json:"in from"`
		GivenUp         float64 `json:"given up"`
		AxisFrom        place   `json:"axis from"`
		AxisAim         place   `json:"axis aim"`
		AxisTolerance   float64 `json:"axis tolerance"`
		Across          float64
		ImpactTolerance float64 `json:"impact tolerance"`
		Disc            struct {
			Position  four
			Momentum  four
			Step      float64
			Tolerance float64
		}
	}
	Class struct {
		From         place
		AtTheHole    place `json:"at the hole"`
		Away         place
		Missing      place
		CircleRadius float64 `json:"circle radius"`
	}
	Sun      sunOf
	Weak     weakOf
	Straight straightOf
	Hair     struct {
		Hole  spec
		Place place
	}
	Dot struct {
		Covector  four
		Vector    four
		Want      float64
		TimeOnly  four `json:"time only"`
		SpaceOnly four `json:"space only"`
	}
	Anti struct {
		MomentumScale   float64 `json:"momentum scale"`
		ShortProperTime float64 `json:"short proper time"`
		SpareReports    int     `json:"spare reports"`
		OneStep         int     `json:"one step"`
		NotYetFar       float64 `json:"not yet far"`
		PositionsAgree  float64 `json:"positions agree"`
		SpinBend        struct {
			Start      float64
			Far        float64
			Steps      int
			Impact     float64
			Gap        float64
			SidesAgree float64 `json:"sides agree"`
		} `json:"spin bend"`
		CaptureSweep struct {
			Impacts []float64
		} `json:"capture sweep"`
		LargeBend struct {
			Impact float64
			Agree  float64
		} `json:"large bend"`
		BodyCarter struct {
			Position   place
			Velocity   place
			ProperTime float64 `json:"proper time"`
			Relative   float64
		} `json:"body carter"`
	}
	Draws struct {
		MassLow        float64 `json:"mass low"`
		MassHigh       float64 `json:"mass high"`
		SpinLow        float64 `json:"spin low"`
		SpinHigh       float64 `json:"spin high"`
		CoordinateLow  float64 `json:"coordinate low"`
		CoordinateHigh float64 `json:"coordinate high"`
		OrbitTolerance float64 `json:"orbit tolerance"`
		AimFloor       float64 `json:"aim floor"`
	}
	WeakSpinBend struct {
		Impact float64
		Start  float64
		Far    float64
		Steps  int
		Share  float64
	} `json:"weak spin bend"`
	IscoMidSpins struct {
		Radii []struct {
			Spin   float64
			Radius float64
		}
		Tolerance float64
	} `json:"isco mid spins"`
	Edges  edgesOf
	Moving struct {
		ProperTime   float64 `json:"proper time"`
		Aim          place
		CircleRadius float64 `json:"circle radius"`
		CircleTime   float64 `json:"circle time"`
		CircleShare  float64 `json:"circle share"`
		LeastTurn    float64 `json:"least turn"`
	}
	Thrown struct {
		Hole    spec
		From    float64
		HighTop float64 `json:"high top"`
		LowTop  float64 `json:"low top"`
		Steps   int
	}
	Pace struct {
		Aim      place
		Relative float64
		Loud     float64
	}
}

// edgesOf is the numbers for places where a rule of the fall or the march
// changes.
type edgesOf struct {
	SingularityMargin float64 `json:"singularity margin"`
	Double            spec
	TwoHorizons       spec    `json:"two horizons"`
	InnerHorizon      float64 `json:"inner horizon"`
	OnTheInnerHorizon place   `json:"on the inner horizon"`
	JustInsideInner   place   `json:"just inside the inner horizon"`
	OnTheHorizon      place   `json:"on the horizon"`
	HorizonLight      struct {
		Position, Momentum four
	} `json:"horizon light"`
	Hair         four
	OffTheHair   place `json:"off the hair"`
	FromInfinity four  `json:"from infinity"`
}

// stopper is what a test or a property is stopped with.
type stopper interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test stopper) *picked {
	test.Helper()
	raw, err := os.ReadFile("testdata/hole.json")
	if err != nil {
		test.Fatal(err)
	}
	var numbers picked
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return &numbers
}

// Numbers the tests use that the scale or the arithmetic defines.
const (
	// none is no quantity: the zero of a length, a rate or a sum. It is
	// defined by the arithmetic.
	none = 0
	// positive is the sign math.Inf takes for plus infinity. It is defined
	// by package math.
	positive = 1
)

// The shape of spacetime and its flat metric. Each is defined by relativity.
const (
	// dimensions is the four of spacetime. It is defined by relativity: one
	// of time and three of space.
	dimensions = 4
	// timeRow is the row and column of time in a metric. It is defined by
	// the order (t, x, y, z).
	timeRow = 0
	// timeSign is the flat metric's entry along time. It is defined by the
	// signature, minus, plus, plus, plus.
	timeSign = -1
	// spaceSign is the flat metric's entry along each axis of space. It is
	// defined by the signature.
	spaceSign = 1
	// identity is the identity matrix's entry on its diagonal. It is
	// defined by the identity matrix.
	identity = 1
)

// sunOf is the numbers for a ray grazing the Sun and rays at the edge of a
// shadow.
type sunOf struct {
	RadiusKm    float64 `json:"radius km"`
	MassKm      float64 `json:"mass km"`
	Distance    float64
	Reach       float64
	Beyond      float64
	Steps       int
	Arcseconds  float64
	Tolerance   float64
	JustInside  float64 `json:"just inside"`
	JustOutside float64 `json:"just outside"`
}

// weakOf is the numbers for the weak-field bending at a hundred M.
type weakOf struct {
	Impact      float64
	Start       float64
	Far         float64
	Steps       int
	Relative    float64
	PlanesAgree float64 `json:"planes agree"`
	XY          place   `json:"xy"`
	XZ          place   `json:"xz"`
	Tilted      place
}

// straightOf is the numbers for rays that are not bent.
type straightOf struct {
	From      place
	Outward   place
	Inward    place
	Far       float64
	Steps     int
	Tolerance float64
}

// relativeFloor is the size below which a relative tolerance is read as an
// absolute one, so a quantity near zero is not held to nothing. It is defined
// by this test: one, the unit of M.
const relativeFloor = 1

// The formulas the geodesic tests hold the package to.
const (
	// hamiltonianHalf is the half of H = (1/2) g^{ab} p_a p_b. It is defined
	// by the hamiltonian.
	hamiltonianHalf = 0.5
	// midpoint is half a step. It is defined by the arithmetic.
	midpoint = 0.5
)

// The formulas the frame tests hold the package to.
const (
	// timelikeNorm is a unit timelike vector's squared length. It is defined
	// by the signature, minus, plus, plus, plus.
	timelikeNorm = -1
	// unitEnergy is the energy Null gives its ray as its observer measures
	// it. It is defined by Null's own description: one.
	unitEnergy = 1
	// lorentzOne is the 1 of the Lorentz factor 1 / sqrt(1 - v^2) and of
	// the Doppler factor 1 / (gamma (1 - v)). It is defined by special
	// relativity.
	lorentzOne = 1
	// schwarzschildOne is the 1 of sqrt(1 - 2 M / r), the rate of a still
	// clock. It is defined by the Schwarzschild metric.
	schwarzschildOne = 1
	// schwarzschildTwo is the 2 of sqrt(1 - 2 M / r). It is defined by the
	// Schwarzschild metric.
	schwarzschildTwo = 2
)

// The equation of the equatorial photon orbit, r^2 - 3 M r +- 2 A sqrt(M r)
// = 0, plus for prograde, by Bardeen, Press and Teukolsky (1972): their
// equation 2.17 at its equality, times r^(1/2), read on 2026-10-31.
const (
	// orbitThree is the 3 of 3 M r. It is defined by that equation.
	orbitThree = 3
	// orbitTwo is the 2 of 2 A sqrt(M r). It is defined by that equation.
	orbitTwo = 2
	// progradeSign is the sign of the last term for a prograde orbit. It is
	// defined by that equation.
	progradeSign = 1
	// retrogradeSign is the sign of the last term for a retrograde orbit. It
	// is defined by that equation.
	retrogradeSign = -1
)

// build is the hole of a spec, or the test stops if there is none.
func build(test stopper, from spec) *hole.Hole {
	test.Helper()
	made, err := hole.NewHole(from.Mass, from.Spin)
	if err != nil {
		test.Fatalf("no hole of mass %v and spin %v: %v", from.Mass,
			from.Spin, err)
	}
	return made
}

// vec3 is a place as a hole's Vec3.
func vec3(at place) hole.Vec3 {
	return hole.NewVec3(at.AlongX, at.AlongY, at.AlongZ)
}

// within is whether two numbers are the same to an absolute tolerance. A
// number that is not a number is within nothing.
func within(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// drawnHole draws a hole whose mass and spin come from the ranges the data
// picks.
func drawnHole(test *rapid.T, draws *picked) *hole.Hole {
	mass := rapid.Float64Range(draws.Draws.MassLow, draws.Draws.MassHigh).
		Draw(test, "mass")
	part := rapid.Float64Range(draws.Draws.SpinLow, draws.Draws.SpinHigh).
		Draw(test, "spin as a part of the mass")
	made, err := hole.NewHole(mass, mass*part)
	if err != nil {
		test.Fatalf("no hole of mass %v and spin %v: %v", mass, mass*part,
			err)
	}
	return made
}

// drawnAim draws a direction from the range the data picks, and skips
// one too short to have a way: a direction of a hundred and sixty
// orders of magnitude under one has a length that underflows.
func drawnAim(test *rapid.T, draws *picked) hole.Vec3 {
	aim := drawnPlace(test, draws, "aim")
	length := math.Sqrt(aim.AlongX()*aim.AlongX() +
		aim.AlongY()*aim.AlongY() + aim.AlongZ()*aim.AlongZ())
	if length < draws.Draws.AimFloor {
		test.Skip("a direction too short to point anywhere")
	}
	return aim
}

// drawnPlace draws a point in space from the range the data picks.
func drawnPlace(test *rapid.T, draws *picked, name string) hole.Vec3 {
	coordinate := rapid.Float64Range(
		draws.Draws.CoordinateLow, draws.Draws.CoordinateHigh,
	)
	return hole.NewVec3(
		coordinate.Draw(test, name+" x"),
		coordinate.Draw(test, name+" y"),
		coordinate.Draw(test, name+" z"),
	)
}
