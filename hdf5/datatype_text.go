package hdf5

// The fields of a floating-point datatype's class bit field and its
// properties, each defined by IV.A.2.d Floating-Point. Bits 0 and 6 of
// the first byte together give the byte order, both clear being
// little-endian; bits 4 and 5 give the mantissa's normalization.
//
// Normalization 2 means the leading bit is implied. The sign's place is
// in the bit field's second byte; a little-endian mantissa starts at bit
// 0; the sign is the last bit, precision less one; and IEEE 754 biases
// an exponent by 2^(e-1) - 1.
const (
	floatOrderBits   = 0x41 // defined by IV.A.2.d
	floatNormShift   = 4    // defined by IV.A.2.d
	floatNormMask    = 0x03 // defined by IV.A.2.d
	floatNormImplied = 2    // defined by IV.A.2.d
	signField        = 1    // defined by IV.A.2.d
	mantissaFirst    = 0    // defined by IV.A.2.d
	lastBit          = 1    // defined by IV.A.2.d
	biasOne          = 1    // defined by IEEE 754
)

// ieeeLayout is where an IEEE 754 interchange format keeps its fields, in
// the terms of the Floating-Point properties of IV.A.2.d.
type ieeeLayout struct {
	size, exponentSize, mantissaSize int
}

// The widths of IEEE 754-2008's binary32 and binary64, its table 3.5,
// each defined by it.
const (
	binary32Exponent = 8  // defined by IEEE 754-2008
	binary32Mantissa = 23 // defined by IEEE 754-2008
	binary64Exponent = 11 // defined by IEEE 754-2008
	binary64Mantissa = 52 // defined by IEEE 754-2008
)

// ieeeFormats are binary32 and binary64, the two formats read.
var ieeeFormats = []ieeeLayout{
	{size: fourWide, exponentSize: binary32Exponent,
		mantissaSize: binary32Mantissa},
	{size: eightWide, exponentSize: binary64Exponent,
		mantissaSize: binary64Mantissa},
}

// floatingPoint is the datatype of a floating-point datatype, which must
// be a little-endian IEEE 754 binary32 or binary64 with an implied
// leading bit. IV.A.2.d Floating-Point. It refuses any other, and then
// gives none.
func floatingPoint(bits []byte, size int, props []byte,
	at uint64) (*datatype, error) {
	walk := &cursor{data: props}
	offset, precision := int(walk.u16()), int(walk.u16())
	exponentAt, exponentSize := int(walk.u8()), int(walk.u8())
	mantissaAt, mantissaSize := int(walk.u8()), int(walk.u8())
	bias := int(walk.u32())
	signAt := int(bits[signField])
	normalization := (bits[firstField] >> floatNormShift) & floatNormMask
	order := bits[firstField] & floatOrderBits
	got := ieeeLayout{size: size, exponentSize: exponentSize,
		mantissaSize: mantissaSize}
	// A little-endian binary format puts the sign last, the exponent
	// under it and the mantissa from bit 0, with a bias of 2^(e-1) - 1.
	placed := !walk.short && offset == noBitOffset &&
		precision == size*bitsPerByte && order == noBitOffset &&
		normalization == floatNormImplied &&
		mantissaAt == mantissaFirst && exponentAt == mantissaSize &&
		signAt == exponentAt+exponentSize && signAt == precision-lastBit
	biased := exponentSize > noBitOffset &&
		bias == biasOne<<(exponentSize-biasOne)-biasOne
	for _, format := range ieeeFormats {
		if placed && biased && format == got {
			return &datatype{class: floatClass, size: size}, nil
		}
	}
	return nil, refuse(errUnsupported, at, "IV.A.2.d",
		"a floating-point type of %d bytes (sign at %d, exponent "+
			"%d bits at %d, mantissa %d bits at %d, bias %d, "+
			"normalization %d, order bits %#x) that is not a "+
			"little-endian IEEE 754 binary32 or binary64", size,
		signAt, exponentSize, exponentAt, mantissaSize, mantissaAt,
		bias, normalization, order)
}

// The bit fields of the string datatype classes, each defined by
// IV.A.2.d. The low four bits of a fixed string's first byte are its
// padding and the high four its character set.
//
// A variable-length string has its padding in the high four bits of the
// first byte and its character set in the low four of the second. Of
// the character sets, ascii is 0 and utf-8 1; of the variable-length
// kinds, a sequence is 0 and a string 1.
//
// A string's base type is a one-byte character.
const (
	nibbleMask         = 0x0f // defined by IV.A.2.d
	nibbleShift        = 4    // defined by IV.A.2.d
	charsetUTF8        = 1    // defined by IV.A.2.d
	variableLengthText = 1    // defined by IV.A.2.d
	charsetField       = 1    // defined by IV.A.2.d
	oneByteCharacter   = 1    // defined by IV.A.2.d
)

// variableElementSize is the size of a variable-length element as
// stored: a four byte length, the address of a global heap collection
// and a four byte index into it. Defined by IV.A.2.d Variable-Length
// and III.E.
const variableElementSize = fourWide + addressSize + fourWide

// fixedString is the datatype of a fixed-length string. IV.A.2.d String.
// It refuses a padding or character set not read and a size of nothing,
// and then gives none.
func fixedString(bits []byte, size int, at uint64) (*datatype, error) {
	padding := stringPadding(bits[firstField] & nibbleMask)
	charset := int(bits[firstField] >> nibbleShift)
	if padding >= paddingKinds || size == noBytes ||
		charset > charsetUTF8 {
		return nil, refuse(errUnsupported, at, "IV.A.2.d",
			"a string of %d bytes with padding %d and character "+
				"set %d", size, padding, charset)
	}
	return &datatype{class: stringClass, size: size, padding: padding}, nil
}

// variableLength is the datatype of a variable-length string; a sequence
// of anything else is refused. The base type that follows the class bit
// field must be a one-byte character. IV.A.2.d Variable-Length.
func variableLength(bits []byte, size int, props []byte,
	at uint64) (*datatype, error) {
	kind := int(bits[firstField] & nibbleMask)
	padding := stringPadding(bits[firstField] >> nibbleShift)
	charset := int(bits[charsetField] & nibbleMask)
	if kind != variableLengthText || padding >= paddingKinds ||
		charset > charsetUTF8 || size != variableElementSize {
		return nil, refuse(errUnsupported, at, "IV.A.2.d",
			"a variable-length type that is not a string "+
				"(kind %d, element size %d, padding %d, "+
				"character set %d)", kind, size, padding,
			charset)
	}
	base, err := readDatatype(props, at+datatypeHeaderSize)
	if err != nil {
		return nil, err
	}
	if base.class != integerClass || base.size != oneByteCharacter {
		return nil, refuse(errUnsupported, at+datatypeHeaderSize,
			"IV.A.2.d", "a string whose characters are %v, not "+
				"one byte", base)
	}
	return &datatype{class: stringClass, size: size, variable: true,
		padding: padding}, nil
}
