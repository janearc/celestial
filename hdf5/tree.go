package hdf5

// treeNode is what the header of a version 1 B-tree node says of it: its
// level, 0 for a leaf, and how many children it uses. III.A.1.
type treeNode struct {
	level int
	used  uint64
}

// readTreeNode reads the header of the version 1 B-tree node at an
// address, which must be of a type, a group's or a dataset's, and at a
// level, unless the level wanted is noLevel. III.A.1.
//
// It refuses a node with no signature, of another type or at another
// level, and then gives none.
func (from *source) readTreeNode(at uint64, kind uint8,
	want int) (*treeNode, error) {
	raw, err := from.read(at, treeHeaderSize, "III.A.1")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	if !walk.opens(treeSignature) {
		return nil, refuse(errCorrupt, at, "III.A.1",
			"no TREE signature")
	}
	got, level, used := walk.u8(), int(walk.u8()), uint64(walk.u16())
	if got != kind {
		return nil, refuse(errCorrupt, at, "III.A.1",
			"a B-tree node of type %d where one of type %d belongs",
			got, kind)
	}
	if want != noLevel && level != want {
		return nil, refuse(errCorrupt, at, "III.A.1",
			"node is at level %d, its parent says %d", level, want)
	}
	return &treeNode{level: level, used: used}, nil
}
