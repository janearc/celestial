package hdf5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// values is the elements of a dataset or an attribute as stored, with
// their type: little-endian, in the C order of the file.
type values struct {
	typ datatype
	raw []byte
}

// wrongType is the error for a read of a kind of value the elements are
// not.
func (held values) wrongType(want string) error {
	return fmt.Errorf("%w: %s is not %s", errWrongType, held.typ, want)
}

// element is the bytes of an element, by its place.
func (held values) element(place int) []byte {
	size := held.typ.size
	return held.raw[place*size : (place+oneElement)*size]
}

// count is how many elements there are.
func (held values) count() int { return len(held.raw) / held.typ.size }

// float64s is the elements as float64; a binary32 widens exactly. It
// refuses elements that are not floating-point, and then gives none.
func (held values) float64s() ([]float64, error) {
	if held.typ.class != floatClass {
		return nil, held.wrongType("a floating-point type")
	}
	out := make([]float64, held.count())
	for place := range out {
		bits := held.element(place)
		if held.typ.size == eightWide {
			word := binary.LittleEndian.Uint64(bits)
			out[place] = math.Float64frombits(word)
			continue
		}
		word := binary.LittleEndian.Uint32(bits)
		out[place] = float64(math.Float32frombits(word))
	}
	return out, nil
}

// The parts of a two's complement integer's sign, each defined by it.
const (
	signBit  = 0x80 // defined: the top bit of a byte
	allOnes  = ^uint64(pastEnd)
	lastByte = 1 // defined: the top byte is size less one
)

// integer is an element as an integer, widened to 64 bits: sign extended
// if the type is signed. The type is an integer of a size the datatype
// reader accepts.
func (held values) integer(place int) uint64 {
	var word [eightWide]byte
	size := held.typ.size
	copy(word[:], held.element(place))
	got := binary.LittleEndian.Uint64(word[:])
	if held.typ.signed && word[size-lastByte]&signBit != noFlag {
		got |= allOnes << (uint(size) * bitsPerByte)
	}
	return got
}

// int64s is the elements as int64; unsigned elements must fit. It
// refuses elements that are not integers, and an unsigned one past an
// int64, and then gives none.
func (held values) int64s() ([]int64, error) {
	if held.typ.class != integerClass {
		return nil, held.wrongType("an integer type")
	}
	out := make([]int64, held.count())
	for place := range out {
		got := held.integer(place)
		if !held.typ.signed && got > math.MaxInt64 {
			return nil, fmt.Errorf("%w: element %d, %d, is more "+
				"than an int64 holds", errWrongType, place, got)
		}
		out[place] = int64(got)
	}
	return out, nil
}

// strings is the elements as strings. A fixed string is cut where its
// padding says; a variable-length string is read from the global heap.
// It refuses elements that are not strings, and then gives none.
func (held values) strings(from *source) ([]string, error) {
	if held.typ.class != stringClass {
		return nil, held.wrongType("a string type")
	}
	out := make([]string, held.count())
	heap := newGlobalHeap(from)
	for place := range out {
		element := held.element(place)
		if !held.typ.variable {
			out[place] = held.typ.fixedText(element)
			continue
		}
		text, err := variableString(heap, element)
		if err != nil {
			return nil, err
		}
		out[place] = text
	}
	return out, nil
}

// spaces is the byte a space-padded string pads with. Defined by
// IV.A.2.d String.
const spaces = " "

// fixedText is the text in a fixed-length string element: up to the
// first null, or without the spaces that pad it. IV.A.2.d String.
func (kind datatype) fixedText(element []byte) string {
	if kind.padding == spacePadded {
		return string(bytes.TrimRight(element, spaces))
	}
	if end := bytes.IndexByte(element, nameEnd); end != notFound {
		element = element[:end]
	}
	return string(element)
}

// noText is a variable-length string of no bytes. Defined by III.E.
const noText = 0

// variableString is the text a variable-length element refers to: its
// length in bytes, the address of a global heap collection and the index
// of the object in it. IV.A.2.d Variable-Length, III.E.
func variableString(heap *globalHeap, element []byte) (string, error) {
	walk := &cursor{data: element}
	length, collectionAt := uint64(walk.u32()), walk.u64()
	index := walk.u32()
	if length == noText {
		return "", nil
	}
	object, err := heap.object(collectionAt, index)
	if err != nil {
		return "", err
	}
	if length > uint64(len(object)) {
		return "", refuse(errCorrupt, collectionAt, "III.E",
			"a string of %d bytes in an object of %d", length,
			len(object))
	}
	return string(object[:length]), nil
}
