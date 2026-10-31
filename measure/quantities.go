package measure

import (
	"math"

	"github.com/janearc/celestial/research"
)

// Each quantity holds its value in natural units and nothing else, so
// the only way to make one is from a unit, and the only way to see it
// is in one.

// The numbers the checks below compare with, each defined.
const (
	// nothing is no amount at all. Defined by the scale.
	nothing = 0.0
	// eitherSign asks math.IsInf about both infinities. Defined by math.
	eitherSign = 0
)

// Positive is whether the mass is above nothing and finite.
func (mass Mass) Positive() bool {
	return mass.natural > nothing && !math.IsInf(mass.natural, eitherSign)
}

// Energy is an energy, in MeV
type Energy struct{ natural float64 }

// Mass is a mass, held as the energy it is at rest
type Mass struct{ natural float64 }

// Temperature is a temperature, held as the energy k T
type Temperature struct{ natural float64 }

// Length is a length, held in one over MeV
type Length struct{ natural float64 }

// Duration is a time, held in one over MeV
type Duration struct{ natural float64 }

// Power is energy given off over time, held in MeV squared
type Power struct{ natural float64 }

// Area is an area, held in one over MeV squared
type Area struct{ natural float64 }

// MeV is an energy in MeV
func MeV(amount float64) Energy { return Energy{amount} }

// Electronvolts is an energy in eV
func Electronvolts(amount float64) Energy {
	return Energy{amount * electronvolt}
}

// Joules is an energy in J
func Joules(amount float64) Energy { return Energy{amount * joule} }

// InMeV is the energy in MeV
func (energy Energy) InMeV() float64 { return energy.natural }

// InElectronvolts is the energy in eV
func (energy Energy) InElectronvolts() float64 {
	return energy.natural / electronvolt
}

// InJoules is the energy in J
func (energy Energy) InJoules() float64 { return energy.natural / joule }

// Times is the energy scaled by a share
func (energy Energy) Times(share float64) Energy {
	return Energy{energy.natural * share}
}

// Kilograms is a mass in kg
func Kilograms(amount float64) Mass { return Mass{amount * kilogram} }

// SolarMasses is a mass in the sun's masses
func SolarMasses(amount float64) Mass {
	return Mass{amount * Sun().Mass().natural}
}

// Daltons is a mass in daltons, a twelfth of a carbon 12 atom
func Daltons(amount float64) Mass {
	return Mass{amount * research.Value(daltonMeV)}
}

// InKilograms is the mass in kg
func (mass Mass) InKilograms() float64 { return mass.natural / kilogram }

// InSolarMasses is the mass in the sun's masses
func (mass Mass) InSolarMasses() float64 {
	return mass.natural / Sun().Mass().natural
}

// InMeV is the mass in MeV, which is its energy at rest
func (mass Mass) InMeV() float64 { return mass.natural }

// Energy is the energy the mass is, at rest
func (mass Mass) Energy() Energy { return Energy(mass) }

// Times is the mass scaled by a share
func (mass Mass) Times(share float64) Mass {
	return Mass{mass.natural * share}
}

// Minus is the mass less another. Each side is rounded first, as an
// explicit float64 conversion does: inlined, a product that made a mass
// could fuse with the subtraction on arm64, and a mass less itself would
// leave the product's rounding behind instead of nothing.
func (mass Mass) Minus(less Mass) Mass {
	return Mass{float64(mass.natural) - float64(less.natural)}
}

// Over is how many of another mass this one is
func (mass Mass) Over(other Mass) float64 {
	return mass.natural / other.natural
}

// Kelvin is a temperature in K
func Kelvin(amount float64) Temperature {
	return Temperature{amount * kelvin}
}

// InKelvin is the temperature in K
func (heat Temperature) InKelvin() float64 { return heat.natural / kelvin }

// InMeV is the temperature as the energy k T, in MeV
func (heat Temperature) InMeV() float64 { return heat.natural }

// Metres is a length in m
func Metres(amount float64) Length { return Length{amount * metre} }

// SolarRadii is a length in the sun's radius
func SolarRadii(amount float64) Length {
	return Length{amount * Sun().Radius().natural}
}

// InMetres is the length in m
func (length Length) InMetres() float64 { return length.natural / metre }

// InSolarRadii is the length in the sun's radius
func (length Length) InSolarRadii() float64 {
	return length.natural / Sun().Radius().natural
}

// Times is the length scaled by a share
func (length Length) Times(share float64) Length {
	return Length{length.natural * share}
}

// Over is how many of another length this one is
func (length Length) Over(other Length) float64 {
	return length.natural / other.natural
}

// Seconds is a time in s
func Seconds(amount float64) Duration { return Duration{amount * second} }

// Years is a time in julian years
func Years(amount float64) Duration { return Duration{amount * year} }

// InSeconds is the time in s
func (span Duration) InSeconds() float64 { return span.natural / second }

// InYears is the time in julian years
func (span Duration) InYears() float64 { return span.natural / year }

// Minus is the time less another. Each side is rounded first, as Mass's
// Minus says why.
func (span Duration) Minus(less Duration) Duration {
	return Duration{float64(span.natural) - float64(less.natural)}
}

// Over is how many of another time this one is
func (span Duration) Over(other Duration) float64 {
	return span.natural / other.natural
}

// Watts is a power in W
func Watts(amount float64) Power { return Power{amount * watt} }

// SolarLuminosities is a power in the sun's luminosity
func SolarLuminosities(amount float64) Power {
	return Power{amount * Sun().Luminosity().natural}
}

// InWatts is the power in W
func (power Power) InWatts() float64 { return power.natural / watt }

// InSolarLuminosities is the power in the sun's luminosity
func (power Power) InSolarLuminosities() float64 {
	return power.natural / Sun().Luminosity().natural
}

// Times is the power scaled by a share
func (power Power) Times(share float64) Power {
	return Power{power.natural * share}
}

// Over is how many of another power this one is
func (power Power) Over(other Power) float64 {
	return power.natural / other.natural
}

// Spending is how long the power takes to give off an energy
func (power Power) Spending(energy Energy) Duration {
	return Duration{energy.natural / power.natural}
}

// SquareMetres is an area in m2
func SquareMetres(amount float64) Area {
	return Area{amount * squareMetre}
}

// InSquareMetres is the area in m2
func (area Area) InSquareMetres() float64 {
	return area.natural / squareMetre
}
