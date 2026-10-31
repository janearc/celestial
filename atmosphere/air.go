package atmosphere

import "github.com/janearc/celestial/measure"

// Air is the air at one place: how warm it is, how hard it presses and
// how much of it there is.
type Air struct {
	temperature measure.Temperature
	pressure    Pressure
	density     Density
}

// newAir is air of a temperature and pressure, its density by the
// ideal gas law at the standard's molar mass.
func newAir(kelvin, pascals float64) Air {
	return Air{temperature: measure.Kelvin(kelvin),
		pressure: Pressure{pascals: pascals},
		density: Density{kilograms: pascals * molarMass /
			(gasConstant * kelvin)}}
}

// Temperature is how warm the air is.
func (air Air) Temperature() measure.Temperature { return air.temperature }

// Pressure is how hard the air presses.
func (air Air) Pressure() Pressure { return air.pressure }

// Density is how much air there is in a volume.
func (air Air) Density() Density { return air.density }

// Pressure is a pressure.
type Pressure struct{ pascals float64 }

// Pascals is a pressure in pascals.
func Pascals(amount float64) Pressure { return Pressure{pascals: amount} }

// InPascals is the pressure in pascals.
func (pressure Pressure) InPascals() float64 { return pressure.pascals }

// Density is mass in a volume.
type Density struct{ kilograms float64 }

// InKilogramsPerCubicMetre is the density in kilograms a cubic metre.
func (density Density) InKilogramsPerCubicMetre() float64 {
	return density.kilograms
}
