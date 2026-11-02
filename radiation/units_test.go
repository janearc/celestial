package radiation_test

import (
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/celestial/si"
)

// A photon's energy times its wavelength is 1239.84 electronvolt
// nanometres, so a one electronvolt photon is that long and a 1 keV photon
// is a thousandth of it.
func TestPhotonUnits(test *testing.T) {
	units := pickedNumbers(test).Units
	if nm := radiation.FromEnergy(radiation.Electronvolt); !(math.Abs(nm-units.HC) <= units.HCTolerance) {
		test.Errorf("hc is %v eV nm", nm)
	}
	if nm := radiation.FromEnergy(si.Kilo(radiation.Electronvolt)); !(math.Abs(nm-units.KeVNM) <= units.HCTolerance) {
		test.Errorf("a 1 keV photon is %v nm", nm)
	}
	if ev := radiation.Energy(units.HC); !near(ev, radiation.Electronvolt, units.HCTolerance) {
		test.Errorf("a photon of hc nanometres carries %v eV", ev)
	}
}

// The property: wavelength through energy, and through frequency, comes
// back as it went.
func TestTheConversionsUndoEachOther(test *testing.T) {
	tolerance := tolerant(pickedNumbers(test).Units.Tolerance)
	rapid.Check(test, tolerance.undo)
}

// tolerant is a relative tolerance, and the draws of the properties that
// are tested to it.
type tolerant float64

// undo is one draw of the property above.
func (tolerance tolerant) undo(draw *rapid.T) {
	nm := wavelength(draw, "nm")
	if got := radiation.FromEnergy(radiation.Energy(nm)); !near(got, nm, float64(tolerance)) {
		draw.Fatalf("%v nm through energy came back %v", nm, got)
	}
	if got := radiation.HertzToNm(radiation.NmToHertz(nm)); !near(got, nm, float64(tolerance)) {
		draw.Fatalf("%v nm through hertz came back %v", nm, got)
	}
	if got := radiation.MetresToNm(radiation.NmToMetres(nm)); !near(got, nm, float64(tolerance)) {
		draw.Fatalf("%v nm through metres came back %v", nm, got)
	}
}

// The anti-test: a photon of twice the energy is half as long, and of
// twice the frequency half as long, so more of either is a shorter wave
// and a longer wave carries less energy.
func TestMoreEnergyIsAShorterWave(test *testing.T) {
	tolerance := tolerant(pickedNumbers(test).Units.Tolerance)
	rapid.Check(test, tolerance.shorter)
}

// shorter is one draw of the property above.
func (tolerance tolerant) shorter(draw *rapid.T) {
	nm := wavelength(draw, "nm")
	if got := radiation.FromEnergy(twice * radiation.Energy(nm)); !near(got, nm/twice, float64(tolerance)) {
		draw.Fatalf("twice the energy of %v nm is %v nm", nm, got)
	}
	if got := radiation.HertzToNm(twice * radiation.NmToHertz(nm)); !near(got, nm/twice, float64(tolerance)) {
		draw.Fatalf("twice the frequency of %v nm is %v nm", nm, got)
	}
	if !(radiation.Energy(twice*nm) < radiation.Energy(nm)) {
		draw.Fatalf("a wave of %v nm is not less energetic than %v nm", twice*nm, nm)
	}
}

// The test and the anti-test: the hydrogen line, at 1420.405751
// megahertz, is the 21 centimetre line, and radio's; the same line at
// twice the frequency is not.
func TestTheHydrogenLine(test *testing.T) {
	units := pickedNumbers(test).Units
	hydrogen := units.HydrogenMHz * si.Mega(radiation.Hertz)
	cm := radiation.NmToMetres(radiation.HertzToNm(hydrogen)) / si.Centi(length.Metre)
	if !(math.Abs(cm-units.HydrogenCM) <= units.HydrogenTol) {
		test.Errorf("the hydrogen line is %v cm", cm)
	}
	if nm := radiation.HertzToNm(hydrogen); !radiation.Radio().Holds(nm) ||
		radiation.Gamma().Holds(nm) {
		test.Errorf("radio does not hold the hydrogen line, or gamma does: %v nm", nm)
	}
	doubled := radiation.NmToMetres(radiation.HertzToNm(twice*hydrogen)) / si.Centi(length.Metre)
	if math.Abs(doubled-units.HydrogenCM) <= units.HydrogenTol {
		test.Errorf("twice the hydrogen line's frequency is %v cm", doubled)
	}
}

// Lengths in metres come into nanometres and go back: a metre is a
// thousand million nanometres, a one metre wave is c hertz, WISE's
// first band of 3.4 microns is 3400 nm, and an angstrom is a tenth of a
// nanometre.
func TestMetresAndNanometres(test *testing.T) {
	units := pickedNumbers(test).Units
	if !near(radiation.MetresToNm(length.Metre), length.Metre/si.Nano(length.Metre), units.Tolerance) {
		test.Errorf("a metre is %v nm", radiation.MetresToNm(length.Metre))
	}
	metre := radiation.MetresToNm(length.Metre)
	if !near(radiation.NmToHertz(metre), si.Speed, units.Tolerance) {
		test.Error("a one metre wave is not c hertz")
	}
	wise := radiation.MetresToNm(units.WiseMicron * si.Micro(length.Metre))
	if !near(wise, units.WiseNM, units.Tolerance) {
		test.Errorf("3.4 microns is %v nm", wise)
	}
	if nm := radiation.MetresToNm(length.Angstrom); !near(nm, units.AngstromNM, units.Tolerance) {
		test.Errorf("an angstrom is %v nm", nm)
	}
}

// The anti-test: a length in metres is not the same number as nanometres,
// and a gamma ray is far less than a metre long.
func TestANanometreIsNotAMetre(test *testing.T) {
	units := pickedNumbers(test).Units
	if radiation.MetresToNm(units.TenMetres) == units.TenMetres {
		test.Error("ten metres is ten nanometres")
	}
	if radiation.NmToMetres(units.GammaNM) >= length.Metre {
		test.Errorf("a gamma ray of %v nm is a metre or more", units.GammaNM)
	}
}

// The units' multiples, from the SI's prefixes, are its powers of ten.
func TestTheUnitsArePowersOfTen(test *testing.T) {
	ladders := pickedNumbers(test).Ladders
	energy := ladder(radiation.Electronvolt)
	frequency := ladder(radiation.Hertz)
	if !slices.Equal(energy, ladders.Energy) {
		test.Errorf("the energy units are %v, not %v", energy, ladders.Energy)
	}
	if !slices.Equal(frequency, ladders.Frequency) {
		test.Errorf("the frequency units are %v, not %v", frequency, ladders.Frequency)
	}
}

// The anti-test: a unit's multiples are not the unit, and they rise in
// order. That each is a thousand times the one before is the test above.
func TestTheMultiplesAreNotTheUnit(test *testing.T) {
	energy := ladder(radiation.Electronvolt)
	if !slices.IsSorted(energy) {
		test.Errorf("the energy units are not in order: %v", energy)
	}
	if distinct := slices.Compact(slices.Clone(energy)); len(distinct) != len(energy) {
		test.Errorf("a multiple is the unit: %v", energy)
	}
}

// ladder is a unit and its kilo, mega, giga and tera multiples.
func ladder(unit float64) []float64 {
	return []float64{unit, si.Kilo(unit), si.Mega(unit), si.Giga(unit),
		si.Tera(unit)}
}
