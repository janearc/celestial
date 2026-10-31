package bands

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// Share is the fraction of light passed in each of a run of bands: what a
// filter lets through, or what a surface sends back. It is not light.
// Light arriving and a share of it are different things, so they are
// different types, and Lit is the one way to put them together.
//
// Outside its run a share passes nothing, unless it was made Even, which
// passes the same part in every band. The zero value passes nothing.
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

// fraction is whether a part is a fraction a share can be: nothing to
// all, and a number. A surface that sends back more than arrives is a
// lamp.
func fraction(part float64) error {
	if part >= passesNothing && part <= passesAll {
		return nil
	}
	return fmt.Errorf("a share is a fraction between 0 and 1, and %v "+
		"is not one", part)
}

// Even is a share that passes the same part of the light in every band:
// a grey filter, or a perfectly grey surface. It refuses a part that is
// not a fraction, and then gives no share.
func Even(part float64) (*Share, error) {
	if err := fraction(part); err != nil {
		return nil, err
	}
	return &Share{beyond: part}, nil
}

// Shares is a share given as the part passed at wavelengths in
// nanometres, as measured: a reflectance or transmittance curve. It
// refuses no wavelengths at all, a wavelength not longer than nothing
// and a part that is not a fraction, and then gives no share.
//
// A band's part is the mean of the wavelengths given in it, since a
// share is a fraction and fractions do not add. A band inside the run
// with none given takes the straight line between its neighbours, read
// at its centre. Outside the run nothing passes.
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

// partIn is one band's part: the mean of the wavelengths given in it, or
// the line between its neighbours at its centre when none are.
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

// Part is the fraction the share passes in one band.
func (share Share) Part(band Band) float64 {
	place := int(band - share.first)
	if place < firstPlace || place >= len(share.part) {
		return share.beyond
	}
	return share.part[place]
}

// Lit is light after a share of it: in every band, the light's power
// times the part passed there. No share at all passes nothing, as the
// zero Share does.
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
