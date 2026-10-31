package atmosphere_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// Straight up from sea level, the air counted molecule by molecule is as
// deep to scattering as bodhaine's hydrostatic sum makes it, to within
// the half percent the two ways of counting part by.
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
	if !near(counted, summed, numbers.CountTol) {
		test.Errorf("straight up is %.5f deep counted, %.5f summed",
			counted, summed)
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
// (1989): how many times the air straight up a way through the air is.
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

// What gets through is less at blue than at red: the sun near the
// horizon is reddened by the air alone.
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
	if !(blue < red && red < whole && blue > nothing) {
		test.Errorf("through 2 deg of air blue keeps %.4f and red %.4f",
			blue, red)
	}
}
