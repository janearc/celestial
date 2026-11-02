package hdf5_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/janearc/celestial/hdf5"
)

// expected is what the tests expect of the strain file, from
// testdata/hdf5.json.
type expected struct {
	File        string     `json:"file"`
	Root        []string   `json:"root"`
	Meta        []string   `json:"meta"`
	Detector    string     `json:"detector"`
	GPSStart    int64      `json:"gps start"`
	UTCStart    string     `json:"utc start"`
	Duration    int64      `json:"duration"`
	RateHz      float64    `json:"rate hz"`
	Strain      string     `json:"strain"`
	Largest     float64    `json:"largest strain"`
	StrainSum   string     `json:"strain sha256"`
	ShortNames  []string   `json:"short names"`
	Mask        string     `json:"mask"`
	AllGood     int64      `json:"all good"`
	FlippedFrom int        `json:"flipped from end"`
	NotHDF5     string     `json:"not hdf5"`
	NotPaths    []string   `json:"not paths"`
	NotAttrs    []string   `json:"not attributes"`
	EveryRead   reads      `json:"every read"`
	Signature   int        `json:"signature length"`
	TooShort    string     `json:"shorter than a signature"`
	Superblock  alone      `json:"superblock"`
	Mutations   []mutation `json:"mutations"`
}

// alone is a superblock's length, and where its end of file address is.
type alone struct {
	Length int `json:"length"`
	EndAt  int `json:"end of file at"`
}

// mutation is bytes of the real file changed at offsets; how the file is
// then read, and at what path; and the cause, the section of the
// specification and the address its refusal must name.
type mutation struct {
	Why     string   `json:"why"`
	Changes []change `json:"changes"`
	Through string   `json:"through"`
	Path    string   `json:"path"`
	Cause   string   `json:"cause"`
	Section string   `json:"section"`
	Address uint64   `json:"address"`
}

// reads is what the property reads of the file: groups' members,
// datasets, and attributes of the strain.
type reads struct {
	Members    []string `json:"members"`
	Datasets   []string `json:"datasets"`
	Attributes []string `json:"attributes"`
}

// change is bytes written over the file at an offset.
type change struct {
	At    int    `json:"at"`
	Bytes []byte `json:"bytes"`
}

// expectations reads what the tests expect, or the test stops.
func expectations(test *testing.T) expected {
	test.Helper()
	var want expected
	raw, err := os.ReadFile("testdata/hdf5.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		test.Fatal(err)
	}
	return want
}

// fileBytes is the strain file's bytes, or the test stops.
func fileBytes(test *testing.T) []byte {
	test.Helper()
	raw, err := os.ReadFile(expectations(test).File)
	if err != nil {
		test.Fatal(err)
	}
	return raw
}

// opened is the HDF5 file of some bytes, or the test stops.
func opened(test *testing.T, raw []byte) *hdf5.File {
	test.Helper()
	size := int64(len(raw))
	file, err := hdf5.NewFile(bytes.NewReader(raw), size)
	if err != nil {
		test.Fatal(err)
	}
	return file
}

// read is the values at a path, or the test stops.
func read(test *testing.T, file *hdf5.File, path string) *hdf5.Values {
	test.Helper()
	values, err := file.Read(path)
	if err != nil {
		test.Fatal(err)
	}
	return values
}

// attribute is an attribute's values, or the test stops.
func attribute(test *testing.T, file *hdf5.File, path,
	name string) *hdf5.Values {
	test.Helper()
	values, err := file.Attribute(path, name)
	if err != nil {
		test.Fatal(err)
	}
	return values
}

// The counts and places these tests use, each defined.
const (
	scalar     = 0   // defined: a scalar has no dimensions
	one        = 1   // defined by counting
	firstValue = 0   // defined by a slice's order
	nothing    = 0.0 // defined by the scale
)

// The file's groups and their members are the centre's layout.
func TestTheGroupsAreTheCentres(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	for path, members := range map[string][]string{"/": want.Root,
		"": want.Root, "meta": want.Meta} {
		got, err := file.Members(path)
		if err != nil || !slices.Equal(got, members) {
			test.Errorf("%q holds %v, %v; not %v", path, got, err, members)
		}
	}
}

// texts is a scalar text's value, or the test stops.
func text(test *testing.T, values *hdf5.Values) string {
	test.Helper()
	got, err := values.Texts()
	if err != nil || len(got) != one {
		test.Fatalf("texts %v, %v", got, err)
	}
	return got[firstValue]
}

// integer is a scalar integer's value, or the test stops.
func integer(test *testing.T, values *hdf5.Values) int64 {
	test.Helper()
	got, err := values.Integers()
	if err != nil || len(got) != one {
		test.Fatalf("integers %v, %v", got, err)
	}
	return got[firstValue]
}

// The file says what its name says: H1, from GPS 1126259447, for 32 seconds;
// and its UTC start is that GPS time's, 17 s earlier.
func TestTheMetadataIsTheFilesName(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	if got := text(test, read(test, file, "meta/Detector")); got !=
		want.Detector {
		test.Errorf("the detector is %q", got)
	}
	if got := integer(test, read(test, file, "meta/GPSstart")); got !=
		want.GPSStart {
		test.Errorf("the gps start is %d", got)
	}
	if got := integer(test, read(test, file, "meta/Duration")); got !=
		want.Duration {
		test.Errorf("the duration is %d", got)
	}
	if got := text(test, read(test, file, "meta/UTCstart")); got !=
		want.UTCStart {
		test.Errorf("the utc start is %q", got)
	}
}

// The strain is the duration times the rate in samples, as its own attributes
// say: from its GPS start, a sample every 1/4096 s; each is a number, and none
// is larger than a strain can be.
func TestTheStrainIsWhole(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	strain := read(test, file, want.Strain)
	samples, err := strain.Floats()
	if err != nil {
		test.Fatal(err)
	}
	count := int64(float64(want.Duration) * want.RateHz)
	shape := strain.Shape()
	if len(shape) != one || int64(shape[firstValue]) != count ||
		int64(len(samples)) != count {
		test.Errorf("the strain is %v, %d samples, not %d", shape,
			len(samples), count)
	}
	if got := integer(test, attribute(test, file, want.Strain,
		"Npoints")); got != count {
		test.Errorf("Npoints is %d", got)
	}
	if got := integer(test, attribute(test, file, want.Strain,
		"Xstart")); got != want.GPSStart {
		test.Errorf("Xstart is %d", got)
	}
	spacing, err := attribute(test, file, want.Strain, "Xspacing").Floats()
	if err != nil || len(spacing) != one ||
		spacing[firstValue] != one/want.RateHz {
		test.Errorf("Xspacing is %v, %v", spacing, err)
	}
	for place, sample := range samples {
		if math.IsNaN(sample) || !(math.Abs(sample) <= want.Largest) {
			test.Fatalf("sample %d is %v", place, sample)
		}
	}
	if got := sumOf(samples); got != want.StrainSum {
		test.Errorf("the strain's samples sum to sha256 %s, not %s", got,
			want.StrainSum)
	}
}

// sumOf is the sha256 of samples' bits, each little-endian, in order.
func sumOf(samples []float64) string {
	sum := sha256.New()
	for _, sample := range samples {
		bits := math.Float64bits(sample)
		sum.Write(binary.LittleEndian.AppendUint64(nil, bits))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// The data quality's names are the seven flags, and every second of the
// 32 has all seven set.
func TestTheDataQualityIsAllGood(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	names, err := read(test, file, "quality/simple/DQShortnames").Texts()
	if err != nil || !slices.Equal(names, want.ShortNames) {
		test.Errorf("the flags are %v, %v", names, err)
	}
	mask, err := read(test, file, want.Mask).Integers()
	if err != nil || int64(len(mask)) != want.Duration {
		test.Fatalf("the mask is %d seconds, %v", len(mask), err)
	}
	for second, flags := range mask {
		if flags != want.AllGood {
			test.Errorf("second %d has flags %b", second, flags)
		}
	}
}
