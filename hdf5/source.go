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

// source is the file's bytes behind an io.ReaderAt, with the file's size,
// so that no read goes past the end and no read is asked for more than
// the file holds.
type source struct {
	reader io.ReaderAt
	size   uint64
}

// read is so many bytes at an address. A range that is not wholly inside
// the file is a damaged file; a reader that fails is its own error.
func (from *source) read(at, count uint64, section string) ([]byte,
	error) {
	if at > from.size || count > from.size-at {
		return nil, refuse(errCorrupt, at, section,
			"%d bytes wanted, but the file is %d bytes long", count,
			from.size)
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

// cursor walks forward through bytes already read, little-endian, which
// is the byte order of every field in the specification. A read past the
// end gives zero and sets short; a caller checks short once, after the
// structure.
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

// take is the next so many bytes, or nothing if fewer remain.
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

// opens is whether the next bytes are a signature, the four or eight
// letters each of the specification's structures starts with.
func (walk *cursor) opens(signature []byte) bool {
	got := walk.take(len(signature))
	return bytes.Equal(got, signature)
}

// skip passes over so many bytes.
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

// align passes over bytes until the position is a multiple of a count.
func (walk *cursor) align(count int) {
	if rest := walk.at % count; rest != noBytes {
		walk.skip(count - rest)
	}
}
