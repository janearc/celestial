package measure_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
)

// The numbers these tests compare with, each defined.
const (
	// nothing is no amount at all. Defined by the scale.
	nothing = 0.0
	// eitherSign asks math.IsInf about both infinities. Defined by math.
	eitherSign = 0
)

// The sun is the IAU's nominal sun, and its mass is its GM over the
// research data's G.
func TestTheSunIsWhatTheIAUSays(test *testing.T) {
	want := pickedNumbers(test)
	sun := measure.Sun()
	if kilograms := sun.Mass().InKilograms(); !near(kilograms, want.SunKg,
		want.SunTol) {
		test.Errorf("the sun weighs %v kg", kilograms)
	}
	if metres := sun.Radius().InMetres(); !near(metres, want.SunRadius,
		want.TripTolerance) {
		test.Errorf("the sun is %v m across", metres)
	}
	if watts := sun.Luminosity().InWatts(); !near(watts, want.SunWatts,
		want.TripTolerance) {
		test.Errorf("the sun shines %v W", watts)
	}
	gm := sun.GM().InMetres() * si.Speed * si.Speed
	if !near(gm, want.SunGM, want.SunGMTol) {
		test.Errorf("the sun's GM is %v m3 s-2, not %v", gm, want.SunGM)
	}
}

// The anti-test: the sun's mass is its GM over G, and not its GM times
// G, nor its GM taken as a mass, which a slip in the order of a division
// would give: its mass in kilograms, times si's G, is the iau's nominal
// GM again, typed from the resolution and not worked out by measure.
func TestTheSunsMassTimesGIsTheIAUsGM(test *testing.T) {
	want := pickedNumbers(test)
	if !(want.SunGM > nothing) {
		test.Fatalf("no GM to hold to: %v", want.SunGM)
	}
	gm := measure.Sun().Mass().InKilograms() * si.Gravitation()
	if !near(gm, want.SunGM, want.SunGMTol) {
		test.Errorf("the sun's mass times G is %v m3 s-2, not %v", gm,
			want.SunGM)
	}
}

// Light of a wavelength carries a full turn over it: green at 500 nm
// carries about 2.48 eV.
func TestAPhotonsEnergyIsItsColour(test *testing.T) {
	want := pickedNumbers(test)
	green := measure.Photon(measure.Metres(want.GreenM)).InElectronvolts()
	if !near(green, want.GreenEV, want.GreenTol) {
		test.Errorf("a %v m photon carries %v eV", want.GreenM, green)
	}
}

// The anti-test: longer light carries less, so a slip that multiplied by
// the wavelength shows.
func TestRedCarriesLessThanGreen(test *testing.T) {
	want := pickedNumbers(test)
	green := measure.Photon(measure.Metres(want.GreenM))
	red := measure.Photon(measure.Metres(want.RedM))
	if red.InElectronvolts() >= green.InElectronvolts() {
		test.Errorf("red carries %v eV and green %v",
			red.InElectronvolts(), green.InElectronvolts())
	}
}

// The sun's temperature follows from its light and its size, to the
// IAU's nominal 5772 K.
func TestTheSunsTemperatureFollowsFromItsLightAndSize(test *testing.T) {
	want := pickedNumbers(test)
	sun := measure.Sun()
	surface := measure.Sphere(sun.Radius())
	heat := measure.SurfaceTemperature(sun.Luminosity(), surface)
	if kelvin := heat.InKelvin(); !near(kelvin, want.SunKelvin,
		want.SunKelvinTol) {
		test.Errorf("the sun is %v kelvin, not %v", kelvin, want.SunKelvin)
	}
}

// The anti-test: the same light from twice the sphere is cooler, so the
// area is where the law says it is.
func TestTheSameLightFromMoreSurfaceIsCooler(test *testing.T) {
	want := pickedNumbers(test)
	sun := measure.Sun()
	surface := measure.Sphere(sun.Radius())
	larger := measure.Sphere(sun.Radius().Times(want.Twice))
	heat := measure.SurfaceTemperature(sun.Luminosity(), surface)
	cooler := measure.SurfaceTemperature(sun.Luminosity(), larger)
	if cooler.InKelvin() >= heat.InKelvin() {
		test.Errorf("twice the sun's size is %v kelvin, the sun %v",
			cooler.InKelvin(), heat.InKelvin())
	}
}

// The eddington limit of a sun's mass is about thirty thousand suns, and
// twice the mass may shine twice as bright.
func TestTheEddingtonLimitIsAboutThirtyThousandSuns(test *testing.T) {
	want := pickedNumbers(test)
	limit := measure.Eddington(measure.SolarMasses(want.One)).
		InSolarLuminosities()
	if !near(limit, want.EddingtonSuns, want.EddingtonTol) {
		test.Errorf("a sun's mass can shine %v suns at most", limit)
	}
	twice := measure.Eddington(measure.SolarMasses(want.Twice)).
		InSolarLuminosities()
	if !near(twice, want.Twice*limit, want.LinearTol) {
		test.Errorf("twice the mass may shine %v, not twice %v", twice,
			limit)
	}
}

// The anti-test: what is no mass, or less than none, or not a number, is
// not positive, and a body refuses it.
func TestWhatWeighsNothingIsNotPositive(test *testing.T) {
	for _, suns := range pickedNumbers(test).NotMasses {
		if measure.SolarMasses(suns).Positive() {
			test.Errorf("%v suns is positive", suns)
		}
	}
	for _, none := range []float64{math.NaN(), math.Inf(eitherSign)} {
		if measure.Kilograms(none).Positive() {
			test.Errorf("%v kilograms is positive", none)
		}
	}
	if !measure.Sun().Mass().Positive() {
		test.Error("the sun is not positive")
	}
}

// Half a turn is pi radians.
func TestHalfATurnIsPiRadians(test *testing.T) {
	want := pickedNumbers(test)
	radians := measure.Degrees(want.HalfTurn).InRadians()
	if !near(radians, math.Pi, want.TripTolerance) {
		test.Errorf("half a turn is %v radians", radians)
	}
}

// The anti-test: a degree is not a radian, and an arcsecond is not a
// degree.
func TestADegreeIsNotARadian(test *testing.T) {
	want := pickedNumbers(test)
	if measure.Degrees(want.OneDegree).InRadians() == want.OneDegree {
		test.Error("a degree is a radian")
	}
	if measure.Arcseconds(want.OneDegree).InDegrees() == want.OneDegree {
		test.Error("an arcsecond is a degree")
	}
}

// The universe is as old as the research data says, in years.
func TestTheCosmosIsAsOldAsTheDataSays(test *testing.T) {
	years := research.Value("cosmos-years")
	if got := measure.Cosmos().InYears(); !near(got, years,
		pickedNumbers(test).TripTolerance) {
		test.Errorf("the universe is %v years old, not %v", got, years)
	}
}

// The anti-test: the universe's age is in years, not in seconds read as
// years.
func TestTheCosmosIsNotItsSeconds(test *testing.T) {
	cosmos := measure.Cosmos()
	if cosmos.InSeconds() == cosmos.InYears() {
		test.Error("the universe's age in seconds is its age in years")
	}
}
