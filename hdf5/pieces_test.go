//ninja:vow dojo a short reader is its own error, a header keeps what is read, bytes are counted in an int64, a grid is covered once and a chunk is placed at its origin, through the readers, readObject, byteCount, checkChunks and place, which hdf5 does not export
package hdf5

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"
)

// shortReads is a size files are said to be, and the cases whose bytes
// their readers hold, each short of it.
type shortReads struct {
	Said  uint64   `json:"said"`
	Cases []parsed `json:"cases"`
}

// keptKinds is an object header, and the types of the messages kept of
// it, in order.
type keptKinds struct {
	Why   string   `json:"why"`
	Bytes string   `json:"bytes"`
	Kinds []uint16 `json:"kinds"`
}

// byteCounted is elements of a size, the bytes they are, and whether an
// int64 counts them.
type byteCounted struct {
	Count uint64 `json:"count"`
	Size  int    `json:"size"`
	Bytes uint64 `json:"bytes"`
	Fits  bool   `json:"fits"`
}

// gridded is a dataset's shape in chunks of a shape, the origins of the
// chunks named, and the cause they are refused for, or none.
type gridded struct {
	Why     string     `json:"why"`
	Shape   []uint64   `json:"shape"`
	Chunk   []uint64   `json:"chunk"`
	Origins [][]uint64 `json:"origins"`
	Cause   string     `json:"cause"`
}

// placement is a dataset's shape in chunks of a shape, of elements of a
// size; its chunks and their origins; and the bytes they make, placed.
type placement struct {
	Why    string     `json:"why"`
	Shape  []uint64   `json:"shape"`
	Chunk  []uint64   `json:"chunk"`
	Size   int        `json:"size"`
	Chunks []atOrigin `json:"chunks"`
	Placed string     `json:"placed"`
}

// atOrigin is a chunk's bytes and its origin.
type atOrigin struct {
	Origin []uint64 `json:"origin"`
	Bytes  string   `json:"bytes"`
}

// The anti-test: in a file said to be longer than the bytes its reader
// holds, each structure that reaches past them gives the reader's own
// error, not a refusal of the file, and gives nothing.
func TestAReaderShortOfItsFileIsItsOwnError(test *testing.T) {
	short := builtBytes(test).Short
	for _, each := range short.Cases {
		read, known := filed[each.Parser]
		if !known {
			test.Fatalf("no file parser %q", each.Parser)
		}
		raw := unhex(test, each.Bytes)
		from := &source{reader: bytes.NewReader(raw), size: short.Said}
		gave, err := read(from)
		// every refusal of the file names the address it was refused at
		if gave || !errors.Is(err, io.EOF) || says(err, "at address") {
			test.Errorf("%s, %s: gave %v, %v", each.Parser, each.Why, gave,
				err)
		}
	}
}

// The test and the anti-test: a header keeps every message but those it
// passes over, in order, and passes over the nil, fill value and
// modification time messages, which change nothing it reads.
func TestAHeaderKeepsAllButWhatItPassesOver(test *testing.T) {
	for _, want := range builtBytes(test).Kept {
		raw := unhex(test, want.Bytes)
		found, err := readObject(fileOf(raw), start)
		if err != nil {
			test.Fatalf("%s: %v", want.Why, err)
		}
		var kinds []uint16
		for _, msg := range found.messages {
			kinds = append(kinds, msg.kind)
		}
		if !slices.Equal(kinds, want.Kinds) {
			test.Errorf("%s: kept %v", want.Why, kinds)
		}
	}
}

// The test and the anti-test: elements are counted in bytes up to the
// most an int64 holds, exactly, and one more element is not counted.
func TestEachByteCountIsItsProductOrNone(test *testing.T) {
	for _, want := range builtBytes(test).ByteCounts {
		got, fits := byteCount(want.Count, want.Size)
		if got != want.Bytes || fits != want.Fits {
			test.Errorf("%d of %d bytes: %d, %v", want.Count, want.Size,
				got, fits)
		}
	}
}

// The test and the anti-test: chunks that name each place of a dataset's
// grid once cover it; a chunk named twice is refused as damaged, and one
// not named as a fill value, which is not read.
func TestEachGridIsCoveredOnce(test *testing.T) {
	for _, want := range builtBytes(test).Grids {
		set := &dataset{description: description{shape: want.Shape},
			layout: &layout{chunk: want.Chunk}}
		var entries []chunkEntry
		for _, origin := range want.Origins {
			entries = append(entries, chunkEntry{origin: origin})
		}
		err := set.checkChunks(entries)
		if !judged(want.Cause, err == nil, err) {
			test.Errorf("%s: %v", want.Why, err)
		}
	}
}

// The test and the anti-test: each chunk's elements land at its origin,
// in the dataset's order, and its padding past the far edges lands
// nowhere.
func TestEachChunkIsPlacedAtItsOrigin(test *testing.T) {
	for _, want := range builtBytes(test).Placements {
		set := &dataset{description: description{shape: want.Shape,
			kind: datatype{size: want.Size}},
			layout: &layout{chunk: want.Chunk}}
		placed := unhex(test, want.Placed)
		out := make([]byte, len(placed))
		for _, each := range want.Chunks {
			set.place(out, unhex(test, each.Bytes), each.Origin)
		}
		if !bytes.Equal(out, placed) {
			test.Errorf("%s: placed % x", want.Why, out)
		}
	}
}
