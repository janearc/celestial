package atmosphere_test

import (
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

// The parts of Rayleigh's phase function looking straight on, 3 over 8 pi, each
// defined by it.
const (
	straightOnScale = 3.0 // defined by rayleigh's phase function
	straightOnOver  = 8.0 // defined by rayleigh's phase function
)

// In thin air, high up, the sky straight overhead with the Sun there too is the
// sunlight times Rayleigh's phase function looking straight on, times the air's
// depth overhead: single scattering's own closed form, to a percent.
func TestThinAirIsSingleScatteringsClosedForm(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	got := skyLight(test, numbers.ThinAir, up, up, numbers.GreenNM)[first]
	depth, err := dry(test).Depth(column(test, numbers.ThinAir,
		numbers.StraightUp), nanometres(numbers.GreenNM))
	if err != nil {
		test.Fatal(err)
	}
	want := straightOnScale / (straightOnOver * math.Pi) * *depth
	if !near(got, want, numbers.ClosedFormTol) {
		test.Errorf("thin air overhead is %.6g; closed form %.6g", got,
			want)
	}
}

// The parts of rayleigh's phase function at any angle, 3 over 16 pi
// times one and the cosine squared, and that cosine squared at 45
// degrees, a half; each defined by the function and the angle.
const (
	lobeScale     = 3.0  // defined by rayleigh's phase function
	lobeOver      = 16.0 // defined by rayleigh's phase function
	halfCosSquare = 0.5  // defined: the cosine of 45 degrees, squared
)

// depthUp is the depth to green light straight up from a height, or the
// test stops.
func depthUp(test *testing.T, metres float64) float64 {
	test.Helper()
	numbers := pickedNumbers(test)
	depth, err := dry(test).Depth(column(test, metres,
		numbers.StraightUp), nanometres(numbers.GreenNM))
	if err != nil {
		test.Fatal(err)
	}
	return *depth
}

// In thin air the sky straight up, with the sun at 45 degrees, is the
// sunlight times rayleigh's phase function at that angle, times the
// depth overhead: the air on the way takes almost none.
func TestThinAirSeesThePhaseAtTheSunsAngle(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	sun := way(test, nothing, whole, whole)
	got := skyLight(test, numbers.ThinAir, up, sun, numbers.GreenNM)[first]
	want := lobeScale / (lobeOver * math.Pi) * (whole + halfCosSquare) *
		depthUp(test, numbers.ThinAir)
	if !near(got, want, numbers.ClosedFormTol) {
		test.Errorf("thin air at 45 degrees is %.6g; closed form %.6g",
			got, want)
	}
}

// The anti-test: at the sea the air on the way takes its share, of the
// sunlight coming down at 45 degrees and of the sky's light going up to
// the eye. In flat layers single scattering keeps e^-t (1 - e^-(r-1)t)
// over (r-1)t of the light with none taken, t the depth straight up and
// r the root of two, the secant of 45 degrees.
//
// Take away either loss and the sky misses that by several percent; the
// earth's curve, which flat layers leave out, moves it by far less than
// the tolerance at 45 degrees.
func TestAtTheSeaTheAirTakesItsShare(test *testing.T) {
	numbers := pickedNumbers(test)
	up := way(test, nothing, whole, nothing)
	sun := way(test, nothing, whole, whole)
	got := skyLight(test, nothing, up, sun, numbers.GreenNM)[first]
	depth := depthUp(test, nothing)
	untaken := lobeScale / (lobeOver * math.Pi) * (whole + halfCosSquare) *
		depth
	slant := math.Sqrt2 - whole
	kept := math.Exp(-depth) * (whole - math.Exp(-slant*depth)) /
		(slant * depth)
	if !near(got, kept*untaken, numbers.ClosedFormTol) {
		test.Errorf("at the sea the sky is %.6g; flat layers keep %.6g",
			got, kept*untaken)
	}
}

// The air is the same all round the up: turn the look and the sun
// together a quarter turn about it, x to z, and the sky is the same, to
// a few roundings. The look here leans toward x, and turned toward z, so
// every part of the way to the sun counts once each way.
func TestTheSkyIsTheSameAllRound(test *testing.T) {
	numbers := pickedNumbers(test)
	turn := numbers.AllRound
	before := skyLight(test, nothing, way(test, turn.Look.AlongX,
		turn.Look.AlongY, turn.Look.AlongZ), way(test, turn.Sun.AlongX,
		turn.Sun.AlongY, turn.Sun.AlongZ), numbers.GreenNM)[first]
	after := skyLight(test, nothing, way(test, -turn.Look.AlongZ,
		turn.Look.AlongY, turn.Look.AlongX), way(test, -turn.Sun.AlongZ,
		turn.Sun.AlongY, turn.Sun.AlongX), numbers.GreenNM)[first]
	if !near(after, before, turn.Tolerance) {
		test.Errorf("turned a quarter about up the sky is %v, not %v",
			after, before)
	}
}

// The anti-test: up is not like the other ways. Turn the same look and
// sun a quarter about x instead, y to z, and the look lies level: the
// sky there is another.
func TestTheSkyIsNotTheSameTurnedAboutAnotherWay(test *testing.T) {
	numbers := pickedNumbers(test)
	turn := numbers.AllRound
	before := skyLight(test, nothing, way(test, turn.Look.AlongX,
		turn.Look.AlongY, turn.Look.AlongZ), way(test, turn.Sun.AlongX,
		turn.Sun.AlongY, turn.Sun.AlongZ), numbers.GreenNM)[first]
	tipped := skyLight(test, nothing, way(test, turn.Look.AlongX,
		-turn.Look.AlongZ, turn.Look.AlongY), way(test, turn.Sun.AlongX,
		-turn.Sun.AlongZ, turn.Sun.AlongY), numbers.GreenNM)[first]
	if near(tipped, before, turn.Tolerance) {
		test.Errorf("turned a quarter about x the sky is still %v", before)
	}
}

// The sky is blue: overhead, with the Sun at 45 degrees, it sends more than
// three times the blue it sends of red, where the sunlight that lights it, come
// down through the air, carries more red than blue.
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

// The anti-test: once the Sun is ten degrees down, the Earth's shadow covers
// the air overhead, and the sky there is a hundredth or less of what it is with
// the Sun ten degrees up.
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

// A way whose parts are too large or too small to square is the way the
// same parts point at an everyday size: looking straight up, with the
// sun at 45 degrees, it sees the very same sky.
func TestAWayBeyondSquaringSeesTheSameSky(test *testing.T) {
	numbers := pickedNumbers(test)
	beyond := numbers.Beyond
	up := way(test, beyond.Up.AlongX, beyond.Up.AlongY, beyond.Up.AlongZ)
	sun := way(test, beyond.Sun.AlongX, beyond.Sun.AlongY,
		beyond.Sun.AlongZ)
	got := skyLight(test, nothing, up, sun, numbers.GreenNM)[first]
	want := skyLight(test, nothing, way(test, nothing, whole, nothing),
		way(test, nothing, whole, whole), numbers.GreenNM)[first]
	if got != want {
		test.Errorf("up of %v sees %v; up sees %v", beyond.Up, got, want)
	}
}

// The anti-test: a way beyond squaring still points where its parts do.
// The same large parts laid level see another sky than straight up.
func TestAWayBeyondSquaringPointsWhereItsPartsDo(test *testing.T) {
	numbers := pickedNumbers(test)
	beyond := numbers.Beyond
	sun := way(test, beyond.Sun.AlongX, beyond.Sun.AlongY,
		beyond.Sun.AlongZ)
	level := way(test, beyond.Level.AlongX, beyond.Level.AlongY,
		beyond.Level.AlongZ)
	up := way(test, beyond.Up.AlongX, beyond.Up.AlongY, beyond.Up.AlongZ)
	across := skyLight(test, nothing, level, sun, numbers.GreenNM)[first]
	overhead := skyLight(test, nothing, up, sun, numbers.GreenNM)[first]
	if across == overhead {
		test.Errorf("level and up both see %v", across)
	}
}

// The anti-test: a way that points nowhere, or whose parts are not
// numbers, is refused, and gives no way.
func TestAWayThatPointsNowhereIsRefused(test *testing.T) {
	for _, parts := range [][]float64{{nothing, nothing, nothing},
		{math.NaN(), whole, nothing}, {math.Inf(upward), nothing, nothing}} {
		made, err := atmosphere.NewWay(parts[first], parts[second],
			parts[len(parts)-int(whole)])
		if err == nil || made != nil {
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
		if err == nil || light != nil {
			test.Errorf("the sky from %g m is %v, %v", metres, light, err)
		}
	}
	light, err := air.Skylight(measure.Metres(nothing), up, up,
		[]measure.Length{microns(numbers.OutsideMicrons[first])})
	if err == nil || light != nil {
		test.Errorf("the sky at %g um is %v, %v",
			numbers.OutsideMicrons[first], light, err)
	}
}
