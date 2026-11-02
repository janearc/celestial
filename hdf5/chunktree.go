package hdf5

import

// treeTypeChunk is the node type of a B-tree that indexes the chunks of
// a dataset. Defined by III.A.1.
"fmt"

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

// chunkEntry records stored address, byte count, skipped-filter mask, and
// dataset origin for one indexed chunk.
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

// chunkEntries traverses a chunk B-tree at address and returns entries in tree
// order (III.A.1). rank is dataset dimensionality; want requires a node level
// unless set to noLevel.
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

// chunkKey parses stored size, filter mask, and rank+1 offsets from a
// chunk-index key (III.A.1, node type 1). The final offset addresses the
// element byte dimension and must be zero.
func chunkKey(key []byte, rank int, at uint64) (*chunkEntry, error) {
	walk := &cursor{data: key}
	entry := chunkEntry{stored: walk.u32(), mask: walk.u32(), at: at}
	for range rank {
		entry.origin = append(entry.origin, walk.u64())
	}
	if last := walk.u64(); last != alongElement {
		return nil, fmt.Errorf("hdf5: file is damaged: the offset "+
			"along the element is %d, it is always 0, at address "+
			"%#x (spec III.A.1)", last, at)
	}
	return &entry, nil
}
