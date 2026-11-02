package hdf5

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"slices"
)

// The counts the chunk grid is walked in, each defined by counting.
const (
	oneChunk    = 1 // defined: a chunk, counted
	oneElement  = 1 // defined: an element, counted
	noFilter    = 0 // defined by III.A.1: a mask of no filter skipped
	firstPlace  = 0 // defined by a slice's order
	beyondLimit = 1 // defined: one byte past a limit shows it was passed
	lastAxis    = 1 // defined: the last dimension is rank less one
	rowAxes     = 2 // defined: the rows stop one short of the last axis
)

// rawChunked reads each indexed chunk, reverses its filters, and places it in
// the result (IV.A.2.i, III.A.1 node type 1, and IV.A.2.l).
//
// The index must cover the dataset; missing fill-value chunks are not
// implemented. Element counts must fit uint64, and declared byte counts must
// fit the file before any chunk allocation or read.
func (set *dataset) rawChunked(from *source) ([]byte, error) {
	elements, counted := elementCount(set.layout.chunk)
	chunkBytes, fits := byteCount(elements, set.kind.size)
	if !counted || !fits {
		return nil, fmt.Errorf("hdf5: file is damaged: chunks of %v "+
			"elements are too large, at address %#x (spec "+
			"IV.A.2.i)", set.layout.chunk, set.layout.at)
	}
	total, fits := byteCount(set.count, set.kind.size)
	if !fits {
		return nil, fmt.Errorf("hdf5: file is damaged: %d elements "+
			"of %d bytes are more than a file holds, at address "+
			"%#x (spec "+
			"IV.A.2.i)", set.count, set.kind.size, set.layout.at)
	}
	if set.count == noBytes {
		return []byte{}, nil
	}
	entries, err := from.chunkEntries(set.layout.address, noLevel,
		len(set.shape))
	if err != nil {
		return nil, err
	}
	if err := set.checkChunks(entries); err != nil {
		return nil, err
	}
	var out []byte
	for _, entry := range entries {
		data, err := set.unfilter(from, entry, chunkBytes)
		if err != nil {
			return nil, err
		}
		if out == nil {
			// The first chunk has come out whole, so the sizes the
			// file states are real, and the result can be made.
			out = make([]byte, total)
		}
		set.place(out, data, entry.origin)
	}
	return out, nil
}

// checkChunks requires exact grid coverage: no off-grid entries, duplicate
// chunks, or missing chunks.
func (set *dataset) checkChunks(entries []chunkEntry) error {
	grid := make([]uint64, len(set.shape))
	want := uint64(oneChunk)
	for axis, size := range set.shape {
		across := set.layout.chunk[axis]
		grid[axis] = (size + across - oneElement) / across
		want *= grid[axis]
	}
	seen := map[uint64]bool{}
	for _, entry := range entries {
		var index uint64
		for axis, at := range entry.origin {
			across := set.layout.chunk[axis]
			if at%across != noBytes || at >= set.shape[axis] {
				return set.offGrid(entry)
			}
			index = index*grid[axis] + at/across
		}
		if seen[index] {
			return fmt.Errorf("hdf5: file is damaged: the chunk "+
				"at %v is named twice, at address %#x (spec "+
				"III.A.1)", entry.origin, entry.at)
		}
		seen[index] = true
	}
	if uint64(len(seen)) != want {
		return fmt.Errorf("hdf5: part of the format not read: %d of "+
			"the %d chunks are stored; the fill value of the "+
			"rest is not read, at address %#x (spec "+
			"III.A.1)", len(seen), want, set.layout.address)
	}
	return nil
}

// offGrid reports a chunk origin outside the dataset’s chunk grid.
func (set *dataset) offGrid(entry chunkEntry) error {
	return fmt.Errorf("hdf5: file is damaged: a chunk at %v is not on "+
		"the grid of %v chunks of %v, at address %#x (spec "+
		"III.A.1)", entry.origin, set.shape, set.layout.chunk, entry.at)
}

// unfilter reads stored chunk bytes and reverses the filter pipeline in reverse
// order to recover element bytes.
func (set *dataset) unfilter(from *source, entry chunkEntry,
	chunkBytes uint64) ([]byte, error) {
	data, err := from.read(entry.address, uint64(entry.stored), "IV.B")
	if err != nil {
		return nil, err
	}
	if set.filters == nil {
		return set.sized(data, chunkBytes, entry)
	}
	if entry.mask != noFilter {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"filter mask %#x: a chunk with filters skipped, at "+
			"address %#x (spec "+
			"III.A.1)", entry.mask, entry.at)
	}
	backward := slices.Clone(set.filters.ids)
	slices.Reverse(backward)
	limit := chunkBytes + checksumSize
	for _, id := range backward {
		switch id {
		case filterDeflate:
			data, err = inflate(data, limit, entry)
		case filterFletcher32:
			data, err = verifyFletcher32(data, entry)
		}
		if err != nil {
			return nil, err
		}
	}
	return set.sized(data, chunkBytes, entry)
}

// sized requires the decoded bytes of one complete chunk.
func (set *dataset) sized(data []byte, want uint64,
	entry chunkEntry) ([]byte, error) {
	if uint64(len(data)) != want {
		return nil, fmt.Errorf("hdf5: file is damaged: the chunk at "+
			"%v is %d bytes unfiltered, %d were expected, at "+
			"address %#x (spec "+
			"IV.B)", entry.origin, len(data), want, entry.at)
	}
	return data, nil
}

// inflate reverses the zlib deflate filter (IV.A.2.l, filter ID 1). Output
// beyond the declared byte limit is corrupt data.
func inflate(data []byte, limit uint64, entry chunkEntry) ([]byte, error) {
	stream, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("hdf5: file is damaged: deflate "+
			"stream of the chunk at %v: %v, at address %#x (spec "+
			"IV.A.2.l)", entry.origin, err, entry.at)
	}
	bounded := io.LimitReader(stream, int64(limit)+beyondLimit)
	out, err := io.ReadAll(bounded)
	if err != nil {
		return nil, fmt.Errorf("hdf5: file is damaged: deflate "+
			"stream of the chunk at %v: %v, at address %#x (spec "+
			"IV.A.2.l)", entry.origin, err, entry.at)
	}
	if uint64(len(out)) > limit {
		return nil, fmt.Errorf("hdf5: file is damaged: deflate "+
			"stream of the chunk at %v gives more than %d bytes, "+
			"at address %#x (spec "+
			"IV.A.2.l)", entry.origin, limit, entry.at)
	}
	return out, nil
}

// place copies a decoded chunk at its dataset origin, clipping partial chunks
// at the dataset’s far edges.
func (set *dataset) place(out, chunk []byte, origin []uint64) {
	rank := len(set.shape)
	size := uint64(set.kind.size)
	last := rank - lastAxis
	// The run copied at a time is along the last dimension, the one that
	// changes fastest, cut where the dataset ends.
	run := min(set.layout.chunk[last], set.shape[last]-origin[last])
	index := make([]uint64, rank) // the row's position within the chunk
	for {
		inside := true
		var from, to uint64
		for axis := range rank {
			from = from*set.layout.chunk[axis] + index[axis]
			to = to*set.shape[axis] + origin[axis] + index[axis]
			reached := origin[axis] + index[axis]
			inside = inside && reached < set.shape[axis]
		}
		if inside {
			copy(out[to*size:(to+run)*size],
				chunk[from*size:(from+run)*size])
		}
		if !next(index, set.layout.chunk) {
			return
		}
	}
}

// next advances to the next chunk row, excluding the last dimension whose
// elements form the row. It returns false after the final row.
func next(index, chunk []uint64) bool {
	for axis := len(index) - rowAxes; axis >= firstPlace; axis-- {
		index[axis]++
		if index[axis] < chunk[axis] {
			return true
		}
		index[axis] = noBytes
	}
	return false
}
