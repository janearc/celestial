package atmosphere_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"pgregory.net/rapid"
)

// geometric is the height above sea level of a geopotential height, by
// the 1976 standard's earth radius.
func geometric(test *testing.T, geopotential float64) measure.Length {
	radius := pickedNumbers(test).EarthRadius
	return measure.Metres(radius * geopotential / (radius - geopotential))
}

// At sea level the 1976 standard atmosphere is 288.15 K, 101325 Pa and
// 1.2250 kg per cubic metre: us standard atmosphere, 1976.
func TestSeaLevelIsTheStandardsOwn(test *testing.T) {
	sea := pickedNumbers(test).Sea
	air := standard(test, measure.Metres(nothing))
	kelvin, pascals := air.Temperature().InKelvin(),
		air.Pressure().InPascals()
	density := air.Density().InKilogramsPerCubicMetre()
	if !(math.Abs(kelvin-sea.Kelvin) <= sea.KelvinTol) ||
		!(math.Abs(pascals-sea.Pascals) <= sea.PascalTol) ||
		!(math.Abs(density-sea.Density) <= sea.DensityTol) {
		test.Errorf("sea level is %g K, %g Pa, %g kg/m3", kelvin, pascals,
			density)
	}
}

// The pressure and temperature at the base of every layer, at its
// geopotential height, are the 1976 standard's own figures.
func TestEveryLayerStartsWhereTheStandardSays(test *testing.T) {
	numbers := pickedNumbers(test)
	for _, at := range numbers.Bases {
		air := standard(test, geometric(test, at.Height))
		kelvin, pascals := air.Temperature().InKelvin(),
			air.Pressure().InPascals()
		if !(math.Abs(kelvin-at.Kelvin) <= numbers.BaseKelvinTol) ||
			!near(pascals, at.Pascals, numbers.BasePascalTol) {
			test.Errorf("at %g m: %g K and %g Pa; the standard says %g "+
				"and %g", at.Height, kelvin, pascals, at.Kelvin, at.Pascals)
		}
	}
}

// A u-2 at 70,000 feet has more than nineteen twentieths of the air
// below it.
func TestAU2FliesOverMostOfTheAir(test *testing.T) {
	numbers := pickedNumbers(test)
	high := standard(test, measure.Metres(numbers.U2Feet*
		numbers.MetresAFoot))
	share := high.Pressure().InPascals() / numbers.Sea.Pascals
	if share >= numbers.U2Above {
		test.Errorf("at 70,000 ft %.4f of the air is still above", share)
	}
}

// The anti-test: the standard reaches from sea level to 86 km. Outside
// it there is no answer, a height that is no number is none, and no
// air is given.
func TestHeightsOutsideTheStandardAreRefused(test *testing.T) {
	heights := append(pickedNumbers(test).OutsideMetres, math.NaN())
	for _, metres := range heights {
		air, err := atmosphere.Standard(measure.Metres(metres))
		if !errors.Is(err, atmosphere.ErrHeight()) || air != nil {
			test.Errorf("%g m gave %v, %v", metres, air, err)
		}
		column, err := atmosphere.Along(measure.Metres(metres),
			measure.Degrees(nothing))
		if !errors.Is(err, atmosphere.ErrHeight()) || column != nil {
			test.Errorf("a way from %g m gave %v, %v", metres, column, err)
		}
	}
}

// The property and its anti-test: of any two heights the standard holds,
// the higher has less pressure and less density, though its temperature
// may rise; and any height below the sea or above the standard's top is
// refused, and gives no air.
func TestTheAirThinsAllTheWayUp(test *testing.T) {
	numbers := pickedNumbers(test).Drawn
	if !(numbers.Top > numbers.LeastRise && numbers.LeastRise > nothing &&
		numbers.OutsideMetres > numbers.LeastRise) {
		test.Fatalf("nothing to draw from: %+v", numbers)
	}
	rapid.Check(test, thinning{drawn: numbers}.check)
}

// thinning draws heights and holds the air at them to thinning upward.
type thinning struct{ drawn drawn }

// check draws two heights inside the standard and one outside it.
func (thin thinning) check(test *rapid.T) {
	drawn := thin.drawn
	low := rapid.Float64Range(nothing, drawn.Top-drawn.LeastRise).Draw(test,
		"low")
	high := rapid.Float64Range(low+drawn.LeastRise, drawn.Top).Draw(test,
		"high")
	below, err := atmosphere.Standard(measure.Metres(low))
	if err != nil {
		test.Fatalf("%g m: %v", low, err)
	}
	above, err := atmosphere.Standard(measure.Metres(high))
	if err != nil {
		test.Fatalf("%g m: %v", high, err)
	}
	if !(above.Pressure().InPascals() < below.Pressure().InPascals()) {
		test.Fatalf("the pressure at %g m is not below %g m's", high, low)
	}
	if !(above.Density().InKilogramsPerCubicMetre() <
		below.Density().InKilogramsPerCubicMetre()) {
		test.Fatalf("the density at %g m is not below %g m's", high, low)
	}
	beyond := rapid.Float64Range(drawn.LeastRise,
		drawn.OutsideMetres).Draw(test, "beyond")
	for _, metres := range []float64{-beyond, drawn.Top + beyond} {
		if air, err := atmosphere.Standard(measure.Metres(metres)); air !=
			nil || !errors.Is(err, atmosphere.ErrHeight()) {
			test.Fatalf("%g m gave %v, %v", metres, air, err)
		}
	}
}
