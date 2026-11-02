package gw

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// conversion pairs an SI unit name with the scale converting one catalogue unit
// to that SI unit.
type conversion struct {
	unit   string
	factor float64
}

// The numbers the units below are built from, each defined.
const (
	// one is one of a unit. Defined by counting.
	one = 1.0
	// ergInJoules is the CGS erg in joules. Defined by CGS: 1e-7 J.
	ergInJoules = 1e-7
	// peakScale is the power of ten the centre writes its peak luminosity's
	// unit with. Defined by that unit, 10^56 erg s^-1.
	peakScale = 1e56
)

// toSI looks up an explicit unit conversion and rejects unknown strings.
func toSI(written string) (*conversion, error) {
	sun := measure.SolarMasses(one).InKilograms()
	into := conversion{factor: one}
	switch written {
	case "":
	case "M_sun":
		into = conversion{unit: "kg", factor: sun}
	case "M_sun c^2":
		into = conversion{unit: "J", factor: sun * si.Speed * si.Speed}
	case "Mpc":
		into = conversion{unit: "m", factor: si.Mega(length.Parsec)}
	case "yr^-1":
		year := measure.Years(one).InSeconds()
		into = conversion{unit: "s^-1", factor: one / year}
	case "10^56 erg s^-1":
		into = conversion{unit: "W", factor: peakScale * ergInJoules}
	case "deg^2":
		degree := measure.Degrees(one).InRadians()
		into = conversion{unit: "sr", factor: math.Pow(degree, squared)}
	default:
		return nil, fmt.Errorf("gw: no conversion for the unit %q",
			written)
	}
	return &into, nil
}

// squared is the power of a square degree. Defined by the unit.
const squared = 2
