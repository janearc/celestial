package hdf5_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// built is what testdata/whole.json holds: a whole file built by hand,
// and what reading it must give.
type built struct {
	Bytes      []string `json:"bytes"`
	Size       int      `json:"size"`
	Root       []string `json:"root"`
	Dataset    string   `json:"dataset"`
	Elements   []int64  `json:"elements"`
	Shape      []uint64 `json:"shape"`
	Attribute  string   `json:"attribute"`
	Text       []string `json:"attribute text"`
	EmptyGroup string   `json:"empty group"`
	NotPaths   []string `json:"not paths"`
	Heap       mutation `json:"heap"`
}

// builtFile is testdata/whole.json, or the test stops.
func builtFile(test *testing.T) built {
	test.Helper()
	var whole built
	raw, err := os.ReadFile("testdata/whole.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &whole); err != nil {
		test.Fatal(err)
	}
	return whole
}

// builtBytes is the built file's bytes, or the test stops.
func builtBytes(test *testing.T, whole built) []byte {
	test.Helper()
	pairs := strings.ReplaceAll(strings.Join(whole.Bytes, " "), " ", "")
	raw, err := hex.DecodeString(pairs)
	if err != nil {
		test.Fatal(err)
	}
	if len(raw) != whole.Size {
		test.Fatalf("the built file is %d bytes, not %d", len(raw),
			whole.Size)
	}
	return raw
}

// The built file reads as it was built: the root's two members, the
// dataset's two elements, its attribute's text, and a group of none.
func TestTheBuiltFileReadsAsBuilt(test *testing.T) {
	whole := builtFile(test)
	file := opened(test, builtBytes(test, whole))
	if got, err := file.Members(""); err != nil ||
		!slices.Equal(got, whole.Root) {
		test.Errorf("the root holds %v, %v", got, err)
	}
	data := read(test, file, whole.Dataset)
	if got, err := data.Integers(); err != nil ||
		!slices.Equal(got, whole.Elements) {
		test.Errorf("%s is %v, %v", whole.Dataset, got, err)
	}
	if got := data.Shape(); !slices.Equal(got, whole.Shape) {
		test.Errorf("%s's shape is %v", whole.Dataset, got)
	}
	unit := attribute(test, file, whole.Dataset, whole.Attribute)
	if got, err := unit.Texts(); err != nil || !slices.Equal(got,
		whole.Text) {
		test.Errorf("%s's %s is %v, %v", whole.Dataset, whole.Attribute,
			got, err)
	}
	if got, err := file.Members(whole.EmptyGroup); err != nil ||
		got == nil || len(got) != scalar {
		test.Errorf("%s holds %v, %v", whole.EmptyGroup, got, err)
	}
}

// The anti-test: a path the built file does not have is not found, an
// attribute it does not have is not found, a group is no dataset nor a
// dataset a group, and none gives anything.
func TestWhatTheBuiltFileLacksIsRefused(test *testing.T) {
	whole := builtFile(test)
	file := opened(test, builtBytes(test, whole))
	for _, path := range whole.NotPaths {
		if values, err := file.Read(path); !says(err, notFound) || values != nil {
			test.Errorf("%q read: %v, %v", path, values, err)
		}
	}
	if values, err := file.Attribute(whole.Dataset,
		whole.NotPaths[firstValue]); !says(err, notFound) ||
		values != nil {
		test.Errorf("an attribute that is not there: %v, %v", values, err)
	}
	for _, path := range whole.NotPaths {
		if values, err := file.Attribute(path,
			whole.Attribute); !says(err, notFound) ||
			values != nil {
			test.Errorf("%q has a %s: %v, %v", path, whole.Attribute,
				values, err)
		}
	}
	if values, err := file.Read(whole.EmptyGroup); !says(err, notThatKind) || values != nil {
		test.Errorf("a group read as a dataset: %v, %v", values, err)
	}
	if got, err := read(test, file, whole.Dataset).Floats(); !says(err, notThatKind) || got != nil {
		test.Errorf("bytes read as floats: %v, %v", got, err)
	}
	unit := attribute(test, file, whole.Dataset, whole.Attribute)
	if got, err := unit.Integers(); !says(err, notThatKind) || got != nil {
		test.Errorf("text read as integers: %v, %v", got, err)
	}
	if names, err := file.Members(whole.Dataset); err == nil ||
		names != nil {
		test.Errorf("a dataset has members: %v", names)
	}
}

// The anti-test: the real file's mutations, at the same offsets, refuse
// the built file where the reader meets them, for the same causes in
// words that name the same sections. A mutation past the built file's
// end, the real file's heap at 680, is the built file's heap at 272
// instead.
func TestAChangedBuiltFileIsRefused(test *testing.T) {
	whole := builtFile(test)
	raw := builtBytes(test, whole)
	for _, want := range append(expectations(test).Mutations, whole.Heap) {
		if past(want, len(raw)) {
			continue
		}
		changed := bytes.Clone(raw)
		for _, write := range want.Changes {
			copy(changed[write.At:], write.Bytes)
		}
		gave, err := refusalOf(test, changed, want)
		if gave || !says(err, causes[want.Cause]) ||
			!strings.Contains(err.Error(), want.Section) {
			test.Errorf("%s: gave %v, %v", want.Why, gave, err)
		}
	}
}

// past is whether any of a mutation's changes lands past a file's end.
func past(want mutation, size int) bool {
	for _, write := range want.Changes {
		if write.At+len(write.Bytes) > size {
			return true
		}
	}
	return false
}
