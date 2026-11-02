package atmosphere_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// base is a layer's base as the standard gives it.
type base struct {
	Height  float64 `json:"height"`
	Kelvin  float64 `json:"kelvin"`
	Pascals float64 `json:"pascals"`
}

// row is a row of Bodhaine and the others' table 3.
type row struct {
	Microns      float64 `json:"microns"`
	CrossSection float64 `json:"cross section cm2"`
	Depth        float64 `json:"depth"`
}

// tilt is a way from a height at an angle.
type tilt struct {
	Metres  float64 `json:"metres"`
	Degrees float64 `json:"degrees"`
}

// picked is the numbers these tests use, from testdata/atmosphere.json.
type picked struct {
	EarthRadius float64 `json:"earth radius m"`
	Sea         struct {
		Kelvin     float64 `json:"kelvin"`
		Pascals    float64 `json:"pascals"`
		Density    float64 `json:"density"`
		KelvinTol  float64 `json:"kelvin tolerance"`
		PascalTol  float64 `json:"pascal tolerance"`
		DensityTol float64 `json:"density tolerance"`
	} `json:"sea"`
	Bases         []base    `json:"bases"`
	BaseKelvinTol float64   `json:"base kelvin tolerance"`
	BasePascalTol float64   `json:"base pascal tolerance"`
	U2Feet        float64   `json:"u-2 feet"`
	MetresAFoot   float64   `json:"metres a foot"`
	U2Above       float64   `json:"u-2 air above"`
	OutsideMetres []float64 `json:"outside metres"`
	CO2           float64   `json:"co2 ppm"`
	NotCO2        []float64 `json:"not co2 ppm"`
	AllRound      struct {
		Look      wayParts `json:"look"`
		Sun       wayParts `json:"sun"`
		Tolerance float64  `json:"tolerance"`
	} `json:"all round"`
	Three struct {
		Metres    []float64 `json:"metres"`
		Tolerance float64   `json:"tolerance"`
	} `json:"three ways"`
	Edges struct {
		Metres []float64 `json:"metres"`
		CO2    []float64 `json:"co2 ppm"`
	} `json:"edges"`
	Bodhaine       []row     `json:"bodhaine"`
	BodhaineTol    float64   `json:"bodhaine tolerance"`
	Latitude       float64   `json:"bodhaine latitude"`
	OutsideMicrons []float64 `json:"outside microns"`
	StraightUp     float64   `json:"straight up degrees"`
	GreenMicrons   float64   `json:"green microns"`
	CountTol       float64   `json:"count tolerance"`
	KastenYoung    struct {
		Scale  float64 `json:"scale"`
		Offset float64 `json:"offset"`
		Power  float64 `json:"power"`
	} `json:"kasten young"`
	AirMassDegrees []float64 `json:"air mass degrees"`
	WayDown        tilt      `json:"way down"`
	WayUp          tilt      `json:"way up"`
	LowSun         float64   `json:"low sun degrees"`
	BlueMicrons    float64   `json:"blue microns"`
	RedMicrons     float64   `json:"red microns"`
	ThinAir        float64   `json:"thin air metres"`
	ClosedFormTol  float64   `json:"closed form tolerance"`
	BlueNM         float64   `json:"blue nm"`
	RedNM          float64   `json:"red nm"`
	GreenNM        float64   `json:"green nm"`
	BlueOverRed    float64   `json:"blue over red"`
	SunUpRadians   float64   `json:"sun up radians"`
	AfterSunset    float64   `json:"after sunset share"`
	Drawn          drawn     `json:"drawn"`
	Anti           anti      `json:"anti"`
	Beyond         struct {
		Up    wayParts `json:"up"`
		Sun   wayParts `json:"sun"`
		Level wayParts `json:"level"`
	} `json:"beyond squaring"`
}

// wayParts is a way's three parts as the test data writes them.
type wayParts struct {
	AlongX float64 `json:"x"`
	AlongY float64 `json:"y"`
	AlongZ float64 `json:"z"`
}

// anti is what the anti-tests compare: more CO2, a share of the sea's pressure
// and how near its depth must come, a way from higher up, and the equator
// against a pole.
type anti struct {
	MoreCO2  float64 `json:"more co2 ppm"`
	Share    float64 `json:"share of pressure"`
	DepthTol float64 `json:"depth tolerance"`
	Higher   float64 `json:"higher metres"`
	Rise     float64 `json:"rise degrees"`
	Equator  float64 `json:"equator degrees"`
	Pole     float64 `json:"pole degrees"`
}

// drawn is what the properties draw from: the standard's top, how far
// apart two heights or two wavelengths are at least, and how far outside
// a height or a wavelength refused is.
type drawn struct {
	Top            float64 `json:"standard top metres"`
	LeastRise      float64 `json:"least rise metres"`
	OutsideMetres  float64 `json:"outside by metres"`
	LeastStep      float64 `json:"least step microns"`
	OutsideMicrons float64 `json:"outside by microns"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/atmosphere.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// The numbers the tests compare with, each defined.
const (
	nothing = 0.0 // defined by the scale
	whole   = 1.0 // defined: a ratio of a thing to itself
	first   = 0   // defined by a slice's order
	second  = 1   // defined by a slice's order
	upward  = 1   // defined by math.Inf: the endless above
)

// near is whether a value is within a share of what is expected.
func near(got, want, share float64) bool {
	return math.Abs(got/want-whole) <= share
}

// microns is a length of so many microns.
func microns(amount float64) measure.Length {
	return measure.Metres(si.Micro(amount))
}

// nanometres is a length of so many nanometres.
func nanometres(amount float64) measure.Length {
	return measure.Metres(si.Nano(amount))
}

// dry is dry air of the test data's CO2, or the test stops.
func dry(test *testing.T) atmosphere.Composition {
	test.Helper()
	air, err := atmosphere.DryAir(pickedNumbers(test).CO2)
	if err != nil {
		test.Fatal(err)
	}
	return *air
}

// standard is the standard air at a height, or the test stops.
func standard(test *testing.T, height measure.Length) atmosphere.Air {
	test.Helper()
	air, err := atmosphere.Standard(height)
	if err != nil {
		test.Fatal(err)
	}
	return *air
}

// column is the air along a way, or the test stops.
func column(test *testing.T, metres, degrees float64) atmosphere.Column {
	test.Helper()
	along, err := atmosphere.Along(measure.Metres(metres),
		measure.Degrees(degrees))
	if err != nil {
		test.Fatal(err)
	}
	return *along
}

// way is a way of three parts, or the test stops.
func way(test *testing.T, x, y, z float64) atmosphere.Way {
	test.Helper()
	made, err := atmosphere.NewWay(x, y, z)
	if err != nil {
		test.Fatal(err)
	}
	return *made
}
