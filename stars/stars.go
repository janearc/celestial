// Package stars derives a simple stellar state from birth mass and age: life
// stage, surface temperature, luminosity, and radius.
//
// Constants and units come from measure; empirical and chosen parameters come
// from research data; mainsequence.txt supplies the published dwarf sequence.
// Comments retain the distinction between measurements and tuning.
//
// Two life stages are implemented: Burning and Ended. Ended currently has no
// emitted light or remnant model. The fixed fuel fraction is a simplified
// lifetime model, particularly inaccurate for massive stars.
package stars

import (
	"fmt"

	"github.com/janearc/celestial/measure"
)

// Life identifies the implemented stellar stage: Burning or Ended.
type Life struct{ name string }

// String returns the name of the part of life.
func (life Life) String() string { return life.name }

// Burning is a star on the main sequence, burning hydrogen.
func Burning() Life { return Life{"burning"} }

// Ended marks the end of hydrogen burning. Until remnant models are available,
// the returned state has no light, temperature, or radius.
func Ended() Life { return Life{"ended"} }

// Star stores the modelled state at an age and the light it emits.
type Star struct {
	life       Life
	heat       measure.Temperature
	luminosity measure.Power
	radius     measure.Length
}

// whole is a ratio of a thing to itself. Defined by division.
const whole = 1.0

// nothing is no amount at all. Defined by the scale.
const nothing = 0.0

// NewStar derives a state from birth mass and age. It rejects masses outside
// the published main-sequence table and ages below zero or above the universe’s
// age. NaN also fails these range checks.
func NewStar(mass measure.Mass, age measure.Duration) (*Star, error) {
	table, err := mainSequence()
	if err != nil {
		return nil, err
	}
	born, err := newDwarf(table, mass)
	if err != nil {
		return nil, err
	}
	cosmos := measure.Cosmos()
	if lived := age.Over(cosmos); !(lived >= nothing && lived <= whole) {
		return nil, fmt.Errorf("stars: a star %v years old: the "+
			"universe is %v", age.InYears(), cosmos.InYears())
	}
	star := newStar(*born, age)
	return &star, nil
}

// newStar advances through life stages until the age falls within one.
func newStar(born dwarf, age measure.Duration) Star {
	at, left := newSteps(), age
	for at.next != nil {
		span := at.lasts(born)
		if left.Over(span) < whole {
			break
		}
		left = left.Minus(span)
		at = at.next
	}
	return at.is(born)
}

// step describes a life stage by its duration, state, and successor. The final
// stage has no end.
type step struct {
	lasts func(dwarf) measure.Duration
	is    func(dwarf) Star
	next  *step
}

// newSteps returns the steps of a star's life, first to last.
func newSteps() *step {
	ended := &step{is: over}
	return &step{lasts: lifetime, is: shining, next: ended}
}

// over returns the current nonluminous Ended state.
func over(dwarf) Star { return Star{life: Ended()} }

// shining derives surface temperature from the table’s luminosity and radius
// for a main-sequence star.
func shining(born dwarf) Star {
	surface := measure.Sphere(born.radius)
	heat := measure.SurfaceTemperature(born.light, surface)
	return Star{life: Burning(), luminosity: born.light,
		radius: born.radius, heat: heat}
}

// lifetime divides the model’s available hydrogen-fusion energy by luminosity
// to estimate the burning duration.
func lifetime(born dwarf) measure.Duration {
	fuel := born.mass.Energy().Times(burnedShare() * fusionYield())
	return born.light.Spending(fuel)
}

// Life returns the part of its life the star is in at its age.
func (star Star) Life() Life { return star.life }

// Temperature returns surface temperature, or zero in the current Ended model.
func (star Star) Temperature() measure.Temperature { return star.heat }

// Luminosity returns total emitted power across all wavelengths, not only
// visible light.
func (star Star) Luminosity() measure.Power { return star.luminosity }

// Radius returns stellar radius, or zero in the current Ended model.
func (star Star) Radius() measure.Length { return star.radius }
