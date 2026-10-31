package atmosphere

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// errWay is a way that points nowhere.
var errWay = errors.New("atmosphere: a way that points nowhere")

// ErrWay is a way that points nowhere: no length, or parts that are not
// numbers.
func ErrWay() error { return errWay }

// Way is a direction from where the sky is seen: y is up, and x and z
// lie level.
type Way struct{ x, y, z float64 }

// NewWay is the direction of three parts. It refuses parts that point
// nowhere, and then gives no way.
func NewWay(x, y, z float64) (*Way, error) {
	length := math.Sqrt(x*x + y*y + z*z)
	if !(length > none && length <= math.MaxFloat64) {
		return nil, fmt.Errorf("%w: %g, %g, %g", errWay, x, y, z)
	}
	return &Way{x: x / length, y: y / length, z: z / length}, nil
}

// dot is the cosine of the angle between two ways.
func (way Way) dot(other Way) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// Skylight is the light the sky sends down a look from a height, at each
// wavelength, for each watt a square metre of sunlight from the sun's
// way: a radiance, in watts a square metre a steradian.
//
// It is single scattering. Sunlight comes down through the air to each
// step of the look, and rayleigh's phase function for unpolarised light
// turns a share of it toward the eye. The air between takes its share
// on the way, and a step the earth hides from the sun sends nothing.
//
// Not here: light scattered twice or more, the ground's light, haze,
// ozone, the bending of light, and the air's depolarisation, which
// changes the phase function by under one and a half percent.
//
// Skylight refuses a height outside the standard and a wavelength
// CrossSection refuses, and then gives no light.
func (air Composition) Skylight(height measure.Length, view, sun Way,
	wavelengths []measure.Length) ([]float64, error) {
	start := height.InMetres()
	if !(start >= seaLevel && start <= topHeight) {
		return nil, fmt.Errorf("%w: %g m", errHeight, start)
	}
	areas := make([]float64, len(wavelengths))
	for place, wavelength := range wavelengths {
		area, err := air.CrossSection(wavelength)
		if err != nil {
			return nil, err
		}
		areas[place] = area.InSquareMetres()
	}
	known := measured()
	return skylight(known.meanRadius+start, view, sun, areas,
		known.skySteps, known.sunPieces), nil
}

// The parts of rayleigh's phase function and the sum along a look, each
// defined.
const (
	phaseScale = 3.0  // defined by rayleigh's phase function: 3/(16 pi)
	phaseOver  = 16.0 // defined by rayleigh's phase function
	middle     = 0.5  // defined: a step is summed at its middle
)

// skylight is the sky's light down a look from a distance off the
// earth's middle, for molecules that scatter the areas given: the look
// summed in so many steps at their middles, and the sunlight's way to
// each step in so many pieces.
func skylight(from float64, view, sun Way, areas []float64, count,
	sunCount int) []float64 {
	cosine := view.dot(sun)
	phase := phaseScale / (phaseOver * math.Pi) * (whole + cosine*cosine)
	length, _ := reach(from, view.y)
	step := length / float64(count)
	sums := make([]float64, len(areas))
	before := none // molecules a square metre from the eye to this step
	for place := range count {
		at := (float64(place) + middle) * step
		here := density(from, view.y, at)
		between := before + here*step*middle
		toSun, lit := sunward(from, view, sun, at, sunCount)
		through := toSun + between
		if lit {
			for band, area := range areas {
				sums[band] += here * math.Exp(-area*through)
			}
		}
		before += here * step
	}
	light := make([]float64, len(areas))
	for band, area := range areas {
		light[band] = phase * area * sums[band] * step
	}
	return light
}

// sunward is the air between the sun and a point so far along a look, in
// molecules a square metre, and whether the sun reaches the point: not
// when the earth stands between.
func sunward(from float64, view, sun Way, at float64, count int) (
	float64, bool) {
	x, y, z := at*view.x, from+at*view.y, at*view.z
	distance := math.Sqrt(x*x + y*y + z*z)
	sine := (x*sun.x + y*sun.y + z*sun.z) / distance
	if _, grounded := reach(distance, sine); grounded {
		return none, false
	}
	return columnOf(distance, sine, count).molecules, true
}
