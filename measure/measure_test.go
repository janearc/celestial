package measure_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// amounts is one amount of each unit measure takes in and gives out.
type amounts struct {
	Kilograms, Suns, Kelvin, Metres     float64
	SolarRadii                          float64 `json:"solar radii"`
	Seconds, Years, Watts, Sunlight     float64
	SquareMetres                        float64 `json:"square metres"`
	Joules, MeV, Electronvolts, Radians float64
	Degrees, Arcseconds                 float64
}

// picked is the numbers these tests check against, from
// testdata/measure.json, which says where each came from.
type picked struct {
	Trips          amounts   `json:"round trips"`
	Fused          []float64 `json:"fused"`
	SunGM          float64   `json:"sun gm"`
	SunGMTol       float64   `json:"sun gm tolerance"`
	TripTolerance  float64   `json:"round trip tolerance"`
	KilogramMeV    float64   `json:"kilogram mev"`
	KilogramTol    float64   `json:"kilogram tolerance"`
	ProtonKg       float64   `json:"proton kg"`
	ProtonTol      float64   `json:"proton tolerance"`
	One            float64   `json:"one"`
	Stefan         float64   `json:"stefan"`
	StefanTol      float64   `json:"stefan tolerance"`
	OneKelvin      float64   `json:"one kelvin"`
	OneSquareMetre float64   `json:"one square metre"`
	BackTol        float64   `json:"back tolerance"`
	ThomsonM2      float64   `json:"thomson m2"`
	ThomsonTol     float64   `json:"thomson tolerance"`
	SunKg          float64   `json:"sun kg"`
	SunTol         float64   `json:"sun tolerance"`
	SunRadius      float64   `json:"sun radius m"`
	SunWatts       float64   `json:"sun watts"`
	GreenM         float64   `json:"green m"`
	GreenEV        float64   `json:"green ev"`
	GreenTol       float64   `json:"green tolerance"`
	RedM           float64   `json:"red m"`
	SunKelvin      float64   `json:"sun kelvin"`
	SunKelvinTol   float64   `json:"sun kelvin tolerance"`
	EddingtonSuns  float64   `json:"eddington suns"`
	EddingtonTol   float64   `json:"eddington tolerance"`
	Twice          float64   `json:"twice"`
	LinearTol      float64   `json:"linear tolerance"`
	HalfTurn       float64   `json:"half turn degrees"`
	OneDegree      float64   `json:"one degree"`
	NotMasses      []float64 `json:"not masses"`
	Shares         struct{ Least, Most float64 }
	Hydrogens      float64 `json:"hydrogens in helium"`
	Deficit        float64 `json:"fusion deficit"`
	DeficitTol     float64 `json:"fusion tolerance"`
	SunLifetime    float64 `json:"sun lifetime years"`
	Arcseconds     float64 `json:"arcseconds a degree"`
	YearSeconds    float64 `json:"julian year seconds"`
	Spent          struct {
		Joules  float64 `json:"joules"`
		Watts   float64 `json:"watts"`
		Seconds float64 `json:"seconds"`
	} `json:"spent"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/measure.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// near is whether a value lands within a share of what is published for
// it.
func near(got, published, share float64) bool {
	return math.Abs(got-published) <= share*math.Abs(published)
}

// tripper checks that an amount came back as it went.
type tripper struct {
	test      *testing.T
	tolerance float64
}

// check says so when an amount did not come back as it went.
func (trips tripper) check(name string, went, came float64) {
	trips.test.Helper()
	if !near(came, went, trips.tolerance) {
		trips.test.Errorf("%s went in as %v and came back %v", name,
			went, came)
	}
}

// Every unit, in and out again, to sixteen digits.
func TestEveryUnitComesBackAsItWent(test *testing.T) {
	want := pickedNumbers(test)
	in := want.Trips
	trips := tripper{test: test, tolerance: want.TripTolerance}
	trips.check("kilograms", in.Kilograms,
		measure.Kilograms(in.Kilograms).InKilograms())
	trips.check("suns", in.Suns, measure.SolarMasses(in.Suns).InSolarMasses())
	trips.check("kelvin", in.Kelvin, measure.Kelvin(in.Kelvin).InKelvin())
	trips.check("metres", in.Metres, measure.Metres(in.Metres).InMetres())
	trips.check("solar radii", in.SolarRadii,
		measure.SolarRadii(in.SolarRadii).InSolarRadii())
	trips.check("seconds", in.Seconds,
		measure.Seconds(in.Seconds).InSeconds())
	trips.check("years", in.Years, measure.Years(in.Years).InYears())
	trips.check("watts", in.Watts, measure.Watts(in.Watts).InWatts())
	trips.check("sunlight", in.Sunlight,
		measure.SolarLuminosities(in.Sunlight).InSolarLuminosities())
	trips.check("square metres", in.SquareMetres,
		measure.SquareMetres(in.SquareMetres).InSquareMetres())
	trips.check("joules", in.Joules, measure.Joules(in.Joules).InJoules())
	trips.check("mev", in.MeV, measure.MeV(in.MeV).InMeV())
	trips.check("electronvolts", in.Electronvolts,
		measure.Electronvolts(in.Electronvolts).InElectronvolts())
	trips.check("radians", in.Radians,
		measure.Radians(in.Radians).InRadians())
	trips.check("degrees", in.Degrees,
		measure.Degrees(in.Degrees).InDegrees())
	trips.check("arcseconds", in.Arcseconds,
		measure.Arcseconds(in.Arcseconds).InArcseconds())
}

// The anti-test: a unit is converted, not passed through. A kilogram in
// MeV is not a kilogram's number, nor is a metre or a second in natural
// units, so a slip that dropped a conversion shows.
func TestNoUnitIsItsOwnNaturalNumber(test *testing.T) {
	in := pickedNumbers(test).Trips
	if measure.Kilograms(in.Kilograms).InMeV() == in.Kilograms {
		test.Error("a mass in kilograms is its own MeV")
	}
	if measure.Kelvin(in.Kelvin).InMeV() == in.Kelvin {
		test.Error("a temperature in kelvin is its own MeV")
	}
	if measure.Electronvolts(in.Electronvolts).InMeV() == in.Electronvolts {
		test.Error("an energy in electronvolts is its own MeV")
	}
}

// A kilogram is its energy: c squared over the joules in an MeV. Every
// part is defined, so it is exact.
func TestAKilogramIsItsEnergy(test *testing.T) {
	want := pickedNumbers(test)
	mev := measure.Kilograms(want.One).InMeV()
	if !near(mev, want.KilogramMeV, want.KilogramTol) {
		test.Errorf("a kilogram is %v MeV", mev)
	}
}

// The anti-test: a kilogram's energy is not c squared taken as MeV, the
// slip of a missed joule.
func TestAKilogramsEnergyIsNotCSquaredInMeV(test *testing.T) {
	want := pickedNumbers(test)
	mev := measure.Kilograms(want.One).InMeV()
	if near(mev, si.Speed*si.Speed, want.KilogramTol) {
		test.Errorf("a kilogram is c squared, %v MeV", mev)
	}
}

// The proton weighs what CODATA 2022 says, in kilograms, from its energy
// at rest in the research data.
func TestTheProtonWeighsWhatCODATASays(test *testing.T) {
	want := pickedNumbers(test)
	kilograms := measure.Proton().InKilograms()
	if !near(kilograms, want.ProtonKg, want.ProtonTol) {
		test.Errorf("the proton weighs %v kg", kilograms)
	}
}

// The anti-test: the proton is not the dalton, nor the hydrogen atom it
// is the heart of. It is lighter than the atom by an electron and its
// binding, and heavier than a dalton.
func TestTheProtonIsNeitherTheDaltonNorTheAtom(test *testing.T) {
	proton := measure.Proton()
	dalton := measure.Daltons(pickedNumbers(test).One)
	atom := measure.Hydrogen1()
	if proton.Over(dalton) <= pickedNumbers(test).One {
		test.Errorf("the proton is %v daltons", proton.Over(dalton))
	}
	if proton.Over(atom) >= pickedNumbers(test).One {
		test.Errorf("the proton is %v hydrogen atoms", proton.Over(atom))
	}
	withElectron := atom.Minus(measure.Electron())
	if withElectron.Over(proton) >= pickedNumbers(test).One {
		test.Error("the atom less its electron outweighs its proton")
	}
}

// A kilogram's GM, as a length, is G over c squared metres: the research
// data's G, carried through natural units and back.
func TestAKilogramsGMIsGOverCSquared(test *testing.T) {
	want := pickedNumbers(test)
	gm := measure.GM(measure.Kilograms(want.One)).InMetres()
	expected := si.Gravitation() / (si.Speed * si.Speed)
	if !near(gm, expected, want.LinearTol) {
		test.Errorf("a kilogram's GM is %v m, not %v", gm, expected)
	}
}

// The anti-test: a kilogram's GM is not G itself, the slip of a
// conversion left out.
func TestAKilogramsGMIsNotG(test *testing.T) {
	want := pickedNumbers(test)
	gm := measure.GM(measure.Kilograms(want.One)).InMetres()
	if near(gm, si.Gravitation(), want.LinearTol) {
		test.Errorf("a kilogram's GM is G, %v", gm)
	}
}

// Stefan's constant follows from Planck's law alone: a square metre at one
// kelvin gives off what CODATA says, which is exact, and that glow comes from
// one kelvin.
func TestStefansConstantIsPiSquaredOverSixty(test *testing.T) {
	want := pickedNumbers(test)
	area := measure.SquareMetres(want.OneSquareMetre)
	glow := measure.BlackBody(measure.Kelvin(want.OneKelvin), area)
	if !near(glow.InWatts(), want.Stefan, want.StefanTol) {
		test.Errorf("stefan's constant is %v W m-2 K-4", glow.InWatts())
	}
	back := measure.SurfaceTemperature(glow, area).InKelvin()
	if !near(back, want.OneKelvin, want.BackTol) {
		test.Errorf("that glow comes from %v kelvin", back)
	}
}

// The anti-test: a surface with no area gives no finite temperature for
// a glow. No answer is made up for it.
func TestNoAreaHasNoTemperature(test *testing.T) {
	want := pickedNumbers(test)
	glow := measure.Watts(want.SunWatts)
	none := measure.SquareMetres(nothing)
	heat := measure.SurfaceTemperature(glow, none).InKelvin()
	if !math.IsInf(heat, eitherSign) {
		test.Errorf("a glow from no area is %v kelvin", heat)
	}
}

// Thomson's cross-section follows from the fine-structure constant and
// the electron's mass, both from the research data: CODATA 2022's value.
func TestThomsonFollowsFromTheFineStructure(test *testing.T) {
	want := pickedNumbers(test)
	area := measure.Thomson().InSquareMetres()
	if !near(area, want.ThomsonM2, want.ThomsonTol) {
		test.Errorf("thomson's cross-section is %v m2", area)
	}
}

// The anti-test: a heavier particle turns less of a beam aside. Thomson
// is for the electron; the proton's would be smaller by their mass ratio
// squared, so the electron's is no proton's.
func TestThomsonIsTheElectrons(test *testing.T) {
	want := pickedNumbers(test)
	ratio := measure.Proton().Over(measure.Electron())
	protonsArea := measure.Thomson().InSquareMetres() / (ratio * ratio)
	if near(protonsArea, want.ThomsonM2, want.ThomsonTol) {
		test.Errorf("the proton turns aside %v m2, as much as the "+
			"electron", protonsArea)
	}
}
