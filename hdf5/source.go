package hdf5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// undefinedAddress is the address that means "none": every bit set.
// Defined by the specification's II.A, Format Signature and Superblock,
// which names it the undefined address.
const undefinedAddress = math.MaxUint64

// source bounds io.ReaderAt access by the file’s declared size, preventing
// reads outside the file or larger than it.
type source struct {
	reader io.ReaderAt
	size   uint64
}

// read fetches a bounded byte range. Out-of-file ranges are corrupt; errors
// from the underlying reader retain their own cause.
func (from *source) read(at, count uint64, section string) ([]byte,
	error) {
	if at > from.size || count > from.size-at {
		return nil, fmt.Errorf("hdf5: file is damaged: %d bytes "+
			"wanted, but the file is %d bytes long, at address "+
			"%#x (spec %s)", count, from.size, at, section)
	}
	got := make([]byte, count)
	read, err := from.reader.ReadAt(got, int64(at))
	if read == len(got) {
		return got, nil
	}
	if err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, fmt.Errorf("hdf5: reading %d bytes at %#x: %w", count, at,
		err)
}

// cursor decodes previously read little-endian bytes. An overrun returns zero
// and sets short, which the caller checks after parsing a structure.
type cursor struct {
	data  []byte
	at    int
	short bool
}

// The widths of the fields a cursor reads, in bytes, and what a field
// reads as past the end. Each is defined by the field's type.
const (
	byteWide  = 1 // defined: a uint8
	twoWide   = 2 // defined: a uint16
	fourWide  = 4 // defined: a uint32
	eightWide = 8 // defined: a uint64
	noBytes   = 0 // defined: a count of nothing
	pastEnd   = 0 // defined: what a field past the end reads as
)

// take returns the next count bytes, or nothing if too few remain.
func (walk *cursor) take(count int) []byte {
	if count < noBytes || count > len(walk.data)-walk.at {
		walk.short = true
		walk.at = len(walk.data)
		return nil
	}
	got := walk.data[walk.at : walk.at+count]
	walk.at += count
	return got
}

// opens checks the next bytes against a structure signature.
func (walk *cursor) opens(signature []byte) bool {
	got := walk.take(len(signature))
	return bytes.Equal(got, signature)
}

// skip advances by a byte count.
func (walk *cursor) skip(count int) { walk.take(count) }

// u8 is the next byte.
func (walk *cursor) u8() uint8 {
	got := walk.take(byteWide)
	if got == nil {
		return pastEnd
	}
	return got[noBytes]
}

// u16 is the next two bytes as a number.
func (walk *cursor) u16() uint16 {
	got := walk.take(twoWide)
	if got == nil {
		return pastEnd
	}
	return binary.LittleEndian.Uint16(got)
}

// u32 is the next four bytes as a number.
func (walk *cursor) u32() uint32 {
	got := walk.take(fourWide)
	if got == nil {
		return pastEnd
	}
	return binary.LittleEndian.Uint32(got)
}

// u64 is the next eight bytes as a number.
func (walk *cursor) u64() uint64 {
	got := walk.take(eightWide)
	if got == nil {
		return pastEnd
	}
	return binary.LittleEndian.Uint64(got)
}

// align advances to a multiple of the requested byte alignment.
func (walk *cursor) align(count int) {
	if rest := walk.at % count; rest != noBytes {
		walk.skip(count - rest)
	}
}
