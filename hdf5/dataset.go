package hdf5

import (
	"fmt"
	"math"
)

// description describes a dataset in the file: the sizes of its
// dimensions and the type of its elements.
type description struct {
	// shape is the size of each dimension. A scalar has no dimensions,
	// and holds one element.
	shape []uint64
	// kind is the type of one element.
	kind datatype
}

// dataset is a dataset object read: its description and where its bytes
// are.
type dataset struct {
	description
	layout  *layout
	filters *filters
	// count is the number of elements.
	count uint64
}

// datasetOf reads the messages of an object header that make a dataset:
// a dataspace, a datatype, a layout and, if there is one, a filter
// pipeline. An object without the first three is a group, or something
// else, and is refused.
func datasetOf(found *object) (*dataset, error) {
	var space, kind, place, pipeline *message
	for spot := range found.messages {
		msg := &found.messages[spot]
		switch msg.kind {
		case messageDataspace:
			space = msg
		case messageDatatype:
			kind = msg
		case messageLayout:
			place = msg
		case messageFilters:
			pipeline = msg
		}
	}
	if space == nil || kind == nil || place == nil {
		return nil, fmt.Errorf("%w: the object at %#x is not a dataset",
			errWrongType, found.at)
	}
	return buildDataset(space, kind, place, pipeline)
}

// buildDataset parses the four messages of a dataset.
func buildDataset(space, kind, place, pipeline *message) (*dataset,
	error) {
	shape, err := readDataspace(space.data, space.at)
	if err != nil {
		return nil, err
	}
	typ, err := readDatatype(kind.data, kind.at)
	if err != nil {
		return nil, err
	}
	count, counted := elementCount(shape)
	if !counted {
		return nil, refuse(errCorrupt, space.at, "IV.A.2.b",
			"the dimensions %v hold more elements than a uint64 "+
				"counts", shape)
	}
	where, err := readLayout(*place, len(shape), typ.size)
	if err != nil {
		return nil, err
	}
	made := &dataset{description: description{shape: shape, kind: *typ},
		layout: where, count: count}
	if pipeline == nil {
		return made, nil
	}
	if !where.chunked {
		return nil, refuse(errCorrupt, pipeline.at, "IV.A.2.l",
			"a filter pipeline on data that is not chunked")
	}
	made.filters, err = readFilters(*pipeline)
	if err != nil {
		return nil, err
	}
	return made, nil
}

// raw is all of a dataset's elements as stored, little-endian, in the
// order of the file's C layout: the last dimension changes fastest.
func (set *dataset) raw(from *source) ([]byte, error) {
	if set.layout.chunked {
		return set.rawChunked(from)
	}
	return set.rawContiguous(from)
}

// rawContiguous reads a dataset stored as one run of bytes. IV.A.2.i,
// contiguous storage.
func (set *dataset) rawContiguous(from *source) ([]byte, error) {
	want, fits := byteCount(set.count, set.kind.size)
	if !fits || want > set.layout.size {
		return nil, refuse(errCorrupt, set.layout.at, "IV.A.2.i",
			"%d elements of %d bytes do not fit the %d bytes of "+
				"storage", set.count, set.kind.size,
			set.layout.size)
	}
	if want == noBytes {
		return []byte{}, nil
	}
	if set.layout.address == undefinedAddress {
		return nil, refuse(errUnsupported, set.layout.at, "IV.A.2.i",
			"a dataset with no storage allocated; the fill value "+
				"is not read")
	}
	return from.read(set.layout.address, want, "IV.B")
}

// byteCount is so many elements of so many bytes, or false if that is
// more than an int64 can count, which no file offset can be.
func byteCount(count uint64, size int) (uint64, bool) {
	if count > math.MaxInt64/uint64(size) {
		return noBytes, false
	}
	return count * uint64(size), true
}
