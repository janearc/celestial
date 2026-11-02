package bands

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// Share represents transmission or reflectance fractions, not emitted light.
// Lit applies it to a spectrum. A finite curve passes nothing outside its
// stored bands; Even applies a fraction everywhere. The zero value transmits
// nothing.
type Share struct {
	first  Band
	part   []float64
	beyond float64 // the part passed outside the run
}

// The ends of a share, each defined by what a fraction is.
const (
	passesNothing = 0.0 // defined: none of the light
	passesAll     = 1.0 // defined: all of it
)

// fraction validates a finite value in [0, 1]. A surface returning more light
// than arrives needs an emission model instead.
func fraction(part float64) error {
	if part >= passesNothing && part <= passesAll {
		return nil
	}
	return fmt.Errorf("a share is a fraction between 0 and 1, and %v "+
		"is not one", part)
}

// Even constructs a wavelength-independent fraction, like a grey filter or
// neutral surface. Invalid fractions return an error.
func Even(part float64) (*Share, error) {
	if err := fraction(part); err != nil {
		return nil, err
	}
	return &Share{beyond: part}, nil
}

// Shares constructs transmission or reflectance from samples at integer
// nanometres. Empty samples, nonpositive wavelengths, or fractions outside [0,
// 1] return an error and no share.
//
// Samples within a band are averaged, not summed. Unsampled bands inside the
// run interpolate between neighbouring measurements at the band centre. Outside
// the sampled run, no light passes.
func Shares(spectrum map[int]float64) (*Share, error) {
	if len(spectrum) == empty {
		return nil, errors.New("a share needs at least one " +
			"wavelength and the part passed there")
	}
	var order []int
	for nm, part := range spectrum {
		if nm <= noPower {
			return nil, fmt.Errorf("a share at %d nm: a "+
				"wavelength is longer than nothing", nm)
		}
		if err := fraction(part); err != nil {
			return nil, fmt.Errorf("at %d nm: %w", nm, err)
		}
		order = append(order, nm)
	}
	slices.Sort(order)
	step := grid(PerOctave())
	shortest, longest := slices.Min(order), slices.Max(order)
	first, last := step.of(float64(shortest)), step.of(float64(longest))
	share := Share{first: first, part: make([]float64, last-first+nextBand)}
	for band := first; band <= last; band++ {
		share.part[band-first] = partIn(step, band, order, spectrum)
	}
	return &share, nil
}

// partIn averages measured fractions in a band, or interpolates between
// neighbouring wavelengths at its centre when none fall inside.
func partIn(step grid, band Band, order []int,
	spectrum map[int]float64) float64 {
	var sum float64
	count := empty
	for _, nm := range order {
		if step.of(float64(nm)) == band {
			sum += spectrum[nm]
			count++
		}
	}
	if count > empty {
		return sum / float64(count)
	}
	lo, hi := step.edges(band)
	centre := math.Sqrt(lo * hi)
	above, _ := slices.BinarySearchFunc(order, centre, past)
	below, after := order[above-nextBand], order[above]
	along := (centre - float64(below)) / float64(after-below)
	return spectrum[below] + (spectrum[after]-spectrum[below])*along
}

// How past orders a wavelength against a centre, as a binary search
// wants.
const (
	shortOf = -1 // defined by slices.BinarySearchFunc: before it
	longer  = 1  // defined by slices.BinarySearchFunc: after it
)

// past orders a wavelength against a band's centre: short of it, or
// longer. None is at it, so the search finds the first longer one.
func past(nm int, centre float64) int {
	if float64(nm) > centre {
		return longer
	}
	return shortOf
}

// Part returns the fraction transmitted in a band.
func (share Share) Part(band Band) float64 {
	place := int(band - share.first)
	if place < firstPlace || place >= len(share.part) {
		return share.beyond
	}
	return share.part[place]
}

// Lit returns band power multiplied by the corresponding transmission or
// reflectance fraction. A nil share, like the zero Share, passes no light. The
// input spectrum is not modified.
func Lit(light Light, share *Share) Light {
	if share == nil {
		share = &Share{}
	}
	out := Light{step: light.step, first: light.first,
		power: make([]float64, len(light.power))}
	for place, power := range light.power {
		out.power[place] = power * share.Part(light.first+Band(place))
	}
	return out
}
