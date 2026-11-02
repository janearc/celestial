package atmosphere

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
)

// The 1976 standard's constants, as us standard atmosphere, 1976, fixes
// them for its model; each is defined by it. Its gas constant is its own,
// not the committee on data's.
const (
	// earthRadius is in metres, for geopotential height. Defined.
	earthRadius = 6356766.0
	// gravity is in m s^-2. Defined.
	gravity = 9.80665
	// gasConstant is in J mol^-1 K^-1. Defined.
	gasConstant = 8.31432
	// molarMass is in kg mol^-1, air at sea level. Defined.
	molarMass = 0.0289644
	// seaPressure is in pascals. Defined.
	seaPressure = 101325.0
	// seaTemperature is in kelvin. Defined.
	seaTemperature = 288.15
	// topHeight is the standard's top, in metres above sea level.
	// Defined.
	topHeight = 86000.0
)

// layer is where one of the standard's layers starts, in geopotential
// metres, and how its temperature changes with height there, in K/m.
type layer struct {
	Base     float64 `json:"base"`
	Gradient float64 `json:"gradient"`
}

// standardText is the standard's layers, as standard.json holds them.
//
//go:embed standard.json
var standardText []byte

// readLayers is the standard's seven layers, or nil and the error that
// stopped it.
func readLayers() ([]layer, error) {
	var file struct {
		Layers []layer `json:"layers"`
	}
	if err := json.Unmarshal(standardText, &file); err != nil {
		return nil, err
	}
	return file.Layers, nil
}

// layers is the standard's layers, read once. Above 80 km the standard's
// kinetic temperature departs from the one here by under 0.3 K.
var layers = sync.OnceValues(readLayers)

// The numbers the walk up the layers compares with, each defined.
const (
	seaLevel  = 0.0 // defined by the scale: no height above the sea
	level     = 0.0 // defined: a layer whose temperature does not change
	nextLayer = 1   // defined by a slice's order: the layer above
	upward    = 1   // defined by math.Inf: the sign of the infinity above
)

// Standard returns the 1976 standard atmosphere from sea level to 86 km
// geometric height. Out-of-range input returns an error.
func Standard(height measure.Length) (*Air, error) {
	metres := height.InMetres()
	if !(metres >= seaLevel && metres <= topHeight) {
		return nil, fmt.Errorf("atmosphere: %g m, outside the "+
			"standard's %g to %g m", metres, seaLevel, topHeight)
	}
	all, err := layers()
	if err != nil {
		return nil, err
	}
	geopotential := earthRadius * metres / (earthRadius + metres)
	kelvin, pascals := seaTemperature, seaPressure
	for place, one := range all {
		top := math.Inf(upward)
		if place+nextLayer < len(all) {
			top = all[place+nextLayer].Base
		}
		climb := math.Min(geopotential, top) - one.Base
		kelvin, pascals = one.climbed(kelvin, pascals, climb)
		if geopotential <= top {
			break
		}
	}
	air := newAir(kelvin, pascals)
	return &air, nil
}

// climbed integrates hydrostatic pressure and temperature from a layer base
// through its constant temperature gradient.
func (one layer) climbed(kelvin, pascals, climb float64) (float64,
	float64) {
	scale := gravity * molarMass / gasConstant
	if one.Gradient == level {
		return kelvin, pascals * math.Exp(-scale*climb/kelvin)
	}
	warmer := kelvin + one.Gradient*climb
	return warmer, pascals * math.Pow(kelvin/warmer, scale/one.Gradient)
}
