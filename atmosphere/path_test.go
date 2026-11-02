package atmosphere_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// Straight up from sea level, the air counted molecule by molecule is as deep
// to scattering as Bodhaine's hydrostatic sum makes it, to within the half
// percent the two ways of counting part by.
func TestTheAirStraightUpIsBodhainesSky(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	green := microns(numbers.GreenMicrons)
	counted, err := air.Depth(column(test, nothing, numbers.StraightUp),
		green)
	if err != nil {
		test.Fatal(err)
	}
	summed, err := air.DepthAbove(green,
		atmosphere.Pascals(numbers.Sea.Pascals),
		measure.Degrees(numbers.Latitude), measure.Metres(nothing))
	if err != nil {
		test.Fatal(err)
	}
	if !near(*counted, *summed, numbers.CountTol) {
		test.Errorf("straight up is %.5f deep counted, %.5f summed",
			*counted, *summed)
	}
}

// The anti-test: a way from higher up crosses less air. The same way, at
// the same angle, from a kilometre up holds fewer molecules than from the
// sea, so the start of a way is not left out of what it crosses.
func TestAWayFromHigherUpCrossesLessAir(test *testing.T) {
	numbers := pickedNumbers(test)
	if !(numbers.Anti.Higher > nothing) {
		test.Fatalf("no height to hold: %v", numbers.Anti.Higher)
	}
	low := column(test, nothing, numbers.Anti.Rise).Molecules()
	high := column(test, numbers.Anti.Higher, numbers.Anti.Rise).
		Molecules()
	if !(high < low) {
		test.Errorf("from %v m the way crosses %v molecules, from the sea "+
			"%v", numbers.Anti.Higher, high, low)
	}
}

// kastenYoung is the air mass at an apparent altitude by kasten and
// young, "revised optical air mass tables and approximation formula"
// (1989): the air along a way, as a multiple of the air straight up.
func kastenYoung(test *testing.T, degrees float64) float64 {
	fit := pickedNumbers(test).KastenYoung
	sine := math.Sin(measure.Degrees(degrees).InRadians())
	return whole / (sine + fit.Scale*math.Pow(degrees+fit.Offset,
		fit.Power))
}

// Above ten degrees, where the bending of light hardly matters, the air
// along a way up is kasten and young's air mass, to half a percent.
func TestTheAirAlongAWayIsKastenAndYoungsAirMass(test *testing.T) {
	numbers := pickedNumbers(test)
	straight := column(test, nothing, numbers.StraightUp).Molecules()
	for _, degrees := range numbers.AirMassDegrees {
		mass := column(test, nothing, degrees).Molecules() / straight
		want := kastenYoung(test, degrees)
		if !near(mass, want, numbers.CountTol) {
			test.Errorf("at %g deg the air is %.4f masses; kasten and "+
				"young say %.4f", degrees, mass, want)
		}
	}
}

// The anti-test: a way down meets the ground, and says so; a way up does
// not.
func TestAWayDownEndsOnTheGround(test *testing.T) {
	numbers := pickedNumbers(test)
	down := column(test, numbers.WayDown.Metres, numbers.WayDown.Degrees)
	up := column(test, numbers.WayUp.Metres, numbers.WayUp.Degrees)
	if !down.Grounded() || up.Grounded() {
		test.Error("a way down and a way up are told apart wrongly")
	}
}

// A way down from the sea is on the ground where it starts: it crosses
// no air, and does not go on through the earth to the sky beyond.
func TestAWayDownFromTheSeaIsOnTheGroundAtOnce(test *testing.T) {
	numbers := pickedNumbers(test)
	down := column(test, nothing, numbers.WayDown.Degrees)
	if !down.Grounded() || down.Molecules() != nothing {
		test.Errorf("a way down from the sea crosses %v molecules a "+
			"square metre, grounded %v", down.Molecules(), down.Grounded())
	}
}

// Straight down from a height crosses the air straight up from the sea,
// less the air straight up from that height: the same air, counted three
// ways. In the code as it is they agree to about a part in a hundred
// million at 1, 10 and 50 km; the tolerance picked is a part in a
// million, and an end of the sum counted wrong moves it by a thousandth.
func TestDownIsTheSeasAirLessTheAirAbove(test *testing.T) {
	numbers := pickedNumbers(test)
	upSea := column(test, nothing, numbers.StraightUp).Molecules()
	for _, metres := range numbers.Three.Metres {
		down := column(test, metres, -numbers.StraightUp).Molecules()
		upHere := column(test, metres, numbers.StraightUp).Molecules()
		if !near(down, upSea-upHere, numbers.Three.Tolerance) {
			test.Errorf("down from %v m is %v molecules a square metre; "+
				"the sea's less the air above, %v", metres, down,
				upSea-upHere)
		}
	}
}

// halfway is half of a thing. Defined.
const halfway = 0.5

// The anti-test: the ways down and up from a height split the air as the
// pressure there says. Pressure is the weight of the air overhead, so
// where less than half the sea's is left above, the way down crosses
// more air than the way up, and where more than half, less: from 1 km,
// less; from 10 and 50 km, more. Counts that were all nought, or the
// same either way, fail it.
func TestDownAndUpSplitTheAirAsThePressureSays(test *testing.T) {
	numbers := pickedNumbers(test)
	sea := standard(test, measure.Metres(nothing)).Pressure().InPascals()
	for _, metres := range numbers.Three.Metres {
		down := column(test, metres, -numbers.StraightUp).Molecules()
		up := column(test, metres, numbers.StraightUp).Molecules()
		left := standard(test, measure.Metres(metres)).Pressure().
			InPascals()
		if (down > up) != (left < halfway*sea) {
			test.Errorf("from %v m down crosses %v and up %v, with %v Pa "+
				"of the sea's %v left above", metres, down, up, left, sea)
		}
	}
}

// The anti-test: a level way from the sea is not on the ground. The
// earth curves away under it, so it goes on, through more air than a way
// straight up.
func TestALevelWayFromTheSeaGoesOn(test *testing.T) {
	numbers := pickedNumbers(test)
	level := column(test, nothing, nothing)
	up := column(test, nothing, numbers.StraightUp)
	if level.Grounded() || !(level.Molecules() > up.Molecules()) {
		test.Errorf("a level way from the sea crosses %v molecules a "+
			"square metre, grounded %v; straight up %v",
			level.Molecules(), level.Grounded(), up.Molecules())
	}
}

// What gets through is less at blue than at red: the Sun near the horizon is
// reddened by the air alone.
func TestTheLowSunIsReddened(test *testing.T) {
	numbers := pickedNumbers(test)
	air := dry(test)
	low := column(test, nothing, numbers.LowSun)
	blue, err := air.Through(low, microns(numbers.BlueMicrons))
	if err != nil {
		test.Fatal(err)
	}
	red, err := air.Through(low, microns(numbers.RedMicrons))
	if err != nil {
		test.Fatal(err)
	}
	if !(*blue < *red && *red < whole && *blue > nothing) {
		test.Errorf("through 2 deg of air blue keeps %.4f and red %.4f",
			*blue, *red)
	}
}
