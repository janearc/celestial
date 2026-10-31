package gw_test

import (
	"encoding/json"
	"errors"
	"maps"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/janearc/celestial/gw"
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
	"github.com/janearc/celestial/sky"
	"pgregory.net/rapid"
)

// expected is a value and its interval, in the centre's units.
type expected struct {
	Value   float64 `json:"value"`
	Lowest  float64 `json:"lowest"`
	Highest float64 `json:"highest"`
}

// picked is what the tests expect of GW150914, from testdata/gw.json.
type picked struct {
	File    string  `json:"file"`
	Name    string  `json:"name"`
	Catalog string  `json:"catalog"`
	GPS     float64 `json:"gps"`
	UTC     struct {
		Year, Month, Day, Hour, Minute int
		Second                         float64
	} `json:"utc"`
	SecondTol        float64             `json:"second tolerance"`
	Suns             map[string]expected `json:"suns"`
	Megaparsecs      map[string]expected `json:"megaparsecs"`
	Pure             map[string]expected `json:"pure"`
	Unbounded        string              `json:"unbounded"`
	UnboundedVal     float64             `json:"unbounded value"`
	UnboundedWritten string              `json:"unbounded written"`
	BoundedWritten   string              `json:"bounded written"`
	Absent           []string            `json:"absent"`
	PeakWatts        float64             `json:"peak luminosity watts"`
	PerYear          float64             `json:"per year"`
	Tolerance        float64             `json:"tolerance"`
	Drawn            struct {
		Lowest  int `json:"lowest power"`
		Highest int `json:"highest power"`
	} `json:"drawn"`
	Units      []unitOf `json:"units"`
	NearMisses []miss   `json:"near misses"`
}

// miss is a unit nearly one of the centre's, written for a parameter.
type miss struct {
	Parameter string `json:"parameter"`
	Unit      string `json:"unit"`
}

// unitOf is a parameter the file gives in one of the centre's units, the
// value it gives, and that unit in si, typed from its definition.
type unitOf struct {
	Parameter string  `json:"parameter"`
	Written   float64 `json:"written"`
	Unit      string  `json:"unit"`
	SIUnit    string  `json:"si unit"`
	Per       float64 `json:"per"`
	Tolerance float64 `json:"tolerance"`
}

// readJSON reads a test file into a place, or the test stops.
func readJSON(test *testing.T, name string, into any) {
	test.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		test.Fatal(err)
	}
}

// pickedNumbers reads what the tests expect, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	readJSON(test, "testdata/gw.json", &numbers)
	return numbers
}

// gw150914 is the centre's GW150914, read, or the test stops.
func gw150914(test *testing.T) *gw.Event {
	test.Helper()
	raw, err := os.ReadFile(pickedNumbers(test).File)
	if err != nil {
		test.Fatal(err)
	}
	events, err := gw.ReadEvents(raw)
	if err != nil {
		test.Fatal(err)
	}
	if len(events) != single {
		test.Fatalf("%d events read", len(events))
	}
	return events[firstEvent]
}

// The counts and places these tests use, each defined.
const (
	single     = 1 // defined by counting: one event in the file
	firstEvent = 0 // defined by a slice's order
	oneSun     = 1 // defined by the unit: one of the sun's masses
	oneYear    = 1 // defined by the unit: one julian year
)

// estimate is an event's estimate of a parameter, or the test stops.
func estimate(test *testing.T, event *gw.Event, name string) *gw.Estimate {
	test.Helper()
	found, err := event.Estimate(name)
	if err != nil {
		test.Fatal(err)
	}
	return found
}

// near is whether a value lands within a share of what is expected.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// check holds an estimate, turned back into the centre's unit, to what is
// expected of it.
func check(test *testing.T, event *gw.Event, name string, want expected,
	per float64, unit string) {
	test.Helper()
	found := estimate(test, event, name)
	tolerance := pickedNumbers(test).Tolerance
	lowest, highest, known := found.Interval()
	if found.Unit() != unit || !known ||
		!near(found.Value()/per, want.Value, tolerance) ||
		!near(lowest/per, want.Lowest, tolerance) ||
		!near(highest/per, want.Highest, tolerance) {
		test.Errorf("%s is %v (%v to %v, known %v), not %+v %s", name,
			found, lowest/per, highest/per, known, want, unit)
	}
}

// GW150914 is read with its name, catalogue and gps time, and the time
// is the open science centre's gps 1126259462.4, 09:50:45.4 utc on
// 2015-09-14. The discovery paper gives 09:50:45 utc, to the second
// (read 2026-10-31); the tenth is gwosc's.
func TestGW150914IsReadWhole(test *testing.T) {
	want := pickedNumbers(test)
	event := gw150914(test)
	if event.String() != want.Name || event.Catalog() != want.Catalog {
		test.Errorf("read %s in %s", event, event.Catalog())
	}
	seconds, err := event.At().GPSSeconds()
	if err != nil || !near(seconds, want.GPS, want.Tolerance) {
		test.Errorf("at gps %v, %v", seconds, err)
	}
	utc, err := event.At().On(sky.UTC())
	if err != nil {
		test.Fatal(err)
	}
	year, month, day, hour, minute, second := utc.Calendar()
	at := want.UTC
	if year != at.Year || month != at.Month || day != at.Day ||
		hour != at.Hour || minute != at.Minute ||
		!(math.Abs(second-at.Second) <= want.SecondTol) {
		test.Errorf("at %d-%d-%d %d:%d:%v utc", year, month, day, hour,
			minute, second)
	}
}

// Its masses are in kilograms, its distance in metres and its spin and
// redshift pure numbers, each with its interval, as the centre gives them.
func TestItsEstimatesAreInSI(test *testing.T) {
	want := pickedNumbers(test)
	event := gw150914(test)
	sun := measure.SolarMasses(oneSun).InKilograms()
	for name, mass := range want.Suns {
		check(test, event, name, mass, sun, "kg")
	}
	megaparsec := si.Mega(length.Parsec)
	for name, distance := range want.Megaparsecs {
		check(test, event, name, distance, megaparsec, "m")
	}
	for name, pure := range want.Pure {
		check(test, event, name, pure, oneSun, "")
	}
}

// Each of the centre's units reads in si by the factor its definition
// gives, typed in the test data and not worked out by measure, so a wrong
// factor in gw cannot cancel against the same one in the test: a mass in
// suns, a distance in megaparsecs, an energy in suns times c squared and
// an area of sky in square degrees.
func TestEachUnitIsItsDefinitionInSI(test *testing.T) {
	want := pickedNumbers(test)
	if len(want.Units) == noUnits {
		test.Fatal("no units to hold")
	}
	event := gw150914(test)
	for _, unit := range want.Units {
		found := estimate(test, event, unit.Parameter)
		if found.Unit() != unit.SIUnit || !near(found.Value(),
			unit.Written*unit.Per, unit.Tolerance) {
			test.Errorf("%s of %v %s is %v %s, not %v %s", unit.Parameter,
				unit.Written, unit.Unit, found.Value(), found.Unit(),
				unit.Written*unit.Per, unit.SIUnit)
		}
	}
}

// noUnits is test data with no units in it. Defined by counting.
const noUnits = 0

// The peak luminosity's 10^56 erg s^-1 is in watts, and the false alarm
// rate's per year is per second.
func TestItsOddUnitsAreInSIToo(test *testing.T) {
	want := pickedNumbers(test)
	event := gw150914(test)
	peak := estimate(test, event, "L_peak")
	if peak.Unit() != "W" || !near(peak.Value(), want.PeakWatts,
		want.Tolerance) {
		test.Errorf("the peak luminosity is %v", peak)
	}
	alarms := estimate(test, event, "far")
	perSecond := want.PerYear / measure.Years(oneYear).InSeconds()
	if alarms.Unit() != "s^-1" || !near(alarms.Value(), perSecond,
		want.Tolerance) {
		test.Errorf("the false alarm rate is %v", alarms)
	}
}

// The anti-test: an estimate the centre gives without bounds has no known
// interval, and its two ends are the value.
func TestAnEstimateWithoutBoundsHasNoInterval(test *testing.T) {
	want := pickedNumbers(test)
	snr := estimate(test, gw150914(test), want.Unbounded)
	lowest, highest, known := snr.Interval()
	if known || lowest != snr.Value() || highest != snr.Value() ||
		!near(snr.Value(), want.UnboundedVal, want.Tolerance) {
		test.Errorf("%s is %v: %v to %v, known %v", want.Unbounded, snr,
			lowest, highest, known)
	}
	if written := snr.String(); written != want.UnboundedWritten {
		test.Errorf("%s is written %q", want.Unbounded, written)
	}
	if written := estimate(test, gw150914(test), "a_final").String(); written !=
		want.BoundedWritten {
		test.Errorf("a_final is written %q", written)
	}
}

// The anti-test: a parameter the centre left blank, or never gave, is absent:
// refused, with no estimate, never a nought.
func TestABlankParameterIsAbsent(test *testing.T) {
	event := gw150914(test)
	for _, name := range pickedNumbers(test).Absent {
		found, err := event.Estimate(name)
		if !errors.Is(err, gw.ErrAbsent()) || found != nil {
			test.Errorf("%s gave %v, %v", name, found, err)
		}
	}
	for _, name := range event.Parameters() {
		if _, err := event.Estimate(name); err != nil {
			test.Errorf("%s is listed and absent: %v", name, err)
		}
	}
}

// refusal is a document gw must refuse, and why.
type refusal struct {
	Why  string `json:"why"`
	JSON string `json:"json"`
}

// The anti-test: what is not the centre's events, an event without its name
// or time, a unit gw cannot convert, and a bound on the wrong side are
// refused, and give no events.
func TestWhatIsNotTheCentresIsRefused(test *testing.T) {
	var refused struct {
		Documents []refusal `json:"documents"`
	}
	readJSON(test, "testdata/refused.json", &refused)
	for _, document := range refused.Documents {
		events, err := gw.ReadEvents([]byte(document.JSON))
		if err == nil || events != nil {
			test.Errorf("%s: read %v", document.Why, events)
		}
		if !errors.Is(err, gw.ErrEvents()) && !errors.Is(err, gw.ErrUnit()) {
			test.Errorf("%s: refused with %v", document.Why, err)
		}
	}
}

// The property and its anti-test: any value written for a mass or a
// distance, in the centre's unit, reads as that value times the one
// factor the file's own value reads with, the conversion having no
// offset and no bend; and the same estimate in a unit the centre's table
// does not have is refused, and gives no events.
func TestAnyValueReadsInSIByOneFactor(test *testing.T) {
	numbers := pickedNumbers(test)
	if numbers.Drawn.Highest <= numbers.Drawn.Lowest {
		test.Fatalf("no powers to draw from: %+v", numbers.Drawn)
	}
	raw, err := os.ReadFile(numbers.File)
	if err != nil {
		test.Fatal(err)
	}
	event, factors := gw150914(test), map[string]float64{}
	for _, written := range []map[string]expected{numbers.Suns,
		numbers.Megaparsecs} {
		for name, want := range written {
			factors[name] = estimate(test, event, name).Value() /
				want.Value
		}
	}
	rapid.Check(test, rewriting{raw: raw, name: numbers.Name,
		factors: factors, numbers: numbers}.check)
}

// rewriting writes drawn values and units over the file's and reads it.
type rewriting struct {
	raw     []byte
	name    string
	factors map[string]float64
	numbers picked
}

// The parts a drawn value and unit are made of, each defined.
const (
	leastMantissa = 1.0  // defined: a mantissa's least
	mostMantissa  = 10.0 // defined: a mantissa's most, one decade on
	unitSuffix    = "_unit"
	smallLetters  = "[a-z]{1,8}" // defined: no unit the centre writes
)

// check draws a parameter, a value and a unit, and reads each written in.
func (rewrite rewriting) check(test *rapid.T) {
	names := slices.Sorted(maps.Keys(rewrite.factors))
	name := rapid.SampledFrom(names).Draw(test, "parameter")
	mantissa := rapid.Float64Range(leastMantissa, mostMantissa).Draw(test,
		"mantissa")
	power := rapid.IntRange(rewrite.numbers.Drawn.Lowest,
		rewrite.numbers.Drawn.Highest).Draw(test, "power")
	value := mantissa * math.Pow10(power)
	events, err := gw.ReadEvents(rewrite.written(test, name, value))
	if err != nil || len(events) != single {
		test.Fatalf("%s of %v: %v, %v", name, value, events, err)
	}
	got := events[firstEvent]
	found, err := got.Estimate(name)
	if err != nil {
		test.Fatalf("%s of %v: %v", name, value, err)
	}
	want := value * rewrite.factors[name]
	if !near(found.Value(), want, rewrite.numbers.Tolerance) {
		test.Fatalf("%s of %v reads %v, not %v", name, value,
			found.Value(), want)
	}
	unit := rapid.StringMatching(smallLetters).Draw(test, "unit")
	unwritten := rewrite.written(test, name+unitSuffix, unit)
	if events, err := gw.ReadEvents(unwritten); events != nil ||
		!errors.Is(err, gw.ErrUnit()) {
		test.Fatalf("%s in %q: %v, %v", name, unit, events, err)
	}
}

// written is the file with one of its event's fields written over.
func (rewrite rewriting) written(test *rapid.T, field string,
	value any) []byte {
	return writtenOver(test, rewrite.raw, rewrite.name, field, value)
}

// stopper is what a test or a draw stops with: both testing's T and
// rapid's T are one.
type stopper interface {
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// writtenOver is a file of events with one field of a named event
// written over, or the test stops.
func writtenOver(test stopper, raw []byte, name, field string,
	value any) []byte {
	var file struct {
		Events map[string]map[string]any `json:"events"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		test.Fatal(err)
	}
	event, held := file.Events[name]
	if !held {
		test.Fatalf("no event %q in the file", name)
	}
	event[field] = value
	out, err := json.Marshal(file)
	if err != nil {
		test.Fatal(err)
	}
	return out
}

// The anti-test: a unit that is nearly one of the centre's, but not it,
// is refused, and gives no events, so the table converts only what the
// centre writes: a capital S in M_sun, a lower-case Mpc, a square degree
// without its caret, a space, and c cubed for c squared.
func TestANearlyRightUnitIsRefused(test *testing.T) {
	want := pickedNumbers(test)
	if len(want.NearMisses) == noUnits {
		test.Fatal("no near misses to hold")
	}
	raw, err := os.ReadFile(want.File)
	if err != nil {
		test.Fatal(err)
	}
	for _, miss := range want.NearMisses {
		changed := writtenOver(test, raw, want.Name,
			miss.Parameter+unitSuffix, miss.Unit)
		if events, err := gw.ReadEvents(changed); events != nil ||
			!errors.Is(err, gw.ErrUnit()) {
			test.Errorf("%s in %q: %v, %v", miss.Parameter, miss.Unit,
				events, err)
		}
	}
}
