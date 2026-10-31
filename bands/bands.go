// Package bands is light carried as amounts in slices of the spectrum: the
// form a renderer needs when light has to be added, filtered, redshifted or
// bent by wavelength, before an eye reduces it to three numbers.
//
// Light in bands is what light is, coarsely. What it looks like is a
// colour, and colour is libtheme's: its bands package folds this light
// into a swatch through the 1931 observer.
package bands

import (
	"math"
	"slices"

	"github.com/janearc/celestial/research"
)

// PerOctave is how many bands there are for every doubling of
// wavelength. It is picked, not derived, and is the one number this
// package takes; celestial's research data holds it, and says who picked
// it and from what.
//
// The bands are even in the logarithm of wavelength, not in nanometres.
// Redshift multiplies every wavelength by the same factor, so it is a
// slide along the bands, not a resampling.
func PerOctave() Band { return Band(research.Value(perOctaveName)) }

// perOctaveName is PerOctave's name in the research data.
const perOctaveName = "bands-per-octave"

// Band is one slice of the spectrum. Band i runs from 2^(i/PerOctave)
// nanometres up to, but not including, 2^((i+1)/PerOctave). Band 0
// starts at one nanometre, and the wavelength doubles every PerOctave
// bands. Visible light is about bands 205 to 230.
type Band int

// The counts the bands are built of, each defined.
const (
	nextBand   = 1 // defined by counting: the band after
	noPower    = 0 // defined by the scale: no light
	asItIs     = 1 // defined by multiplying: a scale that changes nothing
	firstPlace = 0 // defined by a slice's order: its first place
	empty      = 0 // defined by counting: none at all
	above      = 1 // defined by math.IsInf: the infinity above
)

// Of is the band a wavelength in nanometres falls in.
func Of(nm float64) Band { return grid(PerOctave()).of(nm) }

// Edges is where a band starts and where the next one does, in
// nanometres.
func (band Band) Edges() (lo, hi float64) {
	return grid(PerOctave()).edges(band)
}

// grid is a step, in bands per octave. The package only ever uses
// PerOctave; libtheme-css's FUDGE.md holds the steps that were tried.
type grid int

// of is Of at a step.
func (step grid) of(nm float64) Band {
	bands := math.Floor(float64(step) * math.Log2(nm))
	return Band(bands)
}

// edges is Edges at a step.
func (step grid) edges(band Band) (lo, hi float64) {
	per := float64(step)
	return math.Exp2(float64(band) / per),
		math.Exp2(float64(band+nextBand) / per)
}

// Light is the power in each of a run of neighbouring bands, starting at
// band first. Its units are whatever the spectrum it came from used,
// times a nanometre. The zero value is no light.
type Light struct {
	step  grid
	first Band
	power []float64
}

// Split is a spectrum, given as power by wavelength in nanometres, as
// light in bands: each wavelength's power goes into the band it falls in.
//
// Every wavelength counts, including those an eye never sees. Light in
// infrared bands is still light.
func Split(spectrum map[int]float64) Light {
	return grid(PerOctave()).split(spectrum)
}

// split is Split at a step.
func (step grid) split(spectrum map[int]float64) Light {
	lines := make(map[float64]float64, len(spectrum))
	for nm, power := range spectrum {
		lines[float64(nm)] = power
	}
	return step.lines(lines)
}

// Span is the first and last band this light has any entry for. Empty
// light reports first greater than last.
func (light Light) Span() (first, last Band) {
	return light.first, light.first + Band(len(light.power)) - nextBand
}

// Power is the light in one band, nothing outside the span.
func (light Light) Power(band Band) float64 {
	place := int(band - light.first)
	if place < firstPlace || place >= len(light.power) {
		return noPower
	}
	return light.power[place]
}

// Shift slides the light along the bands: up by steps is redder.
// Redshift by a factor of 1 + z is PerOctave times log2(1 + z) steps,
// exact when that is whole. The power in each band travels with it
// unchanged; dimming with distance is the scene's business.
func (light Light) Shift(steps int) Light {
	out := Light{step: light.step, first: light.first + Band(steps)}
	out.power = slices.Clone(light.power)
	return out
}

// Scale is the light made so many times as strong in every band.
func (light Light) Scale(times float64) Light {
	out := Light{step: light.step, first: light.first}
	out.power = make([]float64, len(light.power))
	for place, power := range light.power {
		out.power[place] = power * times
	}
	return out
}

// Add is two lights shining on the same place: in every band, the sum.
func Add(one, other Light) Light {
	if len(one.power) == empty {
		return other.Scale(asItIs)
	}
	if len(other.power) == empty {
		return one.Scale(asItIs)
	}
	oneFirst, oneLast := one.Span()
	otherFirst, otherLast := other.Span()
	first, last := min(oneFirst, otherFirst), max(oneLast, otherLast)
	out := Light{step: one.step, first: first}
	out.power = make([]float64, last-first+nextBand)
	for band := first; band <= last; band++ {
		out.power[band-first] = one.Power(band) + other.Power(band)
	}
	return out
}

// Lines is light given at any wavelengths, each a line, grouped into the
// bands by wavelength in nanometres. Gamma rays are a few thousandths of
// a nanometre, and they have bands too.
func Lines(spectrum map[float64]float64) Light {
	return grid(PerOctave()).lines(spectrum)
}

// lines is Lines at a step, which Split and the tests share. A line at
// no wavelength or less, at an endless one, or at none that is a number,
// is no light, and is left out: there is no band for it.
func (step grid) lines(spectrum map[float64]float64) Light {
	var order []float64
	for nm := range spectrum {
		if nm > noPower && !math.IsInf(nm, above) {
			order = append(order, nm)
		}
	}
	if len(order) == empty {
		return Light{step: step}
	}
	slices.Sort(order)
	first, last := step.of(slices.Min(order)), step.of(slices.Max(order))
	light := Light{step: step, first: first,
		power: make([]float64, last-first+nextBand)}
	// shortest to longest, so the sum in each band comes out the same on
	// every run, as a map's order would not.
	for _, nm := range order {
		light.power[step.of(nm)-first] += spectrum[nm]
	}
	return light
}
