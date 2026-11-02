package atmosphere

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Way is a normalized viewing direction, with y up and x/z horizontal.
type Way struct{ x, y, z float64 }

// NewWay normalizes direction components and rejects a zero vector or nonfinite
// components.
//
// Components are scaled by their largest magnitude before squaring, so
// overflow or underflow in their squares does not lose a valid direction.
func NewWay(x, y, z float64) (*Way, error) {
	largest := math.Max(math.Abs(x), math.Abs(y))
	largest = math.Max(largest, math.Abs(z))
	if !(largest > none && largest <= math.MaxFloat64) {
		return nil, fmt.Errorf("atmosphere: a way of %g, %g, %g "+
			"points nowhere", x, y, z)
	}
	x, y, z = x/largest, y/largest, z/largest
	length := math.Sqrt(x*x + y*y + z*z)
	return &Way{x: x / length, y: y / length, z: z / length}, nil
}

// dot is the cosine of the angle between two ways.
func (way Way) dot(other Way) float64 {
	return way.x*other.x + way.y*other.y + way.z*other.z
}

// Skylight returns spectral sky radiance along a look from height, normalized
// per W/m² of incoming sunlight from the Sun direction. For that unit
// illumination, radiance is in W m^-2 sr^-1.
//
// The single-scattering model attenuates sunlight to each path sample, applies
// the unpolarized Rayleigh phase function, and attenuates scattered light back
// toward the observer. Earth-blocked samples receive no direct sunlight.
//
// Multiple scattering, ground reflection, haze, ozone, refraction, and
// depolarization are omitted. Depolarization changes the phase function by less
// than about 1.5%.
//
// Unsupported heights and wavelengths return errors from the standard
// atmosphere or CrossSection.
func (air Composition) Skylight(height measure.Length, view, sun Way,
	wavelengths []measure.Length) ([]float64, error) {
	start := height.InMetres()
	if !(start >= seaLevel && start <= topHeight) {
		return nil, fmt.Errorf("atmosphere: %g m, outside the "+
			"standard's %g to %g m", start, seaLevel, topHeight)
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

// The parts of Rayleigh's phase function and the sum along a look, each
// defined.
const (
	phaseScale = 3.0  // defined by rayleigh's phase function: 3/(16 pi)
	phaseOver  = 16.0 // defined by rayleigh's phase function
	middle     = 0.5  // defined: a step is summed at its middle
)

// skylight integrates along a look from geocentric distance using midpoint
// samples, supplied molecular cross-sections, and the chosen view and Sun-path
// subdivisions.
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

// sunward returns molecular column density between a sample and the Sun, and
// whether Earth blocks that direct path.
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
