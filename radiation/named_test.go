package radiation_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
)

// conversions are the named functions, by the unit each reads.
var conversions = map[string]func(float64) (*float64, error){
	"keV": radiation.KeVToNm,
	"MeV": radiation.MeVToNm,
	"GeV": radiation.GeVToNm,
	"TeV": radiation.TeVToNm,
	"MHz": radiation.MHzToNm,
	"GHz": radiation.GHzToNm,
	"um":  radiation.MicronsToNm,
}

// up and down are the infinities math.Inf gives. Defined by math.Inf.
const up, down = 1, -1

// one of each unit is the wavelength worked by hand, and every named
// function has a hand value.
func TestEachUnitIsItsWavelength(test *testing.T) {
	named := pickedNumbers(test).NamedUnits
	if len(named.Wavelengths) != len(conversions) {
		test.Fatalf("%d hand values for %d functions",
			len(named.Wavelengths), len(conversions))
	}
	for unit, want := range named.Wavelengths {
		convert, known := conversions[unit]
		if !known {
			test.Errorf("no named function reads %s", unit)
			continue
		}
		got, err := convert(whole)
		if err != nil {
			test.Errorf("1 %s: %v", unit, err)
			continue
		}
		if !near(*got, want, named.Tolerance) {
			test.Errorf("1 %s is %v nm, by hand %v", unit, *got, want)
		}
	}
}

// The anti-test: zero, below zero, NaN and either infinity give no
// wavelength in any unit, and each returns nil and ErrWavelength.
func TestWhatIsNoValueGivesNoWavelength(test *testing.T) {
	named := pickedNumbers(test).NamedUnits
	for unit, convert := range conversions {
		for _, value := range []float64{none, named.BelowZero,
			math.NaN(), math.Inf(up), math.Inf(down)} {
			got, err := convert(value)
			if !errors.Is(err, radiation.ErrWavelength()) {
				test.Errorf("%v %s was not refused: %v", value, unit, err)
			}
			if got != nil {
				test.Errorf("%v %s came back with %v nm, not nil", value,
					unit, *got)
			}
		}
	}
}

// The anti-test: more energy, or a higher frequency, is a shorter wave,
// so a keV is longer than a MeV, and a MeV than a GeV, and a GeV than a
// TeV, and a MHz than a GHz. a conversion that multiplied where it should
// divide, or scaled the wavelength in place of the energy, fails it.
func TestALargerNamedUnitIsAShorterWave(test *testing.T) {
	for _, order := range [][]string{{"keV", "MeV", "GeV", "TeV"},
		{"MHz", "GHz"}} {
		for place := range len(order) - firstLonger {
			longer, err := conversions[order[place]](whole)
			if err != nil {
				test.Fatal(err)
			}
			shorter, err := conversions[order[place+firstLonger]](whole)
			if err != nil {
				test.Fatal(err)
			}
			if !(*longer > *shorter) {
				test.Errorf("1 %s is %v nm, not longer than 1 %s at %v nm",
					order[place], *longer, order[place+firstLonger],
					*shorter)
			}
		}
	}
}

// firstLonger is how far along an order the next, shorter, unit is.
// Defined by counting.
const firstLonger = 1
