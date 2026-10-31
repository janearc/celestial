package hdf5

// The Data Layout message read, each defined by IV.A.2.i: version 3, and
// its classes of storage.
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

// readLayout parses a Data Layout message for a dataset of a rank and an
// element size. IV.A.2.i, version 3. It refuses a message too short, a
// version or class not read, and then gives none.
func readLayout(msg message, rank int, elementSize int) (*layout, error) {
	walk := &cursor{data: msg.data}
	version, class := walk.u8(), walk.u8()
	if walk.short {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.i",
			"layout message of %d bytes is shorter than its fixed "+
				"part", len(msg.data))
	}
	if version != versionThreeLayout {
		return nil, refuse(errUnsupported, msg.at, "IV.A.2.i",
			"data layout version %d, only version 3 is read",
			version)
	}
	switch class {
	case layoutContiguous:
		found := &layout{address: walk.u64(), size: walk.u64(),
			at: msg.at}
		if walk.short {
			return nil, refuse(errCorrupt, msg.at, "IV.A.2.i",
				"contiguous layout is cut short")
		}
		return found, nil
	case layoutChunked:
		return readChunked(walk, msg.at, rank, elementSize)
	case layoutCompact:
		return nil, refuse(errUnsupported, msg.at, "IV.A.2.i",
			"compact storage, only contiguous and chunked are read")
	}
	return nil, refuse(errCorrupt, msg.at, "IV.A.2.i",
		"layout class %d is not defined for version 3", class)
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
		return nil, refuse(errCorrupt, at, "IV.A.2.i",
			"chunk layout of %d dimensions for a dataset of %d",
			dimensionality-elementDimension, rank)
	}
	for range rank {
		size := uint64(walk.u32())
		found.chunk = append(found.chunk, size)
	}
	stored := int(walk.u32())
	if walk.short {
		return nil, refuse(errCorrupt, at, "IV.A.2.i",
			"chunked layout is cut short")
	}
	if stored != elementSize {
		return nil, refuse(errCorrupt, at, "IV.A.2.i",
			"chunk element size %d, the datatype's is %d", stored,
			elementSize)
	}
	for _, size := range found.chunk {
		if size == noChunk {
			return nil, refuse(errCorrupt, at, "IV.A.2.i",
				"a chunk dimension of zero")
		}
	}
	return found, nil
}
