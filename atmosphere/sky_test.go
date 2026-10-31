package atmosphere_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// skyLight is the sky's light at wavelengths in nanometres, for each
// watt a square metre of sunlight, or the test stops.
func skyLight(test *testing.T, metres float64, view, sun atmosphere.Way,
	nm ...float64) []float64 {
	test.Helper()
	var wavelengths []measure.Length
	for _, each := range nm {
		wavelengths = append(wavelengths, nanometres(each))
	}
	light, err := dry(test).Skylight(measure.Metres(metres), view, sun,
		wavelengths)
	if err != nil {
		test.Fatal(err)
	}
	return light
}

// The parts of rayleigh's phase function looking straight on, 3 over
// 8 pi, each defined by it.
const (
	straightOnScale = 3.0 // defined by rayleigh's phase function
	straightOnOver  = 8.0 // defined by rayleigh's phase function
)

// In thin air, high up, the sky straight overhead with the sun there too
// is the sunlight times rayleigh's phase function looking straight on,
// times the air's depth overhead: single scattering's own closed form,
// to a percent.
func TestThinAirIsSingleScatteringsClosedForm(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	got := skyLight(test, numbers.ThinAir, up, up, numbers.GreenNM)[first]
	depth, err := dry(test).Depth(column(test, numbers.ThinAir,
		numbers.StraightUp), nanometres(numbers.GreenNM))
	if err != nil {
		test.Fatal(err)
	}
	want := straightOnScale / (straightOnOver * math.Pi) * depth
	if !near(got, want, numbers.ClosedFormTol) {
		test.Errorf("thin air overhead is %.6g; closed form %.6g", got,
			want)
	}
}

// The sky is blue: overhead, with the sun at 45 degrees, it sends more
// than three times the blue it sends of red, where the sunlight that
// lights it, come down through the air, carries more red than blue.
func TestTheSkyIsBlue(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	sun := way(test, nothing, whole, whole)
	light := skyLight(test, nothing, up, sun, numbers.BlueNM,
		numbers.RedNM)
	if light[first] < numbers.BlueOverRed*light[second] {
		test.Errorf("overhead the sky sends %.4g of blue and %.4g of red",
			light[first], light[second])
	}
}

// The anti-test: once the sun is ten degrees down, the earth's shadow
// covers the air overhead, and the sky there is a hundredth or less of
// what it is with the sun ten degrees up.
func TestTheSkyDarkensAfterSunset(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	rise := math.Sin(numbers.SunUpRadians)
	high := skyLight(test, nothing, up, way(test, nothing, rise, whole),
		numbers.GreenNM)[first]
	low := skyLight(test, nothing, up, way(test, nothing, -rise, whole),
		numbers.GreenNM)[first]
	if low > high*numbers.AfterSunset {
		test.Errorf("overhead the sky is %.4g with the sun up, %.4g down",
			high, low)
	}
}

// The anti-test: a way that points nowhere, or whose parts are not
// numbers, is refused, and gives no way.
func TestAWayThatPointsNowhereIsRefused(test *testing.T) {
	for _, parts := range [][]float64{{nothing, nothing, nothing},
		{math.NaN(), whole, nothing}, {math.Inf(upward), nothing, nothing}} {
		made, err := atmosphere.NewWay(parts[first], parts[second],
			parts[len(parts)-int(whole)])
		if !errors.Is(err, atmosphere.ErrWay()) || made != nil {
			test.Errorf("a way of %v gave %v, %v", parts, made, err)
		}
	}
}

// The anti-test: the sky is not seen from outside the standard, nor at a
// wavelength the air's index does not hold, and gives no light.
func TestTheSkyIsRefusedWhereItIsNotKnown(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	air := dry(test)
	for _, metres := range numbers.OutsideMetres {
		light, err := air.Skylight(measure.Metres(metres), up, up,
			[]measure.Length{nanometres(numbers.GreenNM)})
		if !errors.Is(err, atmosphere.ErrHeight()) || light != nil {
			test.Errorf("the sky from %g m is %v, %v", metres, light, err)
		}
	}
	light, err := air.Skylight(measure.Metres(nothing), up, up,
		[]measure.Length{microns(numbers.OutsideMicrons[first])})
	if !errors.Is(err, atmosphere.ErrWavelength()) || light != nil {
		test.Errorf("the sky at %g um is %v, %v",
			numbers.OutsideMicrons[first], light, err)
	}
}
