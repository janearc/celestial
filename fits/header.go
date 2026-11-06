// Package fits reads the images and binary tables in FITS files as the
// FITS Standard 4.0 describes them, and refuses the rest. It reads a
// header's cards, an image's pixels a row at a time, a table's cells a
// few thousand rows at a time, and where an image lies on the sky.
//
// The image is the primary array or an `IMAGE` extension. Its pixels
// are any of the six types the standard allows. They are scaled by
// `BSCALE` and `BZERO`, and `BLANK` is read as no value.
//
// The table is the first `BINTABLE` extension. A column is read when it
// holds one value a row: a logical, an integer of one to eight bytes, a
// single or double float, or text. `TSCAL` and `TZERO` scale the numbers.
//
// A column of many values a row is stepped over: the table opens and its
// other columns read, and a read of it is refused.
//
// An integer equal to its column's `TNULL`, and a float that is not a
// number, are read as no value. A `TNULL` on a float column is ignored:
// archives write one on every column, and a float's null is not a
// number whatever the card says.
//
// Where it lies on the sky is read from the world coordinate cards. The
// projections read are TAN and CAR, in equatorial or galactic
// coordinates.
//
// Anything else is refused, and each refusal says why in words.
package fits

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// The FITS Standard's units of reading.
const (
	// block is the size of a header or data block. It's defined by the
	// FITS Standard 4.0, section 3.1.
	block = 2880
	// cardWidth is the width of a header card. It's defined by section
	// 4.1.
	cardWidth = 80
	// keyWidth is the width of a card's keyword. It's defined by section
	// 4.1.2.1.
	keyWidth = 8
	// valueAt is where a card's value begins, after "= ". It's defined by
	// section 4.1.2.2.
	valueAt = 10
	// noBytes is a read of nothing. It's defined.
	noBytes = 0
)

// errRefused is what Refused gives.
var errRefused = errors.New("not read")

// Refused is wrapped by each error about a file this package does not
// read: a part of the standard it does not read, or a file that breaks the
// standard.
func Refused() error { return errRefused }

// Header is a header's cards by keyword. Each value is kept as its card
// writes it, with its comment taken off.
type Header struct {
	values map[string]string
}

// ReadHeader is the header at the front of a reader, read a block at a
// time to its END card. Headers that end before their END card are
// refused.
func ReadHeader(from io.Reader) (*Header, error) {
	read := &Header{values: map[string]string{}}
	page := make([]byte, block)
	for {
		if _, err := io.ReadFull(from, page); err != nil {
			return nil, fmt.Errorf("%w: a header with no END "+
				"card: %w", errRefused, err)
		}
		if read.keep(page) {
			return read, nil
		}
	}
}

// keep keeps the values on one block's cards, and says whether the block
// held the END card.
func (read *Header) keep(page []byte) bool {
	for at := noBytes; at < block; at += cardWidth {
		card := string(page[at : at+cardWidth])
		key := strings.TrimSpace(card[:keyWidth])
		if key == "END" {
			return true
		}
		if card[keyWidth:valueAt] == "= " {
			read.values[key] = value(card[valueAt:])
		}
	}
	return false
}

// value is a card's value with its comment taken off. Strings keep their
// quotes, so Text can tell it from a number.
func value(field string) string {
	trimmed := strings.TrimSpace(field)
	if !strings.HasPrefix(trimmed, "'") {
		kept, _, _ := strings.Cut(trimmed, "/")
		return strings.TrimSpace(kept)
	}
	var text bytes.Buffer
	text.WriteByte('\'')
	rest := trimmed[len("'"):]
	for {
		before, after, closed := strings.Cut(rest, "'")
		text.WriteString(before)
		if !closed || !strings.HasPrefix(after, "'") {
			break
		}
		text.WriteString("''")
		rest = after[len("'"):]
	}
	text.WriteByte('\'')
	return text.String()
}

// Has is whether the header has a card for a keyword.
func (read *Header) Has(key string) bool {
	_, held := read.values[key]
	return held
}

// Text is a keyword's string value with its quotes taken off. Doubled
// quotes are read as one. The spaces after the value are dropped.
func (read *Header) Text(key string) (string, error) {
	kept := read.values[key]
	if !strings.HasPrefix(kept, "'") {
		return "", fmt.Errorf("%w: %s is not a string", errRefused, key)
	}
	inner := kept[len("'") : len(kept)-len("'")]
	inner = strings.ReplaceAll(inner, "''", "'")
	return strings.TrimRight(inner, " "), nil
}

// Number is a keyword's value as a number. Some writers put a D in the
// exponent, and it is read as E.
func (read *Header) Number(key string) (float64, error) {
	kept, held := read.values[key]
	if !held {
		return math.NaN(), fmt.Errorf("%w: no %s card", errRefused, key)
	}
	written := strings.ReplaceAll(strings.ToUpper(kept), "D", "E")
	number, err := strconv.ParseFloat(written, bits)
	if err != nil {
		return math.NaN(), fmt.Errorf("%w: %s is %q, not a number",
			errRefused, key, kept)
	}
	return number, nil
}

// Integer is a keyword's value as a whole number, read whole: one too
// large for a float to hold is read exactly. It refuses a value that is
// not a whole number.
func (read *Header) Integer(key string) (int64, error) {
	kept, held := read.values[key]
	if !held {
		return noInteger, fmt.Errorf("%w: no %s card", errRefused, key)
	}
	number, err := strconv.ParseInt(kept, decimal, bits)
	if err != nil {
		return noInteger, fmt.Errorf("%w: %s is %q, not a whole number",
			errRefused, key, kept)
	}
	return number, nil
}

// How numbers are read, each defined.
const (
	bits      = 64 // defined: a float64, and an int64, are 64 bits
	decimal   = 10 // defined: the base cards write whole numbers in
	noInteger = 0  // defined: the whole number given beside a refusal
)
