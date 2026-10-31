package hdf5

import (
	"bytes"
	"compress/zlib"
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

// rawChunked reads a chunked dataset: every chunk named by the B-tree is
// read, unfiltered and placed in the result. IV.A.2.i chunked storage,
// III.A.1 node type 1, IV.A.2.l.
//
// It refuses a dataset with a chunk the tree does not name, whose bytes
// would be the fill value; and, before any chunk is read, chunks or a
// dataset of more elements than a uint64 counts or more bytes than a
// file holds.
func (set *dataset) rawChunked(from *source) ([]byte, error) {
	elements, counted := elementCount(set.layout.chunk)
	chunkBytes, fits := byteCount(elements, set.kind.size)
	if !counted || !fits {
		return nil, refuse(errCorrupt, set.layout.at, "IV.A.2.i",
			"chunks of %v elements are too large", set.layout.chunk)
	}
	total, fits := byteCount(set.count, set.kind.size)
	if !fits {
		return nil, refuse(errCorrupt, set.layout.at, "IV.A.2.i",
			"%d elements of %d bytes are more than a file holds",
			set.count, set.kind.size)
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

// checkChunks refuses a set of chunk entries that does not cover the
// dataset exactly once: an entry off the chunk grid, a chunk named
// twice, or a chunk of the grid that is not named.
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
			return refuse(errCorrupt, entry.at, "III.A.1",
				"the chunk at %v is named twice", entry.origin)
		}
		seen[index] = true
	}
	if uint64(len(seen)) != want {
		return refuse(errUnsupported, set.layout.address, "III.A.1",
			"%d of the %d chunks are stored; the fill value of "+
				"the rest is not read", len(seen), want)
	}
	return nil
}

// offGrid is the refusal of a chunk entry that is not on the dataset's
// grid of chunks.
func (set *dataset) offGrid(entry chunkEntry) error {
	return refuse(errCorrupt, entry.at, "III.A.1",
		"a chunk at %v is not on the grid of %v chunks of %v",
		entry.origin, set.shape, set.layout.chunk)
}

// unfilter reads one chunk's stored bytes and undoes the pipeline, last
// filter first, so that the chunk comes out as so many bytes of
// elements.
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
		return nil, refuse(errUnsupported, entry.at, "III.A.1",
			"filter mask %#x: a chunk with filters skipped",
			entry.mask)
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

// sized checks that a chunk, unfiltered, is exactly the bytes of a whole
// chunk of elements.
func (set *dataset) sized(data []byte, want uint64,
	entry chunkEntry) ([]byte, error) {
	if uint64(len(data)) != want {
		return nil, refuse(errCorrupt, entry.at, "IV.B",
			"the chunk at %v is %d bytes unfiltered, %d were "+
				"expected", entry.origin, len(data), want)
	}
	return data, nil
}

// inflate undoes the deflate filter, whose output is zlib's format
// (IV.A.2.l, filter id 1). It reads no more than a limit of bytes: a
// stream that would give more is damaged.
func inflate(data []byte, limit uint64, entry chunkEntry) ([]byte, error) {
	stream, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, refuse(errCorrupt, entry.at, "IV.A.2.l",
			"deflate stream of the chunk at %v: %v", entry.origin,
			err)
	}
	bounded := io.LimitReader(stream, int64(limit)+beyondLimit)
	out, err := io.ReadAll(bounded)
	if err != nil {
		return nil, refuse(errCorrupt, entry.at, "IV.A.2.l",
			"deflate stream of the chunk at %v: %v", entry.origin,
			err)
	}
	if uint64(len(out)) > limit {
		return nil, refuse(errCorrupt, entry.at, "IV.A.2.l",
			"deflate stream of the chunk at %v gives more than "+
				"%d bytes", entry.origin, limit)
	}
	return out, nil
}

// place copies one chunk, whole, into the result at the chunk's origin.
// A chunk on the far edge of the dataset reaches past it, and only the
// part inside is copied.
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

// next steps an index to the next row of a chunk, the last dimension
// being the row itself and so not stepped. It says false when the rows
// are done.
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
