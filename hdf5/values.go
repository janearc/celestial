package hdf5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// values stores typed element bytes in little-endian C order.
type values struct {
	typ datatype
	raw []byte
}

// element is the bytes of an element, by its place.
func (held values) element(place int) []byte {
	size := held.typ.size
	return held.raw[place*size : (place+oneElement)*size]
}

// count is how many elements there are.
func (held values) count() int { return len(held.raw) / held.typ.size }

// float64s decodes floats in file order; binary32 widens exactly.
// Non-floating-point elements return an error.
func (held values) float64s() ([]float64, error) {
	if held.typ.class != floatClass {
		return nil, fmt.Errorf("hdf5: not that kind of value: %s is "+
			"not a floating-point type", held.typ)
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

// integer widens a supported stored integer to 64 bits, sign-extending when the
// datatype is signed.
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

// int64s decodes integer elements. Other types and unsigned values above
// int64’s maximum return an error.
func (held values) int64s() ([]int64, error) {
	if held.typ.class != integerClass {
		return nil, fmt.Errorf("hdf5: not that kind of value: %s is "+
			"not an integer type", held.typ)
	}
	out := make([]int64, held.count())
	for place := range out {
		got := held.integer(place)
		if !held.typ.signed && got > math.MaxInt64 {
			return nil, fmt.Errorf("hdf5: not that kind of "+
				"value: element %d, %d, is more than an int64 "+
				"holds", place, got)
		}
		out[place] = int64(got)
	}
	return out, nil
}

// strings decodes fixed strings using their padding rule and variable strings
// through the global heap. Other element types return an error.
func (held values) strings(from *source) ([]string, error) {
	if held.typ.class != stringClass {
		return nil, fmt.Errorf("hdf5: not that kind of value: %s is "+
			"not a string type", held.typ)
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

// fixedText removes null or space padding according to the fixed-string
// datatype (IV.A.2.d, String).
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

// variableString resolves a stored byte length, global-heap address, and object
// index to text (IV.A.2.d, Variable-Length; III.E).
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
		return "", fmt.Errorf("hdf5: file is damaged: a string of %d "+
			"bytes in an object of %d, at address %#x (spec "+
			"III.E)", length, len(object), collectionAt)
	}
	return string(object[:length]), nil
}
