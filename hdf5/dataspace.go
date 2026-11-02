package hdf5

import (
	"fmt"
	"math"
	"slices"
)

// The Dataspace message read, each defined by IV.A.2.b: version 1, which
// version 2 changes by adding a type byte and dropping the permutation
// indices; and 5 bytes after the flags, one reserved and then four.
//
// IV.A.2.b sets no most for the dimensions. The hdf5 library writes at
// most 32, its H5S_MAX_RANK (src/H5Spublic.h, read 2026-11-01), and this
// reader reads no more: more is refused as not read, not as damaged.
//
// Bit 1 of the flags says permutation indices follow the sizes. Bit 0
// says the maximum sizes follow them, which are not read.
const (
	versionOneDataspace = 1    // defined by IV.A.2.b
	maxRank             = 32   // defined by the hdf5 library's H5S_MAX_RANK
	permutationsFlag    = 0x02 // defined by IV.A.2.b
	dataspaceReserved   = 5    // defined by IV.A.2.b
	noFlag              = 0    // defined: a flag that is clear
)

// readDataspace parses dimension sizes from a version 1 Dataspace message
// (IV.A.2.b). Scalars have no dimensions. Truncation, unsupported versions,
// permutation indices, and excessive rank return an error.
func readDataspace(data []byte, at uint64) ([]uint64, error) {
	walk := &cursor{data: data}
	version, rank, flags := walk.u8(), int(walk.u8()), walk.u8()
	walk.skip(dataspaceReserved)
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: dataspace "+
			"message of %d bytes is shorter than its fixed part, "+
			"at address %#x (spec IV.A.2.b)", len(data), at)
	}
	if version != versionOneDataspace {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"dataspace version %d, only version 1 is read, at "+
			"address %#x (spec IV.A.2.b)", version, at)
	}
	if flags&permutationsFlag != noFlag {
		return nil, fmt.Errorf("hdf5: part of the format not read: a "+
			"dataspace with permutation indices, at address %#x "+
			"(spec IV.A.2.b)", at)
	}
	if rank > maxRank {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"%d dimensions, more than the %d the hdf5 library "+
			"writes, at address %#x (spec "+
			"IV.A.2.b)", rank, maxRank, at)
	}
	shape := make([]uint64, rank)
	for axis := range shape {
		shape[axis] = walk.u64()
	}
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: %d dimensions "+
			"do not fit in %d bytes, at address %#x (spec "+
			"IV.A.2.b)", rank, len(data), at)
	}
	return shape, nil
}

// elementCount multiplies dimension sizes. A scalar contains one element; a
// zero dimension makes the total zero regardless of other sizes. counted is
// false if a nonzero product exceeds uint64.
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
