package atmosphere

import (
	"sync"

	"github.com/janearc/celestial/research"
)

// coefficients is every measured number atmosphere takes from Bodhaine and the
// others (1999), and the Earth's mean radius, as celestial's research data
// holds them: read once, by name.
type coefficients struct {
	molecules                     float64 // a cm^3 at 288.15 K, 1013.25 mb
	constant, first, firstPole    float64 // peck and reeder's index
	second, secondPole            float64
	shortest, longest             float64 // microns the index holds over
	co2Scale                      float64
	nitrogenConstant, nitrogenUp  float64 // bates's king factors
	oxygenConstant, oxygenUp      float64
	oxygenCurve, argonKing        float64
	co2King                       float64
	nitrogen, oxygen, argon       float64 // percent by volume
	molarDry, molarCO2            float64 // g/mol
	columnSlope, columnOffset     float64 // m
	seaGravity, seaLatitude       float64 // list's, cm s^-2
	seaLatitudeSquared            float64
	height, heightLatitude        float64
	heightSquared, heightSquaredL float64
	heightCubed, heightCubedL     float64
	meanRadius                    float64 // m
	pathPieces, skySteps          int     // picked: how finely to sum
	sunPieces                     int
}

// read is the coefficients, from the research data.
func read() coefficients {
	value := research.Value
	return coefficients{
		molecules:          value("rayleigh-molecules-cm3"),
		constant:           value("peck-reeder-constant"),
		first:              value("peck-reeder-first"),
		firstPole:          value("peck-reeder-first-pole"),
		second:             value("peck-reeder-second"),
		secondPole:         value("peck-reeder-second-pole"),
		shortest:           value("peck-reeder-shortest-um"),
		longest:            value("peck-reeder-longest-um"),
		co2Scale:           value("co2-refraction-scale"),
		nitrogenConstant:   value("king-nitrogen-constant"),
		nitrogenUp:         value("king-nitrogen-slope"),
		oxygenConstant:     value("king-oxygen-constant"),
		oxygenUp:           value("king-oxygen-slope"),
		oxygenCurve:        value("king-oxygen-curve"),
		argonKing:          value("king-argon"),
		co2King:            value("king-co2"),
		nitrogen:           value("nitrogen-percent"),
		oxygen:             value("oxygen-percent"),
		argon:              value("argon-percent"),
		molarDry:           value("molar-mass-dry-air"),
		molarCO2:           value("molar-mass-co2-slope"),
		columnSlope:        value("column-height-slope"),
		columnOffset:       value("column-height-offset"),
		seaGravity:         value("list-gravity-sea"),
		seaLatitude:        value("list-gravity-latitude"),
		seaLatitudeSquared: value("list-gravity-latitude-squared"),
		height:             value("list-height"),
		heightLatitude:     value("list-height-latitude"),
		heightSquared:      value("list-height-squared"),
		heightSquaredL:     value("list-height-squared-latitude"),
		heightCubed:        value("list-height-cubed"),
		heightCubedL:       value("list-height-cubed-latitude"),
		meanRadius:         value("earth-mean-radius-m"),
		pathPieces:         int(value("path-pieces")),
		skySteps:           int(value("sky-steps")),
		sunPieces:          int(value("sky-sun-pieces")),
	}
}

// measured is the coefficients, read once, the first time they are
// asked for.
var measured = sync.OnceValue(read)
