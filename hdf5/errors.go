package hdf5

import (
	"errors"
	"fmt"
)

// The causes of hdf5's errors, each told by a function below.
var (
	errNotHDF5     = errors.New("hdf5: not an HDF5 file")
	errUnsupported = errors.New("hdf5: part of the format not read")
	errCorrupt     = errors.New("hdf5: file is damaged")
	errNotFound    = errors.New("hdf5: not found")
	errWrongType   = errors.New("hdf5: not that kind of value")
)

// ErrNotHDF5 is the cause of a refusal when the bytes are not an HDF5
// file: the format signature is missing.
func ErrNotHDF5() error { return errNotHDF5 }

// ErrUnsupported is the cause of a refusal when the file is HDF5 but uses
// a part of the format this reader does not read. The refusal names the
// part.
func ErrUnsupported() error { return errUnsupported }

// ErrCorrupt is the cause of a refusal when the file contradicts the
// specification or itself: a field out of range, an address outside the
// file, a checksum that does not match, a stream that does not inflate.
func ErrCorrupt() error { return errCorrupt }

// ErrNotFound is the cause of an error when a path or an attribute name
// is not in the file.
func ErrNotFound() error { return errNotFound }

// ErrWrongType is the cause of an error when a read asks for a kind of
// value the object does not hold, such as text from a float dataset.
func ErrWrongType() error { return errWrongType }

// refusal is an error about one place in the file. It says what was
// found, the address of the bytes, and the section of the HDF5 File Format
// Specification, version 3.0, that describes them.
type refusal struct {
	cause   error  // errNotHDF5, errUnsupported or errCorrupt
	what    string // what was found, in words
	at      uint64 // the file address of the bytes the refusal is about
	section string // the specification section that describes them
}

// Error is the refusal as one line: the cause, what was found, where and
// which section.
func (refused *refusal) Error() string {
	return fmt.Sprintf("%v: %s, at address %#x (spec %s)", refused.cause,
		refused.what, refused.at, refused.section)
}

// Unwrap is the refusal's cause, so errors.Is finds ErrCorrupt and the
// others.
//
//ninja:vow waste errors.Is calls it through the error, which no call site names
func (refused *refusal) Unwrap() error { return refused.cause }

// refuse is a refusal with its words made from a format and arguments.
func refuse(cause error, at uint64, section, format string,
	args ...any) error {
	return &refusal{cause: cause, what: fmt.Sprintf(format, args...),
		at: at, section: section}
}
