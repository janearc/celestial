package hdf5

import

// The header message type numbers of IV.A.2, Disk Format: Level 2 - Data
// Object Header Messages, each defined by it. Only the ones this reader
// reads or knowingly passes over are named.
"fmt"

const (
	// messageNil is IV.A.2.a, NIL. Defined by IV.A.2.
	messageNil = 0x0000
	// messageDataspace is IV.A.2.b. Defined by IV.A.2.
	messageDataspace = 0x0001
	// messageDatatype is IV.A.2.d. Defined by IV.A.2.
	messageDatatype = 0x0003
	// messageFillValue is IV.A.2.f, Data Storage - Fill Value. Defined by
	// IV.A.2.
	messageFillValue = 0x0005
	// messageLayout is IV.A.2.i, Data Storage - Layout. Defined by
	// IV.A.2.
	messageLayout = 0x0008
	// messageFilters is IV.A.2.l, Data Storage - Filter Pipeline. Defined
	// by IV.A.2.
	messageFilters = 0x000b
	// messageAttribute is IV.A.2.m. Defined by IV.A.2.
	messageAttribute = 0x000c
	// messageContinuation is IV.A.2.p, Object Header Continuation.
	// Defined by IV.A.2.
	messageContinuation = 0x0010
	// messageSymbolTable is IV.A.2.q. Defined by IV.A.2.
	messageSymbolTable = 0x0011
	// messageModified is IV.A.2.r, Object Modification Time. Defined by
	// IV.A.2.
	messageModified = 0x0012
)

// The prefixes of a version 1 object header and of its messages, each
// defined by IV.A.1.a. The header's prefix is the version, a reserved
// byte, the message count (2), the reference count (4), the header size
// (4) and 4 bytes of alignment.
//
// A message's prefix is its type (2), data size (2), flags (1) and 3
// reserved bytes. Bit 1 of its flags says the message is shared, its
// bytes a reference and not the message.
const (
	versionOneHeader  = 1    // defined by IV.A.1.a
	headerPrefixSize  = 16   // defined by IV.A.1.a
	messagePrefixSize = 8    // defined by IV.A.1.a
	referenceCount    = 4    // defined by IV.A.1.a
	messageReserved   = 3    // defined by IV.A.1.a
	messageSharedFlag = 0x02 // defined by IV.A.2, flags
	nothingSeen       = 0    // defined: no message counted yet
	nextBlock         = 1    // defined: the block after the first
)

// message is one header message: its type and its data, with the
// address of its data for refusals.
type message struct {
	kind uint16
	data []byte
	at   uint64
}

// object is an object header read whole: its messages, in file order.
type object struct {
	at       uint64
	messages []message
}

// readObject reads the object header at an address, and every
// continuation block it names, and gives the header's messages with
// continuation and NIL messages removed. IV.A.1.a.
func readObject(from *source, at uint64) (*object, error) {
	prefix, err := from.read(at, headerPrefixSize, "IV.A.1.a")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: prefix}
	version := walk.u8()
	if version != versionOneHeader {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"object header version %d, only version 1 is read, "+
			"at address %#x (spec IV.A.1.a)", version, at)
	}
	walk.skip(byteWide) // reserved
	count := int(walk.u16())
	walk.skip(referenceCount)
	size := uint64(walk.u32())
	found := &object{at: at}
	blocks := []block{{at: at + headerPrefixSize, size: size}}
	seen := nothingSeen
	for len(blocks) > noBytes {
		first := blocks[firstPlace]
		more, err := found.readBlock(from, first, &seen, count)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks[nextBlock:], more...)
	}
	return found, nil
}

// block is a run of header messages in the file: the first block follows
// the prefix, and each continuation message names another.
type block struct {
	at, size uint64
}

// readBlock reads the messages of one block into the object, counting
// them in seen. A header holds at most count messages in all, which also
// stops a loop of continuations. It gives the blocks the continuations
// name.
func (found *object) readBlock(from *source, in block, seen *int,
	count int) ([]block, error) {
	raw, err := from.read(in.at, in.size, "IV.A.1.a")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	var more []block
	for len(raw)-walk.at >= messagePrefixSize {
		*seen++
		if *seen > count {
			reached := in.at + uint64(walk.at)
			return nil, fmt.Errorf("hdf5: file is damaged: more "+
				"messages than the %d the header declares, "+
				"at address %#x (spec "+
				"IV.A.1.a)", count, reached)
		}
		kind, size := walk.u16(), uint64(walk.u16())
		flags := walk.u8()
		walk.skip(messageReserved)
		start := in.at + uint64(walk.at)
		data := walk.take(int(size))
		if walk.short {
			return nil, fmt.Errorf("hdf5: file is damaged: "+
				"message of type %#x holds %d bytes, past "+
				"its block, at address %#x (spec "+
				"IV.A.1.a)", kind, size, start)
		}
		if flags&messageSharedFlag != noFlag {
			return nil, fmt.Errorf("hdf5: part of the format not "+
				"read: shared message of type %#x, at "+
				"address %#x (spec "+
				"IV.A.2)", kind, start)
		}
		switch kind {
		case messageContinuation:
			next, err := continuation(data, start)
			if err != nil {
				return nil, err
			}
			more = append(more, *next)
		case messageNil, messageFillValue, messageModified:
			// Passed over: none changes what a reader reads.
		default:
			found.messages = append(found.messages,
				message{kind: kind, data: data, at: start})
		}
	}
	return more, nil
}

// continuation is the block a header continuation message names: the
// address of the block and its length, each eight bytes. IV.A.2.p.
// A message too short to hold them returns an error.
func continuation(data []byte, at uint64) (*block, error) {
	walk := &cursor{data: data}
	next := block{at: walk.u64(), size: walk.u64()}
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: continuation "+
			"message is %d bytes, 16 are needed, at address %#x "+
			"(spec IV.A.2.p)", len(data), at)
	}
	return &next, nil
}
