package hdf5

import (
	"fmt"
	"math"
)

// description records dimension sizes and element type.
type description struct {
	// shape is the size of each dimension. A scalar has no dimensions,
	// and holds one element.
	shape []uint64
	// kind is the type of one element.
	kind datatype
}

// dataset combines its parsed description with element storage.
type dataset struct {
	description
	layout  *layout
	filters *filters
	// count is the number of elements.
	count uint64
}

// datasetOf requires dataspace, datatype, and layout messages, plus an optional
// filter pipeline. Objects missing the required messages are not readable
// datasets.
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
		return nil, fmt.Errorf("hdf5: not that kind of value: the "+
			"object at %#x is not a dataset", found.at)
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
		return nil, fmt.Errorf("hdf5: file is damaged: the "+
			"dimensions %v hold more elements than a uint64 "+
			"counts, at address %#x (spec "+
			"IV.A.2.b)", shape, space.at)
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
		return nil, fmt.Errorf("hdf5: file is damaged: a filter "+
			"pipeline on data that is not chunked, at address "+
			"%#x (spec IV.A.2.l)", pipeline.at)
	}
	made.filters, err = readFilters(*pipeline)
	if err != nil {
		return nil, err
	}
	return made, nil
}

// raw returns stored element bytes in little-endian C order, where the last
// dimension changes fastest.
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
		return nil, fmt.Errorf("hdf5: file is damaged: %d elements "+
			"of %d bytes do not fit the %d bytes of storage, at "+
			"address %#x (spec IV.A.2.i)", set.count, set.kind.size,
			set.layout.size, set.layout.at)
	}
	if want == noBytes {
		return []byte{}, nil
	}
	if set.layout.address == undefinedAddress {
		return nil, fmt.Errorf("hdf5: part of the format not read: a "+
			"dataset with no storage allocated; the fill value "+
			"is not read, at address %#x (spec "+
			"IV.A.2.i)", set.layout.at)
	}
	return from.read(set.layout.address, want, "IV.B")
}

// byteCount computes storage size and reports false on int64 overflow.
func byteCount(count uint64, size int) (uint64, bool) {
	if count > math.MaxInt64/uint64(size) {
		return noBytes, false
	}
	return count * uint64(size), true
}
