package bands_test

import (
	"encoding/json"
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/bands"
)

// picked is the numbers these tests pick, from testdata/bands.json.
type picked struct {
	Spectrum      map[int]float64 `json:"spectrum"`
	OtherSpectrum map[int]float64 `json:"other spectrum"`
	OctaveNM      []float64       `json:"octave nm"`
	Shift         int             `json:"shift"`
	Visible       struct {
		Short float64 `json:"short nm"`
		Long  float64 `json:"long nm"`
		First int     `json:"first band"`
		Last  int     `json:"last band"`
		Width int     `json:"width"`
	}
	SameBand struct {
		Low   int     `json:"low nm"`
		High  int     `json:"high nm"`
		LowP  float64 `json:"low part"`
		HighP float64 `json:"high part"`
		Mean  float64 `json:"mean"`
	} `json:"same band"`
	Between struct {
		Low    int     `json:"low nm"`
		High   int     `json:"high nm"`
		LowP   float64 `json:"low part"`
		HighP  float64 `json:"high part"`
		Middle float64 `json:"middle nm"`
		Sample float64 `json:"sample part"`
	}
	Run struct {
		Low     int     `json:"low nm"`
		High    int     `json:"high nm"`
		Inside  float64 `json:"inside nm"`
		Outside float64 `json:"outside nm"`
	}
	Half       float64   `json:"half"`
	Refused    []float64 `json:"refused parts"`
	BadNM      []int     `json:"bad nm"`
	ShortestNM int       `json:"shortest nm"`
	LinePower  float64   `json:"line power"`
	LineNM     float64   `json:"line nm"`
	GammaNM    float64   `json:"gamma nm"`
	GammaPower float64   `json:"gamma power"`
	InfraredNM []int     `json:"infrared nm"`
	Repeats    int       `json:"repeats"`
	Band       int       `json:"band"`
	Tolerance  float64   `json:"relative tolerance"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/bands.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// Numbers the tests use that the scale or the arithmetic defines.
const (
	// none is no wavelength, no power and no steps. It is defined by the
	// scale.
	none = 0
	// unitPower is a power of exactly one. It is defined by the scale.
	unitPower = 1.0
	// twice is the factor of doubling. It is defined by the arithmetic.
	twice = 2
	// oneNM is the step from one whole nanometre to the next. It is defined
	// by the unit.
	oneNM = 1
	// neighbour is the distance from a band to the one beside it. It is
	// defined by the numbering: bands are counted from one.
	neighbour bands.Band = 1
	// relativeTolerance is how near two sums of floats must be to be the
	// same sum. It is defined by this test: float addition is not exact.
	relativeTolerance = 1e-9
)

// The ranges a drawn spectrum is drawn from. Each is defined by this test.
const (
	// shortestDrawn is the shortest wavelength drawn, in nanometres. It is
	// defined by this test: band 0 starts here.
	shortestDrawn = 1
	// longestDrawn is the longest wavelength drawn, in nanometres. It is
	// defined by this test: far into the infrared.
	longestDrawn = 30000
	// strongestDrawn is the most power a drawn wavelength has. It is defined
	// by this test.
	strongestDrawn = 100.0
	// fewestDrawn is the fewest wavelengths a drawn spectrum has. It is
	// defined by this test: a light needs one line.
	fewestDrawn = 1
	// mostDrawn is the most wavelengths a drawn spectrum has. It is defined by
	// this test.
	mostDrawn = 40
	// weakest is the least a drawn light is scaled by. It is defined by this
	// test.
	weakest = 0.1
	// strongest is the most a drawn light is scaled by. It is defined by this
	// test.
	strongest = 10.0
	// slide is how many bands a drawn light is slid, at most. It is defined by
	// this test: enough to make the spans differ.
	slide = 30
)

// spectrum draws a spectrum: whole wavelengths and a power at each.
func spectrum(test *rapid.T, name string) map[int]float64 {
	return rapid.MapOfN(
		rapid.IntRange(shortestDrawn, longestDrawn),
		rapid.Float64Range(none, strongestDrawn),
		fewestDrawn, mostDrawn).Draw(test, name)
}

// lamp is a drawn light: a drawn spectrum, made some strength and slid
// some bands, so two of them have spans that differ.
func lamp(test *rapid.T, name string) bands.Light {
	strength := rapid.Float64Range(weakest, strongest).Draw(test, name+" strength")
	steps := rapid.IntRange(-slide, slide).Draw(test, name+" slide")
	return bands.Split(spectrum(test, name)).Scale(strength).Shift(steps)
}

// powers is every band's power in a light across a span.
func powers(light bands.Light, first, last bands.Band) []float64 {
	var out []float64
	for band := first; band <= last; band++ {
		out = append(out, light.Power(band))
	}
	return out
}

// sameLight is whether two lights have the same power in every band either
// has an entry for.
func sameLight(left, right bands.Light) bool {
	leftFirst, leftLast := left.Span()
	rightFirst, rightLast := right.Span()
	first, last := min(leftFirst, rightFirst), max(leftLast, rightLast)
	got, want := powers(left, first, last), powers(right, first, last)
	for place := range got {
		if got[place] != want[place] {
			return false
		}
	}
	return true
}
