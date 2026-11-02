package hdf5

import

// The Data Layout message read, each defined by IV.A.2.i: version 3, and
// its classes of storage.
"fmt"

const (
	versionThreeLayout = 3 // defined by IV.A.2.i
	layoutCompact      = 0 // defined by IV.A.2.i
	layoutContiguous   = 1 // defined by IV.A.2.i
	layoutChunked      = 2 // defined by IV.A.2.i
	elementDimension   = 1 // defined by IV.A.2.i: rank and one dimensions
	noChunk            = 0 // defined: a chunk dimension of nothing is none
)

// layout says where a dataset's bytes are and how they are stored.
type layout struct {
	// chunked says whether the data is cut into chunks found through a
	// B-tree. If it is not, it is one run of bytes at address.
	chunked bool
	// address is the contiguous data, or the root of the chunk B-tree.
	address uint64
	// size is the bytes of contiguous data.
	size uint64
	// chunk is the size in elements of a chunk, along each dimension.
	chunk []uint64
	// at is the address of the layout message, for refusals.
	at uint64
}

// readLayout parses a version 3 Data Layout message for the dataset’s rank
// and element size (IV.A.2.i). Truncated messages and unsupported versions
// or storage classes return an error.
func readLayout(msg message, rank int, elementSize int) (*layout, error) {
	walk := &cursor{data: msg.data}
	version, class := walk.u8(), walk.u8()
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: layout "+
			"message of %d bytes is shorter than its fixed part, "+
			"at address %#x (spec "+
			"IV.A.2.i)", len(msg.data), msg.at)
	}
	if version != versionThreeLayout {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"data layout version %d, only version 3 is read, at "+
			"address %#x (spec IV.A.2.i)", version, msg.at)
	}
	switch class {
	case layoutContiguous:
		found := &layout{address: walk.u64(), size: walk.u64(),
			at: msg.at}
		if walk.short {
			return nil, fmt.Errorf("hdf5: file is damaged: "+
				"contiguous layout is cut short, at address "+
				"%#x (spec IV.A.2.i)", msg.at)
		}
		return found, nil
	case layoutChunked:
		return readChunked(walk, msg.at, rank, elementSize)
	case layoutCompact:
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"compact storage, only contiguous and chunked are "+
			"read, at address %#x (spec IV.A.2.i)", msg.at)
	}
	return nil, fmt.Errorf("hdf5: file is damaged: layout class %d is "+
		"not defined for version 3, at address %#x (spec "+
		"IV.A.2.i)", class, msg.at)
}

// readChunked reads the rest of a chunked layout: the dimensionality,
// the address of the chunk B-tree, then the chunk sizes, one more than
// the dataset's rank, the last being the element size in bytes.
// IV.A.2.i.
func readChunked(walk *cursor, at uint64, rank,
	elementSize int) (*layout, error) {
	dimensionality := int(walk.u8())
	found := &layout{chunked: true, address: walk.u64(), at: at}
	if dimensionality != rank+elementDimension {
		return nil, fmt.Errorf("hdf5: file is damaged: chunk layout "+
			"of %d dimensions for a dataset of %d, at address "+
			"%#x (spec "+
			"IV.A.2.i)", dimensionality-elementDimension, rank, at)
	}
	for range rank {
		size := uint64(walk.u32())
		found.chunk = append(found.chunk, size)
	}
	stored := int(walk.u32())
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: chunked "+
			"layout is cut short, at address %#x (spec "+
			"IV.A.2.i)", at)
	}
	if stored != elementSize {
		return nil, fmt.Errorf("hdf5: file is damaged: chunk element "+
			"size %d, the datatype's is %d, at address %#x (spec "+
			"IV.A.2.i)", stored, elementSize, at)
	}
	for _, size := range found.chunk {
		if size == noChunk {
			return nil, fmt.Errorf("hdf5: file is damaged: a "+
				"chunk dimension of zero, at address %#x "+
				"(spec IV.A.2.i)", at)
		}
	}
	return found, nil
}
