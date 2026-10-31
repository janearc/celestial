package atmosphere_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
	"pgregory.net/rapid"
)

// squareCentimetre is a square centimetre in square metres. Defined by
// the si's centi.
var squareCentimetre = si.Centi(whole) * si.Centi(whole)

// A molecule of dry air scatters what bodhaine's table says, to its five
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

// The air over sea level at 45 degrees is as deep, to scattering, as
// bodhaine's table says, to its five figures.
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
		if !near(depth, each.Depth, numbers.BodhaineTol) {
			test.Errorf("at %g um the sky is %.6f deep; bodhaine says "+
				"%.6f", each.Microns, depth, each.Depth)
		}
	}
}

// The anti-test: the sky's depth is the air's above, not a fixed figure.
// Half the sea's pressure, at the same height and latitude, is half the
// depth, to a few roundings, as the depth is linear in the pressure
// there; a depth that left the pressure out would pass bodhaine's table,
// all at the sea's, and fail here.
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
	if !near(part, share*full, numbers.Anti.DepthTol) {
		test.Errorf("%v of the air is %v deep, not %v of %v", share, part,
			share, full)
	}
}

// The anti-test: the sky's depth is not the same at every latitude. The
// earth pulls harder at the poles, so the same air over the sea is
// deeper to scattering at the equator than at a pole. A depth whose
// gravity left the latitude out, or turned its sign, would pass
// bodhaine's table, all at 45 degrees where list's latitude term is
// nought, and fail here.
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
	return depth
}

// The anti-test: co2 scatters too. Air with more of it than bodhaine's
// scatters more, at every wavelength of their table; a cross-section that
// left the co2 out would pass their table, all at 360 ppm, and fail here.
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

// The anti-test: peck and reeder's refractive index holds from 0.23 to
// 1.69 microns; outside it, and at no number, the air's scattering is
// not known here, and every way of asking for it is refused.
func TestWavelengthsOutsideTheIndexAreRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	along := column(test, nothing, numbers.StraightUp)
	sea := atmosphere.Pascals(numbers.Sea.Pascals)
	for _, each := range append(numbers.OutsideMicrons, math.NaN()) {
		wavelength := microns(each)
		if area, err := air.CrossSection(wavelength); !errors.Is(err,
			atmosphere.ErrWavelength()) || area != nil {
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

// The property and its anti-test: of any two wavelengths the refractive
// index holds, the shorter is scattered more, as rayleigh's fourth power
// and the index both say; and any wavelength outside the index's range
// is refused, and gives no area.
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
			nil || !errors.Is(err, atmosphere.ErrWavelength()) {
			test.Fatalf("%g um gave %v, %v", each, area, err)
		}
	}
}

// The anti-test: air with less than no co2, or more co2 than air, is
// not made.
func TestAirOfNoSuchCO2IsRefused(test *testing.T) {
	for _, ppm := range append(pickedNumbers(test).NotCO2, math.NaN()) {
		if air, err := atmosphere.DryAir(ppm); err == nil || air != nil {
			test.Errorf("air of %g ppm of co2 was made", ppm)
		}
	}
}
