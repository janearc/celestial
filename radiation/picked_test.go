package radiation_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/radiation"
)

// edges is a range as the test data writes it: its short and long ends.
type edges struct {
	Lo float64 `json:"lo"`
	Hi float64 `json:"hi"`
}

// point is a wavelength and a temperature to read planck's law at.
type point struct {
	NM     float64 `json:"nm"`
	Kelvin float64 `json:"kelvin"`
}

// written is a profile as the test data writes it.
type written struct {
	Name   string `json:"name"`
	Source edges  `json:"source"`
	Render edges  `json:"render"`
}

// picked is the numbers these tests pick, from testdata/radiation.json.
type picked struct {
	Units struct {
		HC          float64 `json:"hc ev nm"`
		HCTolerance float64 `json:"hc tolerance"`
		KeVNM       float64 `json:"kev nm"`
		HydrogenMHz float64 `json:"hydrogen mhz"`
		HydrogenCM  float64 `json:"hydrogen cm"`
		HydrogenTol float64 `json:"hydrogen tolerance"`
		WiseMicron  float64 `json:"wise micron"`
		WiseNM      float64 `json:"wise nm"`
		AngstromNM  float64 `json:"angstrom nm"`
		Tolerance   float64 `json:"relative tolerance"`
		GammaNM     float64 `json:"gamma nm"`
		TenMetres   float64 `json:"ten metres"`
	}
	Ladders struct{ Energy, Frequency []float64 }
	Named   map[string]edges `json:"named ranges"`
	Jansky  struct {
		One       float64 `json:"one jansky"`
		CGSOne    float64 `json:"cgs one jansky"`
		ABZeroJy  float64 `json:"ab zero jy"`
		ABZeroSI  float64 `json:"ab zero si"`
		Tolerance float64
		Lowest    float64 `json:"lowest jy"`
		Highest   float64 `json:"highest jy"`
	}
	Planck planckNumbers `json:"planck"`
	Fermi  struct {
		Highest   float64 `json:"highest ev"`
		Lowest    float64 `json:"lowest ev"`
		Tolerance float64
	}
	Mapping struct {
		Target     edges
		Wide       edges
		JustInside float64   `json:"just inside"`
		Fractions  []float64 `json:"fractions"`
		Tolerance  float64
		Flat       written
		BackRend   written `json:"backwards render"`
		BackSrc    written `json:"backwards source"`
		ZeroSrc    written `json:"source from nothing"`
		NegRend    written `json:"render from below nothing"`
		Sound      written
		InsideNM   float64 `json:"inside nm"`
		AboveNM    float64 `json:"above nm"`
		RenderNM   float64 `json:"render inside nm"`
	}
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/radiation.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// stopper is a test that can stop: a testing.T, or rapid's.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// span is the range of the test data's edges, or the test stops.
func span(test stopper, written edges) radiation.Range {
	test.Helper()
	made, err := radiation.NewRange(written.Lo, written.Hi)
	if err != nil {
		test.Fatal(err)
	}
	return *made
}

// built is a profile and the ranges it was built of.
type built struct {
	*radiation.Profile
	name           string
	source, render radiation.Range
}

// profileOf is the profile of a name from a source onto a render, or the
// test stops.
func profileOf(test stopper, name string, source,
	render radiation.Range) built {
	test.Helper()
	made, err := radiation.NewProfile(name, source, render)
	if err != nil {
		test.Fatal(err)
	}
	return built{Profile: made, name: name, source: source, render: render}
}

// Numbers the tests use that the scale or the arithmetic defines.
const (
	// none is no wavelength. It is defined by the scale.
	none = 0
	// twice is the factor of doubling. It is defined by the arithmetic.
	twice = 2
)

// The range a drawn wavelength is drawn from, as powers of ten of a
// nanometre, so every decade is as likely as another. Each is defined by
// this test.
const (
	// shortestPower is the power of ten of the shortest wavelength drawn.
	// It is defined by this test: 1e-5 nm, a gamma ray.
	shortestPower = -5.0
	// longestPower is the power of ten of the longest wavelength drawn. It
	// is defined by this test: 1e9 nm, a wave a metre long.
	longestPower = 9.0
	// base is the base the drawn powers are of. It is defined by the
	// decimal system.
	base = 10.0
)

// wavelength draws a wavelength in nanometres, evenly in the logarithm.
func wavelength(test *rapid.T, name string) float64 {
	power := rapid.Float64Range(shortestPower, longestPower).Draw(test, name)
	return math.Pow(base, power)
}

// near is whether two numbers are the same to a relative tolerance.
func near(got, want, tolerance float64) bool {
	larger := math.Max(math.Abs(got), math.Abs(want))
	return math.Abs(got-want) <= tolerance*larger
}
