package hdf5

import (
	"encoding/binary"
	"fmt"
)

// checksumSize is the bytes the Fletcher32 filter (filter id 3, IV.A.2.l)
// appends to a chunk. Defined by reading them: four bytes.
//
// The specification names the filter and says no more. The size, the
// word order and the way the sums are kept were read from the chunks of
// a real file of the open science centre's, in which every chunk's
// checksum agrees.
const checksumSize = fourWide

// fletcherModulus is the modulus of a Fletcher-32 sum: 2^16 - 1, the
// largest value of a 16-bit word. Defined by Fletcher's checksum.
const fletcherModulus = 1<<halfShift - 1

// The halves of a Fletcher-32 sum: a 32-bit sum of two 16-bit halves,
// the second running sum in the high half.
const (
	halfShift = 16 // defined: a half's bits
	evenWords = 2  // defined: a word's bytes
	whole     = 0  // defined: what an even count leaves over
)

// verifyFletcher32 checks the checksum that ends a chunk, and gives the
// chunk without it. The checksum is a Fletcher-32 over the chunk's bytes
// taken as big-endian 16-bit words, stored little-endian. A chunk of an
// odd number of bytes has no defined sum and is refused.
func verifyFletcher32(data []byte, entry chunkEntry) ([]byte, error) {
	if len(data) < checksumSize {
		return nil, fmt.Errorf("hdf5: file is damaged: the chunk at "+
			"%v is %d bytes, too short for a checksum, at "+
			"address %#x (spec "+
			"IV.A.2.l)", entry.origin, len(data), entry.at)
	}
	cut := len(data) - checksumSize
	body, stored := data[:cut], data[cut:]
	if len(body)%evenWords != whole {
		return nil, fmt.Errorf("hdf5: part of the format not read: a "+
			"Fletcher32 sum over %d bytes, an odd number: its "+
			"words are not defined, at address %#x (spec "+
			"IV.A.2.l)", len(body), entry.at)
	}
	got, want := fletcher32(body), binary.LittleEndian.Uint32(stored)
	if got != want {
		return nil, fmt.Errorf("hdf5: file is damaged: the chunk at "+
			"%v sums to %#08x, its checksum says %#08x, at "+
			"address %#x (spec "+
			"IV.A.2.l)", entry.origin, got, want, entry.at)
	}
	return body, nil
}

// fletcher32 is the Fletcher-32 sum of data, which has an even length,
// read as big-endian 16-bit words: the second running sum in the high
// half and the first in the low half.
func fletcher32(data []byte) uint32 {
	var first, second uint32
	for at := whole; at < len(data); at += evenWords {
		word := uint32(binary.BigEndian.Uint16(data[at:]))
		first = (first + word) % fletcherModulus
		second = (second + first) % fletcherModulus
	}
	return second<<halfShift | first
}
