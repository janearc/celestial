package hdf5

import "bytes"

// The sizes of the group structures with eight-byte offsets and lengths,
// each defined by the section it names.
const (
	// heapHeaderSize is the local heap header of III.D: signature (4),
	// version (1), 3 reserved bytes, data segment size (8), free-list
	// offset (8) and data segment address (8). Defined by III.D.
	heapHeaderSize = 32
	// treeHeaderSize is the version 1 B-tree node header of III.A.1:
	// signature (4), node type (1), node level (1), entries used (2) and
	// the two sibling addresses (8 each). Defined by III.A.1.
	treeHeaderSize = 24
	// symbolEntrySize is a symbol table entry of III.C: name offset (8),
	// object header address (8), cache type (4), 4 reserved bytes and a
	// scratch-pad of 16 bytes. Defined by III.C.
	symbolEntrySize = 40
	// nodeHeaderSize is the symbol table node header of III.B: signature
	// (4), version (1), a reserved byte and the number of symbols (2).
	// Defined by III.B.
	nodeHeaderSize = 8
)

// The versions, types and counts of a group's structures, each defined
// by the section it names: a symbolic link's cache type; a group's node
// type; the only versions of a node and a heap; a node of at most 2K
// symbols; and keys and children that alternate, a key more than them.
const (
	cacheSymbolicLink = 2  // defined by III.C
	treeTypeGroup     = 0  // defined by III.A.1
	nodeVersion       = 1  // defined by III.B
	heapVersion       = 0  // defined by III.D
	heapReserved      = 3  // defined by III.D
	twiceK            = 2  // defined by III.B
	keysAround        = 2  // defined by III.A.1
	lastKey           = 1  // defined by III.A.1
	noLevel           = -1 // defined here: the level a tree's root takes
)

// link is one member of a group: its name and the address of its object
// header.
type link struct {
	name   string
	header uint64
}

// groupTable is what the symbol table message of an old-style group
// gives: the version 1 B-tree of its members and the local heap of their
// names. IV.A.2.q.
type groupTable struct {
	tree, heap uint64
}

// parseSymbolTable reads the symbol table message's two addresses. It
// refuses a message too short for them, and then gives none.
func parseSymbolTable(msg message) (*groupTable, error) {
	walk := &cursor{data: msg.data}
	table := groupTable{tree: walk.u64(), heap: walk.u64()}
	if walk.short {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.q",
			"symbol table message is %d bytes, 16 are needed",
			len(msg.data))
	}
	return &table, nil
}

// The signatures that open a local heap (III.D), a version 1 B-tree node
// (III.A.1) and a group symbol table node (III.B).
var (
	heapSignature = []byte("HEAP")
	treeSignature = []byte("TREE")
	nodeSignature = []byte("SNOD")
)

// members is the links of a group, in the order of the B-tree, which is
// the order of the names.
func (from *source) members(table groupTable, leafK uint16) ([]link,
	error) {
	names, err := from.readHeap(table.heap)
	if err != nil {
		return nil, err
	}
	symbols, err := from.groupSymbols(table.tree, noLevel, leafK)
	if err != nil {
		return nil, err
	}
	found := make([]link, noBytes, len(symbols))
	for _, entry := range symbols {
		name, err := names.name(entry.nameOffset, entry.at)
		if err != nil {
			return nil, err
		}
		found = append(found, link{name: name, header: entry.header})
	}
	return found, nil
}

// symbol is a symbol table entry as III.C gives it, and where it was
// read.
type symbol struct {
	nameOffset, header uint64
	at                 uint64
}

// groupSymbols reads the version 1 B-tree node at an address, and below
// it, and gives the symbol table entries of every leaf in order. want is
// the level the node must be at, or noLevel. III.A.1 and III.B.
func (from *source) groupSymbols(at uint64, want int,
	leafK uint16) ([]symbol, error) {
	node, err := from.readTreeNode(at, treeTypeGroup, want)
	if err != nil {
		return nil, err
	}
	// Keys and children alternate, a key first and last: used children
	// and used+1 keys, 8 bytes each.
	body, err := from.read(at+treeHeaderSize,
		(keysAround*node.used+lastKey)*addressSize, "III.A.1")
	if err != nil {
		return nil, err
	}
	found := make([]symbol, noBytes, node.used)
	for child := range node.used {
		spot := int((keysAround*child + lastKey) * addressSize)
		address := (&cursor{data: body, at: spot}).u64()
		var below []symbol
		if node.level == leafLevel {
			below, err = from.readNode(address, leafK)
		} else {
			below, err = from.groupSymbols(address,
				node.level-levelBelow, leafK)
		}
		if err != nil {
			return nil, err
		}
		found = append(found, below...)
	}
	return found, nil
}

// readNode reads the symbol table node at an address. III.B.
func (from *source) readNode(at uint64, leafK uint16) ([]symbol, error) {
	raw, err := from.read(at, nodeHeaderSize, "III.B")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	if !walk.opens(nodeSignature) {
		return nil, refuse(errCorrupt, at, "III.B", "no SNOD signature")
	}
	if version := walk.u8(); version != nodeVersion {
		return nil, refuse(errUnsupported, at, "III.B",
			"symbol table node version %d, only 1 exists", version)
	}
	walk.skip(byteWide) // reserved
	count := uint64(walk.u16())
	most := twiceK * uint64(leafK)
	if count > most {
		return nil, refuse(errCorrupt, at, "III.B",
			"%d symbols in a node that holds at most 2K = %d",
			count, most)
	}
	body, err := from.read(at+nodeHeaderSize, count*symbolEntrySize,
		"III.C")
	if err != nil {
		return nil, err
	}
	found := make([]symbol, noBytes, count)
	for place := range count {
		here := at + nodeHeaderSize + place*symbolEntrySize
		entry := &cursor{data: body, at: int(place * symbolEntrySize)}
		made := symbol{nameOffset: entry.u64(), header: entry.u64(),
			at: here}
		if cache := entry.u32(); cache == cacheSymbolicLink {
			return nil, refuse(errUnsupported, here, "III.C",
				"a symbolic link in a group")
		}
		found = append(found, made)
	}
	return found, nil
}

// heap is a local heap's data segment, where a group's names are kept.
type heap struct {
	data []byte
}

// readHeap reads the local heap at an address. III.D.
func (from *source) readHeap(at uint64) (*heap, error) {
	raw, err := from.read(at, heapHeaderSize, "III.D")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	if !walk.opens(heapSignature) {
		return nil, refuse(errCorrupt, at, "III.D", "no HEAP signature")
	}
	if version := walk.u8(); version != heapVersion {
		return nil, refuse(errUnsupported, at, "III.D",
			"local heap version %d, only 0 exists", version)
	}
	walk.skip(heapReserved)
	size := walk.u64()
	walk.skip(addressSize) // head of the free list
	segment := walk.u64()
	data, err := from.read(segment, size, "III.D")
	if err != nil {
		return nil, err
	}
	return &heap{data: data}, nil
}

// name is the null-terminated name at an offset in the heap's data
// segment; at is where the offset was read, for a refusal.
func (store *heap) name(offset, at uint64) (string, error) {
	if offset >= uint64(len(store.data)) {
		return "", refuse(errCorrupt, at, "III.D",
			"name offset %d is outside the heap's %d bytes", offset,
			len(store.data))
	}
	rest := store.data[offset:]
	end := bytes.IndexByte(rest, nameEnd)
	if end == notFound {
		return "", refuse(errCorrupt, at, "III.D",
			"the name at heap offset %d has no end", offset)
	}
	return string(rest[:end]), nil
}
