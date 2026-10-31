package hdf5

// treeTypeChunk is the node type of a B-tree that indexes the chunks of
// a dataset. Defined by III.A.1.
const treeTypeChunk = 1

// chunkKeyFixed is the part of a chunk B-tree key before the chunk's
// offsets: the chunk's stored size (4 bytes) and the filter mask (4
// bytes). Defined by III.A.1, node type 1.
const chunkKeyFixed = fourWide + fourWide

// The levels and offsets of a chunk B-tree, each defined by III.A.1.
const (
	leafLevel     = 0 // defined: the level whose children are chunks
	levelBelow    = 1 // defined: a child is one level down
	elementOffset = 1 // defined: a key holds rank and one offsets
	alongElement  = 0 // defined: the last offset, along the element, is 0
)

// chunkEntry is one chunk of a dataset as the B-tree describes it: where
// its stored bytes are, how many, which filters were skipped, and where
// the chunk sits in the dataset.
type chunkEntry struct {
	address uint64
	stored  uint32
	mask    uint32
	// origin is the dataset index of the chunk's first element, along
	// each dimension of the dataset.
	origin []uint64
	// at is the address of the key, for refusals.
	at uint64
}

// chunkEntries reads the chunk B-tree node at an address, and below it,
// and gives the entry of every chunk in the order of the tree. rank is
// the dataset's number of dimensions; want is the level the node must be
// at, or noLevel. III.A.1.
func (from *source) chunkEntries(at uint64, want,
	rank int) ([]chunkEntry, error) {
	node, err := from.readTreeNode(at, treeTypeChunk, want)
	if err != nil {
		return nil, err
	}
	level, used := node.level, node.used
	// Each key holds the chunk's size, the filter mask and rank+1 offsets
	// of 8 bytes, the last offset being along the element, always 0.
	keySize := uint64(chunkKeyFixed + (rank+elementOffset)*addressSize)
	step := keySize + addressSize
	body, err := from.read(at+treeHeaderSize, used*step+keySize, "III.A.1")
	if err != nil {
		return nil, err
	}
	found := make([]chunkEntry, noBytes, used)
	for key := range used {
		start := key * step
		child := (&cursor{data: body, at: int(start + keySize)}).u64()
		if level > leafLevel {
			below, err := from.chunkEntries(child, level-levelBelow,
				rank)
			if err != nil {
				return nil, err
			}
			found = append(found, below...)
			continue
		}
		entry, err := chunkKey(body[start:start+keySize], rank,
			at+treeHeaderSize+start)
		if err != nil {
			return nil, err
		}
		entry.address = child
		found = append(found, *entry)
	}
	return found, nil
}

// chunkKey reads a key of a chunk B-tree, III.A.1 node type 1: the
// chunk's stored size, the filter mask, and the offsets of the chunk in
// the dataset, one more than the dataset's rank. It refuses a last
// offset that is not nothing, and then gives none.
func chunkKey(key []byte, rank int, at uint64) (*chunkEntry, error) {
	walk := &cursor{data: key}
	entry := chunkEntry{stored: walk.u32(), mask: walk.u32(), at: at}
	for range rank {
		entry.origin = append(entry.origin, walk.u64())
	}
	if last := walk.u64(); last != alongElement {
		return nil, refuse(errCorrupt, at, "III.A.1",
			"the offset along the element is %d, it is always 0",
			last)
	}
	return &entry, nil
}
