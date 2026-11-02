package research_test

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"gopkg.in/yaml.v3"

	"github.com/janearc/celestial/research"
)

// cited is sources.yaml as these tests read it: the key of every source,
// and whether it was read and when, or that it was not.
type cited struct {
	Sources []struct {
		Key  string `yaml:"key"`
		Read string `yaml:"read"`
	} `yaml:"sources"`
}

// sources is sources.yaml read, or the test stops.
func sources(test *testing.T) cited {
	test.Helper()
	raw, err := os.ReadFile("sources.yaml")
	if err != nil {
		test.Fatal(err)
	}
	var read cited
	if err := yaml.Unmarshal(raw, &read); err != nil {
		test.Fatal(err)
	}
	return read
}

// sourceKeys is every key sources.yaml holds, or the test stops.
func sourceKeys(test *testing.T) map[string]bool {
	test.Helper()
	keys := map[string]bool{}
	for _, source := range sources(test).Sources {
		keys[source.Key] = true
	}
	return keys
}

// notRead is how sources.yaml says a source has not been read against:
// its read begins so. Defined by sources.yaml's own entries.
const notRead = "not yet"

// unread is every number measured from a source not yet read against,
// by its name, from the research data and sources.yaml together.
func unread(test *testing.T) []string {
	test.Helper()
	carried := map[string]bool{}
	for _, source := range sources(test).Sources {
		if strings.HasPrefix(source.Read, notRead) {
			carried[source.Key] = true
		}
	}
	var names []string
	for _, number := range data(test) {
		if number.From != nil && carried[number.From.Key] {
			names = append(names, number.Name)
		}
	}
	slices.Sort(names)
	return names
}

// ledger is testdata/unread.json: the numbers carried and not yet read
// against their source, by name.
type ledger struct {
	Unread []string `json:"unread"`
}

// The test and anti-test: unread-source values must match the ledger exactly. A
// new value from an unread source must enter the ledger; a newly verified
// source must remove its values. Neither transition can hide in an overlooked
// comment.
func TestTheUnreadNumbersAreTheLedgers(test *testing.T) {
	raw, err := os.ReadFile("testdata/unread.json")
	if err != nil {
		test.Fatal(err)
	}
	var want ledger
	if err := json.Unmarshal(raw, &want); err != nil {
		test.Fatal(err)
	}
	got := unread(test)
	slices.Sort(want.Unread)
	if !slices.Equal(got, want.Unread) {
		test.Errorf("the numbers on unread sources are %q; the ledger "+
			"says %q", got, want.Unread)
	}
}

// data is the research data, or the test stops.
func data(test *testing.T) numbers.Numbers {
	test.Helper()
	read, err := research.Read()
	if err != nil {
		test.Fatal(err)
	}
	return read
}

// Every measured number is from a source sources.yaml holds, and every
// picked one says who picked it and names no source.
func TestEveryNumberSaysWhereItCameFrom(test *testing.T) {
	keys := sourceKeys(test)
	for _, number := range data(test) {
		switch number.Kind {
		case numbers.Measured:
			if !keys[number.From.Key] {
				test.Errorf("%s names source %q, which sources.yaml "+
					"does not hold", number.Name, number.From.Key)
			}
		case numbers.Picked:
			if number.By == "" || number.From != nil {
				test.Errorf("%s is picked, by %q, from %v", number.Name,
					number.By, number.From)
			}
		default:
			test.Errorf("%s is %s, neither measured nor picked",
				number.Name, number.Kind)
		}
	}
}

// The anti-test: a source no number reads from is one nobody needs, and
// the list of sources says only what is used.
func TestNoSourceGoesUncited(test *testing.T) {
	used := map[string]bool{}
	for _, number := range data(test) {
		if number.From != nil {
			used[number.From.Key] = true
		}
	}
	for key := range sourceKeys(test) {
		if !used[key] {
			test.Errorf("sources.yaml holds %s, which no number reads", key)
		}
	}
}

// A number asked for by name is the number the data holds.
func TestValueIsTheNumberTheDataHolds(test *testing.T) {
	for _, number := range data(test) {
		if got := research.Value(number.Name); got != number.Value {
			test.Errorf("%s is %v, and the data holds %v", number.Name,
				got, number.Value)
		}
	}
}

// unheld is a name the data does not hold.
const unheld = "no such number"

// The anti-test: a name the data does not hold stops the program, and
// says which name.
func TestANameTheDataDoesNotHoldStopsTheProgram(test *testing.T) {
	defer stopped(test, unheld)
	research.Value(unheld)
	test.Errorf("asking for %q went on", unheld)
}

// stopped checks the program stopped, saying the name it was asked for.
func stopped(test *testing.T, name string) {
	test.Helper()
	said := recover()
	if said == nil {
		test.Errorf("asking for %q did not stop the program", name)
		return
	}
	if !strings.Contains(fmt.Sprint(said), name) {
		test.Errorf("asking for %q stopped with %v, not naming it", name,
			said)
	}
}
