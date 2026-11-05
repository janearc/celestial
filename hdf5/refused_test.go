package hdf5_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/janearc/celestial/hdf5"
	"pgregory.net/rapid"
)

// halved is how much of the file a truncated copy keeps. Defined: half.
const halved = 2

// opening is what NewFile makes of some bytes, whole.
func opening(raw []byte) (*hdf5.File, error) {
	size := int64(len(raw))
	return hdf5.NewFile(bytes.NewReader(raw), size)
}

// The words each cause puts in a refusal, as hdf5 writes them.
const (
	damaged     = "hdf5: file is damaged"
	notRead     = "hdf5: part of the format not read"
	notHDF5     = "hdf5: not an HDF5 file"
	notFound    = "hdf5: not found"
	notThatKind = "hdf5: not that kind of value"
)

// causes is the words each mutation's cause puts in its refusal.
var causes = map[string]string{"unsupported": notRead, "corrupt": damaged}

// says is whether an error is one and holds the words.
func says(err error, words string) bool {
	return err != nil && strings.Contains(err.Error(), words)
}

// mutated is the strain file with a mutation's bytes written over it.
func mutated(test *testing.T, want mutation) []byte {
	test.Helper()
	raw := bytes.Clone(fileBytes(test))
	for _, write := range want.Changes {
		copy(raw[write.At:], write.Bytes)
	}
	return raw
}

// refusal is what reading the real file, mutated, the way the mutation
// says gives: whether it gave a file, members or values, and its error.
func refusal(test *testing.T, want mutation) (bool, error) {
	test.Helper()
	return refusalOf(test, mutated(test, want), want)
}

// refusalOf is what reading changed bytes the way a mutation says gives.
func refusalOf(test *testing.T, raw []byte, want mutation) (bool, error) {
	test.Helper()
	if want.Through == "open" {
		file, err := opening(raw)
		return file != nil, err
	}
	file := opened(test, raw)
	if want.Through == "members" {
		names, err := file.Members(want.Path)
		return names != nil, err
	}
	values, err := file.Read(want.Path)
	return values != nil, err
}

// The anti-test: each field of the real file changed to a value the
// reader does not read, or that contradicts the file, is refused where
// the reader meets it, for the cause the expectations name, in words
// that name the address and the specification's section; and gives
// nothing.
func TestAChangedFileIsRefused(test *testing.T) {
	for _, want := range expectations(test).Mutations {
		gave, err := refusal(test, want)
		where := fmt.Sprintf("at address %#x (spec %s)", want.Address,
			want.Section)
		if gave || !says(err, causes[want.Cause]) ||
			!strings.Contains(err.Error(), where) {
			test.Errorf("%s: gave %v, %v", want.Why, gave, err)
		}
	}
}

// The test: the real file's superblock alone, its end of file address
// written as its own length, opens: opening reads the superblock and no
// further, though its root group's header, at 96, is past the end.
func TestASuperblockAloneOpens(test *testing.T) {
	super := expectations(test).Superblock
	raw := bytes.Clone(fileBytes(test)[:super.Length])
	binary.LittleEndian.PutUint64(raw[super.EndAt:], uint64(super.Length))
	if file, err := opening(raw); err != nil || file == nil {
		test.Errorf("the superblock alone: %v, %v", file, err)
	}
}

// The anti-test: the superblock a byte short, its end of file address
// written as that length, is refused as damaged, and gives no file.
func TestASuperblockAByteShortIsRefused(test *testing.T) {
	super := expectations(test).Superblock
	short := super.Length - one
	raw := bytes.Clone(fileBytes(test)[:short])
	binary.LittleEndian.PutUint64(raw[super.EndAt:], uint64(short))
	if file, err := opening(raw); file != nil ||
		!says(err, damaged) {
		test.Errorf("the superblock a byte short: %v, %v", file, err)
	}
}

// The property: the real file with any one byte changed opens or is
// refused, and every read of it gives what it reads or refuses and gives
// nothing; none panics.
func TestAnyByteOfTheFileChangedNeverPanics(test *testing.T) {
	want := expectations(test)
	rapid.Check(test, changing{raw: fileBytes(test), strain: want.Strain,
		reads: want.EveryRead}.check)
}

// changing changes a drawn byte of the real file and reads it.
type changing struct {
	raw    []byte
	strain string
	reads  reads
}

// check draws the byte and its new value, then reads all it can.
func (changes changing) check(test *rapid.T) {
	raw := bytes.Clone(changes.raw)
	at := rapid.IntRange(firstValue, len(raw)-one).Draw(test, "at")
	raw[at] = rapid.Byte().Draw(test, "byte")
	file, err := opening(raw)
	if (file == nil) == (err == nil) {
		test.Fatalf("opened: %v, %v", file, err)
	}
	if err != nil {
		return
	}
	for _, path := range changes.reads.Members {
		names, err := file.Members(path)
		either(test, path, names, err)
	}
	for _, path := range changes.reads.Datasets {
		values, err := file.Read(path)
		readEvery(test, path, values, err)
	}
	for _, name := range changes.reads.Attributes {
		values, err := file.Attribute(changes.strain, name)
		readEvery(test, name, values, err)
	}
}

// readEvery holds a read to giving values or refusing, and its values,
// read as every kind, to the same.
func readEvery(test *rapid.T, what string, values *hdf5.Values,
	err error) {
	if (values == nil) == (err == nil) {
		test.Fatalf("%s: %v, %v", what, values, err)
	}
	if err != nil {
		return
	}
	floats, err := values.Floats()
	either(test, what+" as floats", floats, err)
	integers, err := values.Integers()
	either(test, what+" as integers", integers, err)
	texts, err := values.Texts()
	either(test, what+" as texts", texts, err)
}

// either fails a draw whose read gave both a result and an error, or
// neither.
func either[Element any](test *rapid.T, what string, got []Element,
	err error) {
	if (got == nil) == (err == nil) {
		test.Fatalf("%s: gave %v and %v", what, got, err)
	}
}

// stingy is a reader that gives no bytes and says nothing of why.
type stingy struct{}

// ReadAt gives nothing, and no error.
func (stingy) ReadAt([]byte, int64) (int, error) { return noneRead, nil }

// broken is a reader that fails.
type broken struct{}

// ReadAt gives nothing, and its failure.
func (broken) ReadAt([]byte, int64) (int, error) {
	return noneRead, errBroken
}

// noneRead is the bytes a reader that gives nothing gives. Defined.
const noneRead = 0

// errBroken is a reader's own failure.
var errBroken = errors.New("the reader is broken")

// The anti-test: a reader that gives fewer bytes than are asked for is
// its own error, the file cut short, and one that fails is its failure;
// neither is a refusal of the file, and neither gives a file.
func TestAFailingReaderIsItsOwnError(test *testing.T) {
	raw := fileBytes(test)
	size := int64(len(raw))
	if file, err := hdf5.NewFile(stingy{}, size); file != nil ||
		!errors.Is(err, io.ErrUnexpectedEOF) ||
		says(err, damaged) {
		test.Errorf("a stingy reader: %v, %v", file, err)
	}
	if file, err := hdf5.NewFile(broken{}, size); file != nil ||
		!errors.Is(err, errBroken) {
		test.Errorf("a broken reader: %v, %v", file, err)
	}
}

// The anti-test: non-HDF5 bytes, an empty file, a truncated file, and a
// negative size must return an error and no File.
func TestWhatIsNoFileIsRefused(test *testing.T) {
	want := expectations(test)
	if file, err := opening(nil); !says(err, notHDF5) ||
		file != nil {
		test.Errorf("no bytes read as a file: %v, %v", file, err)
	}
	words := []byte(want.NotHDF5)
	if file, err := opening(words); !says(err, notHDF5) ||
		file != nil {
		test.Errorf("words read as a file: %v, %v", file, err)
	}
	short := []byte(want.TooShort)
	if file, err := opening(short); !says(err, notHDF5) ||
		file != nil {
		test.Errorf("%q read as a file: %v, %v", short, file, err)
	}
	raw := fileBytes(test)
	half := raw[:len(raw)/halved]
	if file, err := opening(half); !says(err, damaged) ||
		file != nil {
		test.Errorf("half a file read as one: %v, %v", file, err)
	}
	signed := raw[:want.Signature]
	if file, err := opening(signed); !says(err, damaged) ||
		file != nil {
		test.Errorf("a signature alone read as a file: %v, %v", file, err)
	}
	if file, err := hdf5.NewFile(bytes.NewReader(raw),
		-int64(one)); err == nil || file != nil {
		test.Errorf("a file of less than no bytes read: %v", file)
	}
}

// The anti-test: a byte changed inside the strain's last chunk, a
// hundred bytes from the file's end, is found by the chunk's checksum or
// by its inflation; the strain is refused as damaged, and gives no
// values. The rest of the file still reads.
func TestADamagedChunkIsRefused(test *testing.T) {
	want := expectations(test)
	raw := bytes.Clone(fileBytes(test))
	raw[len(raw)-want.FlippedFrom] ^= flipped
	file := opened(test, raw)
	if values, err := file.Read(want.Strain); !says(err, damaged) ||
		values != nil {
		test.Errorf("the damaged strain read: %v, %v", values, err)
	}
	read(test, file, "meta/Detector")
}

// flipped is every bit of a byte, turned. Defined.
const flipped = 0xff

// The anti-test: a path or an attribute the file does not have is not
// found, and gives no values.
func TestWhatIsNotThereIsNotFound(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	for _, path := range want.NotPaths {
		if values, err := file.Read(path); !says(err, notFound) ||
			values != nil {
			test.Errorf("%q read: %v, %v", path, values, err)
		}
		if names, err := file.Members(path); err == nil || names != nil {
			test.Errorf("%q has members: %v", path, names)
		}
		if values, err := file.Attribute(path, "Xstart"); err == nil ||
			values != nil {
			test.Errorf("%q has an Xstart: %v", path, values)
		}
	}
	for _, name := range want.NotAttrs {
		if values, err := file.Attribute(want.Strain, name); !says(err,
			notFound) || values != nil {
			test.Errorf("the strain's %q read: %v, %v", name, values, err)
		}
	}
}

// The anti-test: values read as a kind they are not are refused, and a
// group is no dataset, nor a dataset a group.
func TestTheWrongKindIsRefused(test *testing.T) {
	want := expectations(test)
	file := opened(test, fileBytes(test))
	names := read(test, file, "quality/simple/DQShortnames")
	if got, err := names.Floats(); !says(err, notThatKind) ||
		got != nil {
		test.Errorf("names read as floats: %v", got)
	}
	strain := read(test, file, want.Strain)
	if got, err := strain.Texts(); !says(err, notThatKind) ||
		got != nil {
		test.Errorf("the strain read as text: %v", got)
	}
	if got, err := strain.Integers(); !says(err, notThatKind) ||
		got != nil {
		test.Errorf("the strain read as integers: %v", got)
	}
	if values, err := file.Read("meta"); !says(err, notThatKind) ||
		values != nil {
		test.Errorf("a group read as a dataset: %v, %v", values, err)
	}
	if got, err := file.Members(want.Strain); err == nil || got != nil {
		test.Errorf("the strain has members: %v", got)
	}
}
