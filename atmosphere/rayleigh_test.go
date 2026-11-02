package atmosphere_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
	"pgregory.net/rapid"
)

// squareCentimetre is a square centimetre in square metres. Defined by the SI's
// centi.
var squareCentimetre = si.Centi(whole) * si.Centi(whole)

// A molecule of dry air scatters what Bodhaine's table says, to its five
// figures.
func TestAMoleculeScattersWhatBodhaineSays(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	for _, each := range numbers.Bodhaine {
		area, err := air.CrossSection(microns(each.Microns))
		if err != nil {
			test.Fatal(err)
		}
		got := area.InSquareMetres() / squareCentimetre
		if !near(got, each.CrossSection, numbers.BodhaineTol) {
			test.Errorf("at %g um a molecule scatters %.5g cm^2; "+
				"bodhaine says %.5g", each.Microns, got, each.CrossSection)
		}
	}
}

// The air over sea level at 45 degrees is as deep, to scattering, as Bodhaine's
// table says, to its five figures.
func TestTheSkyIsAsDeepAsBodhaineSays(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	for _, each := range numbers.Bodhaine {
		if each.Depth == nothing {
			continue
		}
		depth, err := air.DepthAbove(microns(each.Microns),
			atmosphere.Pascals(numbers.Sea.Pascals),
			measure.Degrees(numbers.Latitude), measure.Metres(nothing))
		if err != nil {
			test.Fatal(err)
		}
		if !near(*depth, each.Depth, numbers.BodhaineTol) {
			test.Errorf("at %g um the sky is %.6f deep; bodhaine says "+
				"%.6f", each.Microns, *depth, each.Depth)
		}
	}
}

// The anti-test: optical depth must use local pressure. At half sea-level
// pressure, fixed height and latitude give half the depth within rounding. The
// sea-level table alone cannot detect pressure being ignored.
func TestHalfTheAirIsHalfTheDepth(test *testing.T) {
	numbers := pickedNumbers(test)
	share := numbers.Anti.Share
	if !(share > nothing && share < whole) {
		test.Fatalf("no share of the air to hold: %v", share)
	}
	air := dry(test)
	green := microns(numbers.GreenMicrons)
	latitude := measure.Degrees(numbers.Latitude)
	full, err := air.DepthAbove(green,
		atmosphere.Pascals(numbers.Sea.Pascals), latitude,
		measure.Metres(nothing))
	if err != nil {
		test.Fatal(err)
	}
	part, err := air.DepthAbove(green,
		atmosphere.Pascals(share*numbers.Sea.Pascals), latitude,
		measure.Metres(nothing))
	if err != nil {
		test.Fatal(err)
	}
	if !near(*part, share*(*full), numbers.Anti.DepthTol) {
		test.Errorf("%v of the air is %v deep, not %v of %v", share, *part,
			share, *full)
	}
}

// The anti-test: latitude affects optical depth through gravity. For equal
// pressure, weaker equatorial gravity gives a deeper column than at a pole. The
// reference table’s 45-degree latitude cannot catch a missing or sign-reversed
// latitude term.
func TestTheEquatorsSkyIsDeeperThanAPoles(test *testing.T) {
	numbers := pickedNumbers(test)
	equator := greenDepthAt(test, numbers.Anti.Equator)
	pole := greenDepthAt(test, numbers.Anti.Pole)
	if !(equator > pole) {
		test.Errorf("the sky is %v deep at the equator and %v at a pole",
			equator, pole)
	}
}

// greenDepthAt is the depth of the test data's dry air to green light,
// over the sea at the sea's pressure, at a latitude, or the test stops.
func greenDepthAt(test *testing.T, degrees float64) float64 {
	test.Helper()
	numbers := pickedNumbers(test)
	depth, err := dry(test).DepthAbove(microns(numbers.GreenMicrons),
		atmosphere.Pascals(numbers.Sea.Pascals), measure.Degrees(degrees),
		measure.Metres(nothing))
	if err != nil {
		test.Fatal(err)
	}
	return *depth
}

// The anti-test: CO2 contributes to scattering. Increasing it must increase
// cross-section at each reference wavelength; the table’s single 360 ppm
// mixture would miss a model that ignored CO2.
func TestMoreCO2ScattersMore(test *testing.T) {
	numbers := pickedNumbers(test)
	if !(numbers.Anti.MoreCO2 > numbers.CO2) {
		test.Fatalf("no more co2 to hold: %v", numbers.Anti.MoreCO2)
	}
	thicker, err := atmosphere.DryAir(numbers.Anti.MoreCO2)
	if err != nil {
		test.Fatal(err)
	}
	air := dry(test)
	for _, each := range numbers.Bodhaine {
		wavelength := microns(each.Microns)
		less, err := air.CrossSection(wavelength)
		if err != nil {
			test.Fatal(err)
		}
		more, err := thicker.CrossSection(wavelength)
		if err != nil {
			test.Fatal(err)
		}
		if !(more.InSquareMetres() > less.InSquareMetres()) {
			test.Errorf("at %g um, %v ppm scatters %v m2, %v ppm %v",
				each.Microns, numbers.Anti.MoreCO2, more.InSquareMetres(),
				numbers.CO2, less.InSquareMetres())
		}
	}
}

// The anti-test: Peck and Reeder's refractive index holds from 0.23 to 1.69
// microns; outside it, and at no number, the air's scattering is not known
// here, and every way of asking for it is refused.
func TestWavelengthsOutsideTheIndexAreRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	along := column(test, nothing, numbers.StraightUp)
	sea := atmosphere.Pascals(numbers.Sea.Pascals)
	for _, each := range append(numbers.OutsideMicrons, math.NaN()) {
		wavelength := microns(each)
		if area, err := air.CrossSection(wavelength); err == nil ||
			area != nil {
			test.Errorf("%g um is scattered: %v, %v", each, area, err)
		}
		if _, err := air.DepthAbove(wavelength, sea,
			measure.Degrees(numbers.Latitude),
			measure.Metres(nothing)); err == nil {
			test.Errorf("the sky is deep at %g um", each)
		}
		if _, err := air.Through(along, wavelength); err == nil {
			test.Errorf("light gets through at %g um", each)
		}
	}
}

// The names of the refractive index's range in celestial's research
// data.
const (
	shortestName = "peck-reeder-shortest-um"
	longestName  = "peck-reeder-longest-um"
)

// The property and its anti-test: of any two wavelengths the refractive index
// holds, the shorter is scattered more, as Rayleigh's fourth power and the
// index both say; and any wavelength outside the index's range is refused, and
// gives no area.
func TestShorterLightIsScatteredMore(test *testing.T) {
	numbers := pickedNumbers(test).Drawn
	shortest, longest := research.Value(shortestName),
		research.Value(longestName)
	if !(longest-shortest > numbers.LeastStep && numbers.LeastStep >
		nothing && numbers.OutsideMicrons > numbers.LeastStep) {
		test.Fatalf("nothing to draw from: %+v", numbers)
	}
	rapid.Check(test, scattering{air: dry(test), shortest: shortest,
		longest: longest, drawn: numbers}.check)
}

// scattering draws wavelengths and holds the air's scattering of them
// to falling with wavelength.
type scattering struct {
	air               atmosphere.Composition
	shortest, longest float64
	drawn             drawn
}

// check draws two wavelengths inside the range and one outside it.
func (scatter scattering) check(test *rapid.T) {
	step := scatter.drawn.LeastStep
	short := rapid.Float64Range(scatter.shortest,
		scatter.longest-step).Draw(test, "short")
	long := rapid.Float64Range(short+step, scatter.longest).Draw(test,
		"long")
	shorter, err := scatter.air.CrossSection(microns(short))
	if err != nil {
		test.Fatalf("%g um: %v", short, err)
	}
	longer, err := scatter.air.CrossSection(microns(long))
	if err != nil {
		test.Fatalf("%g um: %v", long, err)
	}
	if !(shorter.InSquareMetres() > longer.InSquareMetres()) {
		test.Fatalf("%g um is scattered no more than %g um", short, long)
	}
	beyond := rapid.Float64Range(step, scatter.drawn.OutsideMicrons).Draw(
		test, "beyond")
	for _, each := range []float64{scatter.shortest - beyond,
		scatter.longest + beyond} {
		if area, err := scatter.air.CrossSection(microns(each)); area !=
			nil || err == nil {
			test.Fatalf("%g um gave %v, %v", each, area, err)
		}
	}
}

// Air of no co2, and air that is all co2, are made: the ends of a share
// are shares.
func TestAirAtTheEndsOfCO2IsMade(test *testing.T) {
	for _, ppm := range pickedNumbers(test).Edges.CO2 {
		if _, err := atmosphere.DryAir(ppm); err != nil {
			test.Errorf("air of %g ppm of co2 was refused: %v", ppm, err)
		}
	}
}

// The anti-test: air with less than no CO2, or more CO2 than air, is not made.
func TestAirOfNoSuchCO2IsRefused(test *testing.T) {
	for _, ppm := range append(pickedNumbers(test).NotCO2, math.NaN()) {
		if air, err := atmosphere.DryAir(ppm); err == nil || air != nil {
			test.Errorf("air of %g ppm of co2 was made", ppm)
		}
	}
}
