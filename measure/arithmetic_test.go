package measure_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/measure"
)

// share draws a share from the test data's range.
func share(test *rapid.T, numbers picked, name string) float64 {
	return rapid.Float64Range(numbers.Shares.Least, numbers.Shares.Most).
		Draw(test, name)
}

// Any quantity scaled by a share is that share of itself, whatever the
// share: Times and Over undo each other.
func TestTimesAndOverUndoEachOther(test *testing.T) {
	rapid.Check(test, scaling{numbers: pickedNumbers(test)}.check)
}

// scaling checks that a quantity scaled by a drawn share is that share
// of itself.
type scaling struct{ numbers picked }

// check draws a share and scales the Sun's quantities by it.
func (scales scaling) check(test *rapid.T) {
	numbers, sun := scales.numbers, measure.Sun()
	scale := share(test, numbers, "share")
	checks := map[string]float64{
		"mass":   sun.Mass().Times(scale).Over(sun.Mass()),
		"length": sun.Radius().Times(scale).Over(sun.Radius()),
		"power":  sun.Luminosity().Times(scale).Over(sun.Luminosity()),
	}
	for name, got := range checks {
		if !near(got, scale, numbers.LinearTol) {
			test.Fatalf("a %s times %v is %v of itself", name, scale, got)
		}
	}
	energy := sun.Mass().Energy()
	if !near(energy.Times(scale).InMeV(), scale*energy.InMeV(),
		numbers.LinearTol) {
		test.Fatalf("an energy times %v is not %v of it", scale, scale)
	}
}

// noAmounts is test data with no amounts in it. Defined by counting.
const noAmounts = 0

// The anti-test: a mass less itself is no mass, and a time less itself
// no time, to the last bit, though each was made by a product inlined
// beside the subtraction: on arm64 the two fuse, and the product's
// rounding is left behind, unless Minus rounds each side first.
func TestAQuantityMadeInlineLessItselfIsNothing(test *testing.T) {
	numbers := pickedNumbers(test)
	if len(numbers.Fused) == noAmounts {
		test.Fatal("no amounts to subtract")
	}
	for _, amount := range numbers.Fused {
		if left := measure.Kilograms(amount).Minus(
			measure.Kilograms(amount)).InKilograms(); left != nothing {
			test.Errorf("%v kg less itself is %v kg", amount, left)
		}
		if left := measure.Seconds(amount).Minus(
			measure.Seconds(amount)).InSeconds(); left != nothing {
			test.Errorf("%v s less itself is %v s", amount, left)
		}
	}
}

// The anti-test: a time less itself is no time, and a mass less itself
// no mass, so Minus takes away and does not add.
func TestAQuantityLessItselfIsNothing(test *testing.T) {
	sun := measure.Sun()
	if left := sun.Mass().Minus(sun.Mass()); left.Positive() {
		test.Errorf("the sun less the sun weighs %v suns",
			left.InSolarMasses())
	}
	cosmos := measure.Cosmos()
	if left := cosmos.Minus(cosmos).InYears(); left != nothing {
		test.Errorf("the universe's age less itself is %v years", left)
	}
	if ratio := cosmos.Over(cosmos); ratio != pickedNumbers(test).One {
		test.Errorf("the universe's age over itself is %v", ratio)
	}
}

// Helium 4 is lighter than the four hydrogen atoms that fuse into it, by about
// seven parts in a thousand: the light the Sun shines by.
func TestHeliumIsLighterThanTheHydrogenItFusesFrom(test *testing.T) {
	numbers := pickedNumbers(test)
	four := measure.Hydrogen1().Times(numbers.Hydrogens)
	deficit := four.Minus(measure.Helium4()).Over(four)
	if !near(deficit, numbers.Deficit, numbers.DeficitTol) {
		test.Errorf("helium is lighter than four hydrogens by %v of them",
			deficit)
	}
}

// A power spends an energy in the energy over the power: ten joules at
// two watts take five seconds.
func TestTenJoulesAtTwoWattsTakeFiveSeconds(test *testing.T) {
	spent := pickedNumbers(test).Spent
	took := measure.Watts(spent.Watts).Spending(measure.Joules(spent.Joules))
	if seconds := took.InSeconds(); !near(seconds, spent.Seconds,
		pickedNumbers(test).TripTolerance) {
		test.Errorf("%v J at %v W takes %v s", spent.Joules, spent.Watts,
			seconds)
	}
}

// The anti-test: the Sun, spending its own mass's energy at its own luminosity,
// would shine for far longer than the universe is old. So Spending divides the
// energy by the power, and not the other way.
func TestSpendingDividesEnergyByPower(test *testing.T) {
	numbers := pickedNumbers(test)
	sun := measure.Sun()
	lifetime := sun.Luminosity().Spending(sun.Mass().Energy())
	if lifetime.InYears() < numbers.SunLifetime {
		test.Errorf("the sun would spend its mass in %v years",
			lifetime.InYears())
	}
	if lifetime.Over(measure.Cosmos()) <= numbers.One {
		test.Error("the sun spends its mass sooner than the universe ages")
	}
}
