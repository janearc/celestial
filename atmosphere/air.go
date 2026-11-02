package atmosphere

import "github.com/janearc/celestial/measure"

// Air stores local temperature, pressure, and mass density.
type Air struct {
	temperature measure.Temperature
	pressure    Pressure
	density     Density
}

// newAir derives density from temperature and pressure through the ideal-gas
// law using the standard atmosphere’s molar mass.
func newAir(kelvin, pascals float64) Air {
	return Air{temperature: measure.Kelvin(kelvin),
		pressure: Pressure{pascals: pascals},
		density: Density{kilograms: pascals * molarMass /
			(gasConstant * kelvin)}}
}

// Temperature returns local air temperature.
func (air Air) Temperature() measure.Temperature { return air.temperature }

// Pressure returns local air pressure.
func (air Air) Pressure() Pressure { return air.pressure }

// Density returns local mass density.
func (air Air) Density() Density { return air.density }

// Pressure represents pressure.
type Pressure struct{ pascals float64 }

// Pascals is a pressure in pascals.
func Pascals(amount float64) Pressure { return Pressure{pascals: amount} }

// InPascals is the pressure in pascals.
func (pressure Pressure) InPascals() float64 { return pressure.pascals }

// Density represents mass per volume.
type Density struct{ kilograms float64 }

// InKilogramsPerCubicMetre returns density in kg/m³.
func (density Density) InKilogramsPerCubicMetre() float64 {
	return density.kilograms
}
