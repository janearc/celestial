package hdf5

import

// The sizes in a global heap collection, each defined by III.E: the
// header is the signature (4), the version (1), 3 reserved bytes and the
// collection size (8); an object starts with its index (2), reference
// count (2), 4 reserved bytes and its size (8), and is padded to 8 bytes.
"fmt"

const (
	collectionHeaderSize = 16 // defined by III.E
	heapObjectHeaderSize = 16 // defined by III.E
	heapObjectAlign      = 8  // defined by III.E
	collectionVersion    = 1  // defined by III.E: the only version
	collectionReserved   = 3  // defined by III.E
	objectReserved       = 6  // defined by III.E: count and reserved
	freeSpace            = 0  // defined by III.E: object 0 is free space
)

// collectionSignature opens a global heap collection, III.E.
var collectionSignature = []byte("GCOL")

// collection is a global heap collection read: its objects by index.
type collection map[uint32][]byte

// globalHeap reads collections as they are asked for, once each.
type globalHeap struct {
	from *source
	read map[uint64]collection
}

// newGlobalHeap is a global heap reader over a file.
func newGlobalHeap(from *source) *globalHeap {
	return &globalHeap{from: from, read: map[uint64]collection{}}
}

// object is the bytes of an object, by its index, in the collection at
// an address.
func (store *globalHeap) object(at uint64, index uint32) ([]byte, error) {
	made, held := store.read[at]
	if !held {
		var err error
		made, err = store.collection(at)
		if err != nil {
			return nil, err
		}
		store.read[at] = made
	}
	data, held := made[index]
	if !held {
		return nil, fmt.Errorf("hdf5: file is damaged: no object %d "+
			"in the global heap collection, at address %#x (spec "+
			"III.E)", index, at)
	}
	return data, nil
}

// collection reads the whole collection at an address. III.E.
func (store *globalHeap) collection(at uint64) (collection, error) {
	raw, err := store.from.read(at, collectionHeaderSize, "III.E")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw}
	if !walk.opens(collectionSignature) {
		return nil, fmt.Errorf("hdf5: file is damaged: no GCOL "+
			"signature, at address %#x (spec III.E)", at)
	}
	if version := walk.u8(); version != collectionVersion {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"global heap version %d, only 1 exists, at address "+
			"%#x (spec III.E)", version, at)
	}
	walk.skip(collectionReserved)
	size := walk.u64()
	if size < collectionHeaderSize {
		return nil, fmt.Errorf("hdf5: file is damaged: collection "+
			"size %d is smaller than its header, at address %#x "+
			"(spec III.E)", size, at)
	}
	body, err := store.from.read(at+collectionHeaderSize,
		size-collectionHeaderSize, "III.E")
	if err != nil {
		return nil, err
	}
	return parseObjects(body, at+collectionHeaderSize)
}

// parseObjects reads the objects of a collection's body: index,
// reference count, reserved, size and data padded to a multiple of 8
// bytes. Object 0 is the free space and ends the list. III.E.
func parseObjects(body []byte, at uint64) (collection, error) {
	made := collection{}
	walk := &cursor{data: body}
	for len(body)-walk.at >= heapObjectHeaderSize {
		index := walk.u16()
		if index == freeSpace {
			break
		}
		walk.skip(objectReserved)
		size, room := walk.u64(), uint64(len(body))
		bounded := int(min(size, room))
		data := walk.take(bounded)
		if walk.short || size > room {
			reached := at + uint64(walk.at)
			return nil, fmt.Errorf("hdf5: file is damaged: "+
				"object %d of %d bytes runs past its "+
				"collection, at address %#x (spec "+
				"III.E)", index, size, reached)
		}
		walk.align(heapObjectAlign)
		made[uint32(index)] = data
	}
	return made, nil
}
