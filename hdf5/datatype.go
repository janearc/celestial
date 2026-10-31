package hdf5

import "fmt"

// typeClass is the kind of value a datatype describes.
type typeClass int

// The classes this reader reads.
const (
	// integerClass is a fixed-point number, signed or not.
	integerClass typeClass = iota
	// floatClass is an IEEE 754 binary32 or binary64 number.
	floatClass
	// stringClass is text, of fixed length or of variable length.
	stringClass
)

// The class numbers of the Datatype message, in its first four bits,
// each defined by IV.A.2.d.
const (
	classFixedPoint     = 0  // defined by IV.A.2.d
	classFloatingPoint  = 1  // defined by IV.A.2.d
	classTime           = 2  // defined by IV.A.2.d
	classString         = 3  // defined by IV.A.2.d
	classBitField       = 4  // defined by IV.A.2.d
	classOpaque         = 5  // defined by IV.A.2.d
	classCompound       = 6  // defined by IV.A.2.d
	classReference      = 7  // defined by IV.A.2.d
	classEnumeration    = 8  // defined by IV.A.2.d
	classVariableLength = 9  // defined by IV.A.2.d
	classArray          = 10 // defined by IV.A.2.d
)

// versionOneDatatype is the version of the Datatype message that is read.
// Defined by IV.A.2.d; versions 2 and 3 differ for compound and array
// types, which are not read, and the files read here write version 1.
const versionOneDatatype = 1

// datatype is the datatype of a dataset's or an attribute's elements.
type datatype struct {
	// class is what kind of value an element is.
	class typeClass
	// size is the bytes in one element as stored. For a variable-length
	// string it is the size of the reference to the text, not the text.
	size int
	// signed says whether an integer has a sign.
	signed bool
	// variable says whether a string is variable-length: the text is not
	// in the element but in the file's global heap.
	variable bool
	// padding says how a fixed string fills its size, IV.A.2.d String.
	padding stringPadding
}

// stringPadding is how a string ends within its size, IV.A.2.d String
// and Variable-Length classes: null-terminated, null-padded, or
// space-padded, in the order the specification numbers them.
type stringPadding int

const (
	nullTerminated stringPadding = iota
	nullPadded
	spacePadded
	paddingKinds
)

// The fields of the Datatype message's header and of its fixed-point
// bit field, each defined by IV.A.2.d.
const (
	classBits          = 0x0f // defined: the class is the low four bits
	versionShift       = 4    // defined: the version is the high four
	classFieldBytes    = 3    // defined: the class bit field's bytes
	datatypeHeaderSize = 8    // defined: class, version, bit field, size
	bigEndianBit       = 0x01 // defined: bit 0 of the fixed-point field
	signedBit          = 0x08 // defined: bit 3 of the fixed-point field
	firstField         = 0    // defined: the bit field's first byte
	noBitOffset        = 0    // defined: an integer that starts at bit 0
	bitsPerByte        = 8    // defined: precision is counted in bits
)

// readDatatype parses a Datatype message at an address. It refuses a
// message too short for its header, a version not read, and a class not
// read, and then gives none.
func readDatatype(data []byte, at uint64) (*datatype, error) {
	walk := &cursor{data: data}
	head := walk.u8()
	class, version := int(head&classBits), int(head>>versionShift)
	bits := walk.take(classFieldBytes)
	size := int(walk.u32())
	if walk.short {
		return nil, refuse(errCorrupt, at, "IV.A.2.d",
			"datatype message of %d bytes is shorter than its "+
				"header", len(data))
	}
	if version != versionOneDatatype {
		return nil, refuse(errUnsupported, at, "IV.A.2.d",
			"datatype version %d, only version 1 is read", version)
	}
	props := data[datatypeHeaderSize:]
	switch class {
	case classFixedPoint:
		return fixedPoint(bits, size, props, at)
	case classFloatingPoint:
		return floatingPoint(bits, size, props, at)
	case classString:
		return fixedString(bits, size, at)
	case classVariableLength:
		return variableLength(bits, size, props, at)
	}
	return nil, refuse(errUnsupported, at, "IV.A.2.d",
		"datatype class %d (%s), only fixed-point, floating-point and "+
			"string are read", class, className(class))
}

// classNames is the specification's name for each class number refused.
var classNames = map[int]string{
	classTime: "time", classBitField: "bit field", classOpaque: "opaque",
	classCompound: "compound", classReference: "reference",
	classEnumeration: "enumeration", classArray: "array",
}

// className is the specification's name for a datatype class number.
func className(class int) string {
	if name, held := classNames[class]; held {
		return name
	}
	return "not defined"
}

// fixedPoint is the datatype of a fixed-point datatype: little-endian,
// with every bit of its size used. IV.A.2.d Fixed-Point. It refuses any
// other, and then gives none.
func fixedPoint(bits []byte, size int, props []byte,
	at uint64) (*datatype, error) {
	if bits[firstField]&bigEndianBit != noBitOffset {
		return nil, refuse(errUnsupported, at, "IV.A.2.d",
			"a big-endian integer, only little-endian is read")
	}
	walk := &cursor{data: props}
	offset, precision := int(walk.u16()), int(walk.u16())
	if walk.short || offset != noBitOffset ||
		precision != size*bitsPerByte || !sizeFits(size) {
		return nil, refuse(errUnsupported, at, "IV.A.2.d",
			"an integer of %d bytes with bit offset %d and "+
				"precision %d: only whole 1, 2, 4 or 8 byte "+
				"integers are read", size, offset, precision)
	}
	signed := bits[firstField]&signedBit != noBitOffset
	return &datatype{class: integerClass, size: size, signed: signed}, nil
}

// sizeFits says whether a count is the size in bytes of an integer the
// reader has code for.
func sizeFits(count int) bool {
	return count == byteWide || count == twoWide || count == fourWide ||
		count == eightWide
}

// String is a description of a datatype for a refusal.
func (kind datatype) String() string {
	return fmt.Sprintf("class %d size %d", kind.class, kind.size)
}
