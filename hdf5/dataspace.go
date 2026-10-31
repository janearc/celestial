package hdf5

import (
	"math"
	"slices"
)

// The Dataspace message read, each defined by IV.A.2.b: version 1, which
// version 2 changes by adding a type byte and dropping the permutation
// indices; at most 32 dimensions; and 5 bytes after the flags, one
// reserved and then four.
//
// Bit 1 of the flags says permutation indices follow the sizes. Bit 0
// says the maximum sizes follow them, which are not read.
const (
	versionOneDataspace = 1    // defined by IV.A.2.b
	maxRank             = 32   // defined by IV.A.2.b
	permutationsFlag    = 0x02 // defined by IV.A.2.b
	dataspaceReserved   = 5    // defined by IV.A.2.b
	noFlag              = 0    // defined: a flag that is clear
)

// readDataspace parses a Dataspace message at an address and gives the
// sizes of its dimensions; a scalar has none. It refuses a message too
// short, a version not read, permutation indices, and too many
// dimensions, and then gives none.
func readDataspace(data []byte, at uint64) ([]uint64, error) {
	walk := &cursor{data: data}
	version, rank, flags := walk.u8(), int(walk.u8()), walk.u8()
	walk.skip(dataspaceReserved)
	if walk.short {
		return nil, refuse(errCorrupt, at, "IV.A.2.b",
			"dataspace message of %d bytes is shorter than its "+
				"fixed part", len(data))
	}
	if version != versionOneDataspace {
		return nil, refuse(errUnsupported, at, "IV.A.2.b",
			"dataspace version %d, only version 1 is read", version)
	}
	if flags&permutationsFlag != noFlag {
		return nil, refuse(errUnsupported, at, "IV.A.2.b",
			"a dataspace with permutation indices")
	}
	if rank > maxRank {
		return nil, refuse(errCorrupt, at, "IV.A.2.b",
			"%d dimensions, at most %d are allowed", rank, maxRank)
	}
	shape := make([]uint64, rank)
	for axis := range shape {
		shape[axis] = walk.u64()
	}
	if walk.short {
		return nil, refuse(errCorrupt, at, "IV.A.2.b",
			"%d dimensions do not fit in %d bytes", rank, len(data))
	}
	return shape, nil
}

// elementCount is the product of the sizes of the dimensions: how many
// elements a dataspace holds. A scalar holds one, and a shape with a
// dimension of nothing holds none, however large the rest. counted is
// false if the product does not fit in a uint64.
func elementCount(shape []uint64) (count uint64, counted bool) {
	if slices.Contains(shape, noBytes) {
		return noBytes, true
	}
	count = oneElement
	for _, size := range shape {
		if count > math.MaxUint64/size {
			return noBytes, false
		}
		count *= size
	}
	return count, true
}
