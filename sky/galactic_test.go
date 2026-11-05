package sky_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
	"pgregory.net/rapid"
)

// The galactic frame by the Hipparcos definition. It is set out in ESA
// SP-1200 volume 1 section 1.5.3. Each is defined, in degrees.
const (
	// northGalacticAscension is the right ascension of the north galactic
	// pole. It's defined.
	northGalacticAscension = 192.85948
	// northGalacticDeclination is the declination of the north galactic
	// pole. It's defined.
	northGalacticDeclination = 27.12825
	// northCelestialLongitude is the galactic longitude of the north
	// celestial pole. It's defined.
	northCelestialLongitude = 122.93192
	// quarterTurn is a quarter turn. It's defined.
	quarterTurn = 90
	// fullTurn is a turn. It's defined.
	fullTurn = 360
	// onTheEquator is the declination or latitude of an equator. It's
	// defined.
	onTheEquator = 0
)

// stopping is a test or a property's draw that can be stopped.
type stopping interface {
	Helper()
	Fatal(args ...any)
}

// equatorial is the direction at a right ascension and declination in
// degrees, or the test stops.
func equatorial(test stopping, east, north float64) sky.Equatorial {
	test.Helper()
	seen, err := sky.NewEquatorial(measure.Degrees(east),
		measure.Degrees(north))
	if err != nil {
		test.Fatal(err)
	}
	return *seen
}

// galactic is the direction at a galactic longitude and latitude in
// degrees, or the test stops.
func galactic(test stopping, along, up float64) sky.Galactic {
	test.Helper()
	seen, err := sky.NewGalactic(measure.Degrees(along),
		measure.Degrees(up))
	if err != nil {
		test.Fatal(err)
	}
	return *seen
}

// apartBy is how far two longitudes differ the short way round, in
// degrees.
func apartBy(one, other float64) float64 {
	return math.Abs(math.Remainder(one-other, fullTurn))
}

// The points the definition fixes land where it puts them, both ways.
// The ascending node is where the galactic plane rises through the
// celestial equator, a quarter turn from each pole.
func TestGalacticLandsWhereTheDefinitionSays(test *testing.T) {
	within := clockNumbers(test).PoleTolerance
	pole := equatorial(test, northGalacticAscension,
		northGalacticDeclination).Galactic()
	if !(math.Abs(pole.Latitude().InDegrees()-quarterTurn) <= within) {
		test.Errorf("the galactic pole at latitude %v", pole.Latitude())
	}
	celestial := equatorial(test, onTheEquator, quarterTurn).Galactic()
	node := equatorial(test, northGalacticAscension+quarterTurn,
		onTheEquator).Galactic()
	for why, each := range map[string]struct {
		seen      sky.Galactic
		along, up float64
	}{
		"the celestial pole": {celestial, northCelestialLongitude,
			northGalacticDeclination},
		"the ascending node": {node, northCelestialLongitude - quarterTurn,
			onTheEquator},
	} {
		if !(apartBy(each.seen.Longitude().InDegrees(), each.along) <=
			within) || !(math.Abs(each.seen.Latitude().InDegrees()-
			each.up) <= within) {
			test.Errorf("%s: %v, %v", why, each.seen.Longitude(),
				each.seen.Latitude())
		}
	}
	back := galactic(test, northCelestialLongitude-quarterTurn,
		onTheEquator).Equatorial()
	if !(apartBy(back.Ascension().InDegrees(),
		northGalacticAscension+quarterTurn) <= within) ||
		!(math.Abs(back.Declination().InDegrees()) <= within) {
		test.Errorf("the node, back: %v", back)
	}
}

// comesBack is the property of TestEveryDirectionComesBack, for one draw.
func comesBack(test *rapid.T) {
	east := rapid.Float64Range(onTheEquator, fullTurn).Draw(test, "east")
	north := rapid.Float64Range(-quarterTurn, quarterTurn).
		Draw(test, "north")
	seen := equatorial(test, math.Mod(east, fullTurn), north)
	back := seen.Galactic().Equatorial()
	nearPole := math.Cos(seen.Declination().InRadians())
	off := apartBy(back.Ascension().InDegrees(), east) * nearPole
	if !(off <= sameWithin) || !(math.Abs(back.Declination().InDegrees()-
		north) <= sameWithin) {
		test.Fatalf("%v came back %v", seen, back)
	}
}

// sameWithin is the pole tolerance for the property, which reads it
// once. TestEveryDirectionComesBack fills it.
var sameWithin float64

// Property: every direction goes to the galactic frame and comes back to
// itself. Near a pole, right ascension is weighed by how short a degree of
// it is there.
func TestEveryDirectionComesBack(test *testing.T) {
	sameWithin = clockNumbers(test).PoleTolerance
	rapid.Check(test, comesBack)
}

// Anti-tests: a galactic latitude past a pole and a longitude more than a
// turn either way are refused.
func TestNewGalacticRefusesWhatIsNotOnTheSky(test *testing.T) {
	numbers := clockNumbers(test)
	for _, up := range numbers.NotDeclinations {
		if _, err := sky.NewGalactic(measure.Degrees(onTheEquator),
			measure.Degrees(up)); err == nil {
			test.Errorf("latitude %v", up)
		}
	}
	for _, along := range numbers.NotAscensions {
		if _, err := sky.NewGalactic(measure.Degrees(along),
			measure.Degrees(onTheEquator)); err == nil {
			test.Errorf("longitude %v", along)
		}
	}
}

// Rounding can push the galactic latitude's sine past one a hair from the
// pole. The latitude there is the pole's, not NaN.
func TestASinePastOneIsThePole(test *testing.T) {
	numbers := clockNumbers(test)
	seen := equatorial(test, numbers.PastOne.Ascension,
		numbers.PastOne.Declination).Galactic()
	if !(math.Abs(seen.Latitude().InDegrees()-quarterTurn) <=
		numbers.PoleTolerance) {
		test.Errorf("latitude %v", seen.Latitude())
	}
}
