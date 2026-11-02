// Package bands carries light as amounts in logarithmic wavelength bins. A
// renderer can add, filter, or shift the spectrum before reducing it to colour.
// Infrared and radio power count as light here too.
//
// This is a coarse spectral representation, not a colour model. Colour
// conversion belongs to libtheme and its CIE 1931 observer integration.
package bands

import (
	"math"
	"slices"

	"github.com/janearc/celestial/research"
)

// PerOctave returns the chosen number of logarithmic bands per doubling of
// wavelength. Research data records the choice and its author.
//
// Uniform spacing in log wavelength makes a redshift a translation through the
// bands rather than a resampling on a linear grid.
func PerOctave() Band { return Band(research.Value(perOctaveName)) }

// perOctaveName is PerOctave's name in the research data.
const perOctaveName = "bands-per-octave"

// Band identifies a half-open wavelength bin. Bin i spans [2^(i/PerOctave),
// 2^((i+1)/PerOctave)) nanometres. Band zero begins at 1 nm; wavelengths double
// every PerOctave bins. At the chosen spacing, visible light occupies roughly
// bands 205–230.
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

// Of returns the band containing a wavelength in nanometres.
func Of(nm float64) Band { return grid(PerOctave()).of(nm) }

// Edges returns the lower-inclusive and upper-exclusive wavelengths of a band,
// in nanometres.
func (band Band) Edges() (lo, hi float64) {
	return grid(PerOctave()).edges(band)
}

// grid is a step, in bands per octave. bands always uses PerOctave.
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

// Light stores power over consecutive wavelength bands, beginning at first.
// Units follow the input spectrum integrated over nanometres. The zero value
// contains no light.
type Light struct {
	step  grid
	first Band
	power []float64
}

// Split groups a spectrum sampled at integer nanometres into bands, adding each
// sample to its containing band. Wavelengths outside human vision are retained.
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

// Span returns the first and last stored bands, both inclusive. For empty
// light, first is greater than last.
func (light Light) Span() (first, last Band) {
	return light.first, light.first + Band(len(light.power)) - nextBand
}

// Power returns the stored power in a band, or zero outside the span.
func (light Light) Power(band Band) float64 {
	place := int(band - light.first)
	if place < firstPlace || place >= len(light.power) {
		return noPower
	}
	return light.power[place]
}

// Shift translates power by whole bands; positive steps move toward longer,
// redder wavelengths. A redshift factor 1+z corresponds to PerOctave*log2(1+z)
// steps, exact when integral. Band powers do not change; distance dimming
// belongs to the caller.
func (light Light) Shift(steps int) Light {
	out := Light{step: light.step, first: light.first + Band(steps)}
	out.power = slices.Clone(light.power)
	return out
}

// Scale returns a copy with every band’s power multiplied by times.
func (light Light) Scale(times float64) Light {
	out := Light{step: light.step, first: light.first}
	out.power = make([]float64, len(light.power))
	for place, power := range light.power {
		out.power[place] = power * times
	}
	return out
}

// Add sums two spectra band by band without modifying either input.
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

// Lines groups power at arbitrary wavelengths in nanometres into bands. Unlike
// Split, it retains sub-nanometre line positions, including gamma rays.
func Lines(spectrum map[float64]float64) Light {
	return grid(PerOctave()).lines(spectrum)
}

// lines implements Lines at the selected grid spacing, shared by Split and the
// tests. Nonpositive, infinite, and NaN wavelengths are omitted because they
// have no corresponding band.
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
