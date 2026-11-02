package hdf5

import

// treeNode records a version 1 B-tree node’s level and used child count. Level
// zero is a leaf (III.A.1).
"fmt"

type treeNode struct {
	level int
	used  uint64
}

// readTreeNode parses a version 1 B-tree header at address (III.A.1). It checks
// signature, expected group or chunk type, and expected level unless want is
// noLevel; mismatches return an error.
func (from *source) readTreeNode(at uint64, kind uint8,
	want int) (*treeNode, error) {
	raw, err := from.read(at, treeHeaderSize, "III.A.1")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	if !walk.opens(treeSignature) {
		return nil, fmt.Errorf("hdf5: file is damaged: no TREE "+
			"signature, at address %#x (spec III.A.1)", at)
	}
	got, level, used := walk.u8(), int(walk.u8()), uint64(walk.u16())
	if got != kind {
		return nil, fmt.Errorf("hdf5: file is damaged: a B-tree node "+
			"of type %d where one of type %d belongs, at address "+
			"%#x (spec III.A.1)", got, kind, at)
	}
	if want != noLevel && level != want {
		return nil, fmt.Errorf("hdf5: file is damaged: node is at "+
			"level %d, its parent says %d, at address %#x (spec "+
			"III.A.1)", level, want, at)
	}
	return &treeNode{level: level, used: used}, nil
}
