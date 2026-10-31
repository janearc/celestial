// Package stars is stars as data: a mass and an age go in, and out
// comes what the star is now, with its temperature, brightness and size.
//
// Nothing here is a number picked by hand in code. measure holds nature's
// constants and the units; celestial's research data what stars takes
// on trust besides, each with its source; mainsequence.txt a table.
//
// A star's life is a run of stages, each a state that lasts a while and
// hands on to the next. Two are built: burning, and ended. What an ended
// star leaves comes with the published tables that say which and how.
package stars

import (
	"fmt"

	"github.com/janearc/celestial/measure"
)

// Stage is a step in a star's life.
type Stage struct{ name string }

// String is the stage's name.
func (stage Stage) String() string { return stage.name }

// Burning is a star on the main sequence, burning hydrogen.
func Burning() Stage { return Stage{"burning"} }

// Ended is a star whose burning is over. Until the remnants come, it
// gives no light here.
func Ended() Stage { return Stage{"ended"} }

// Star is a star at an age: what it is, and what it gives off.
type Star struct {
	stage      Stage
	heat       measure.Temperature
	luminosity measure.Power
	radius     measure.Length
}

// whole is a ratio of a thing to itself. Defined by division.
const whole = 1.0

// nothing is no amount at all. Defined by the scale.
const nothing = 0.0

// NewStar is a star of a mass at birth, at an age. It refuses a mass the
// published main sequence does not reach, either way, and an age below
// nought or older than the universe, and then gives no star.
//
// Each check says what must hold, so a NaN, which fails every
// comparison, is refused too.
func NewStar(mass measure.Mass, age measure.Duration) (*Star, error) {
	table, err := mainSequence()
	if err != nil {
		return nil, err
	}
	lightest, heaviest := ends(table)
	if !(mass.Over(lightest.mass) >= whole &&
		mass.Over(heaviest.mass) <= whole) {
		return nil, refuse("%v suns: the main sequence runs from %v "+
			"(%s) to %v (%s)", mass.InSolarMasses(),
			lightest.mass.InSolarMasses(), lightest.kind,
			heaviest.mass.InSolarMasses(), heaviest.kind)
	}
	cosmos := measure.Cosmos()
	if lived := age.Over(cosmos); !(lived >= nothing && lived <= whole) {
		return nil, refuse("a star %v years old: the universe is %v",
			age.InYears(), cosmos.InYears())
	}
	star := newStar(along(table, mass), age)
	return &star, nil
}

// newStar walks a star's stages from its birth until its age runs out
// inside one, and is the star as that stage has it.
func newStar(born dwarf, age measure.Duration) Star {
	at, left := life(), age
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

// step is a stage as a state: how long it lasts for a star, what the
// star is during it, and the step after. The last lasts for ever.
type step struct {
	lasts func(dwarf) measure.Duration
	is    func(dwarf) Star
	next  *step
}

// life is a star's stages, first to last.
func life() *step {
	ended := &step{is: over}
	return &step{lasts: lifetime, is: shining, next: ended}
}

// over is a star whose burning has ended: it gives no light here.
func over(dwarf) Star { return Star{stage: Ended()} }

// shining is a star burning on the main sequence: its brightness and
// radius from the table, and its temperature what those two make.
func shining(born dwarf) Star {
	surface := measure.Sphere(born.radius)
	heat := measure.SurfaceTemperature(born.light, surface)
	return Star{stage: Burning(), luminosity: born.light,
		radius: born.radius, heat: heat}
}

// lifetime is how long a star burns hydrogen: the energy in the share
// of its mass that becomes helium, over the power it gives off.
func lifetime(born dwarf) measure.Duration {
	fuel := born.mass.Energy().Times(burnedShare() * fusionYield())
	return born.light.Spending(fuel)
}

// Stage is what the star is at its age.
func (star Star) Stage() Stage { return star.stage }

// Temperature is the star's surface temperature; an ended star's is
// nought here.
func (star Star) Temperature() measure.Temperature { return star.heat }

// Luminosity is how bright the star is, at every wavelength, not only
// those an eye sees.
func (star Star) Luminosity() measure.Power { return star.luminosity }

// Radius is the star's radius; an ended star's is nought here.
func (star Star) Radius() measure.Length { return star.radius }

// refuse is a constructor's error, saying what it would not make.
func refuse(format string, about ...any) error {
	return fmt.Errorf("stars: "+format, about...)
}
