//ninja:vow dojo each parser is held to bytes built by hand from the specification, and to those bytes changed at random, through the parsers, which hdf5 does not export
package hdf5

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"math"
	"math/big"
	"os"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// built is what testdata/parsers.json holds: bytes for each parser;
// files built by hand, by name; the datasets made of those files by the
// changes each case writes over them; and chunks too large to read.
type built struct {
	Cases    []parsed             `json:"cases"`
	Bases    map[string]string    `json:"bases"`
	Datasets map[string][]changed `json:"datasets"`
	TooLarge tooLarge             `json:"too large"`
	Shapes   []shapeCount         `json:"shapes"`
}

// shapeCount is a shape, the elements it holds, and whether a uint64
// counts them.
type shapeCount struct {
	Shape   []uint64 `json:"shape"`
	Count   uint64   `json:"count"`
	Counted bool     `json:"counted"`
}

// parsed is bytes for one parser, spelled out or a base with changes
// written over it, and the cause it must refuse them for, or none.
type parsed struct {
	Parser  string   `json:"parser"`
	Why     string   `json:"why"`
	Bytes   string   `json:"bytes"`
	Base    string   `json:"base"`
	Changes []change `json:"changes"`
	Cause   string   `json:"cause"`
}

// tooLarge is chunk shapes that count too many elements or bytes, and a
// chunk tree for them to be read against.
type tooLarge struct {
	Chunks [][]uint64 `json:"chunks"`
	Tree   string     `json:"tree"`
}

// changed is a base with bytes written over it; the cause reading it
// must be refused for, or none; and the kind its elements are read as,
// with what that gives or the cause it is refused for.
type changed struct {
	Why      string    `json:"why"`
	Changes  []change  `json:"changes"`
	Cause    string    `json:"cause"`
	As       string    `json:"as"`
	Integers []int64   `json:"integers"`
	Floats   []float64 `json:"floats"`
	Texts    []string  `json:"texts"`
	AsCause  string    `json:"as cause"`
}

// change is bytes written over the dataset at an offset.
type change struct {
	At    int    `json:"at"`
	Bytes string `json:"bytes"`
}

// builtBytes is what testdata/parsers.json holds, or the test stops.
func builtBytes(test *testing.T) built {
	test.Helper()
	var made built
	raw, err := os.ReadFile("testdata/parsers.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &made); err != nil {
		test.Fatal(err)
	}
	return made
}

// unhex is the bytes a run of hexadecimal pairs spells, spaces between
// them, or the test stops.
func unhex(test *testing.T, text string) []byte {
	test.Helper()
	pairs := strings.ReplaceAll(text, " ", "")
	raw, err := hex.DecodeString(pairs)
	if err != nil {
		test.Fatal(err)
	}
	return raw
}

// based is a base's bytes with changes written over them, or the test
// stops if there is no such base.
func (made built) based(test *testing.T, base string,
	changes []change) []byte {
	test.Helper()
	pairs, known := made.Bases[base]
	if !known {
		test.Fatalf("no base %q", base)
	}
	raw := unhex(test, pairs)
	for _, write := range changes {
		copy(raw[write.At:], unhex(test, write.Bytes))
	}
	return raw
}

// bytesOf is a case's bytes: spelled out, or its base changed.
func (made built) bytesOf(test *testing.T, each parsed) []byte {
	test.Helper()
	if each.Base == "" {
		return unhex(test, each.Bytes)
	}
	return made.based(test, each.Base, each.Changes)
}

// causes is the refusal each case's cause names.
var causes = map[string]error{"corrupt": errCorrupt,
	"unsupported": errUnsupported, "wrong type": errWrongType,
	"not found": errNotFound}

// judged is whether a parse did as its cause says: with no cause, it
// gave its result and no error; with one, it gave nothing and that
// refusal.
func judged(cause string, gave bool, err error) bool {
	if cause == "" {
		return gave && err == nil
	}
	return !gave && errors.Is(err, causes[cause])
}

// The fixed arguments the parsers are given, each defined by
// testdata/parsers.json's about.
const (
	start        = 0   // defined: every structure is at address 0
	oneDimension = 1   // defined: a dataset of one dimension
	eightBytes   = 8   // defined: elements of eight bytes
	leafOfOne    = 1   // defined: a node holds at most two symbols
	firstObject  = 1   // defined: a collection's object 1
	inflateLimit = 8   // defined: inflate gives at most eight bytes
	groupHeap    = 96  // defined: the group base's heap is at 96
	attributeA   = "A" // defined: the attribute the cases name
)

// fileOf is a file of the bytes.
func fileOf(raw []byte) *source {
	return &source{reader: bytes.NewReader(raw), size: uint64(len(raw))}
}

// parsers is each parser the cases name, taking the case's bytes and
// saying whether it gave its result.
var parsers = map[string]func([]byte) (bool, error){
	"dataspace": dataspaceOf, "datatype": datatypeOf,
	"filters": filtersOf, "layout": layoutOf,
	"symbol table": symbolTableOf, "continuation": continuationOf,
	"chunk key": chunkKeyOf, "objects": objectsOf,
	"fletcher32": fletcherOf, "inflate": inflateOf,
	"attribute": attributeOf, "heap": heapOf, "name": nameOf,
	"node": nodeOf, "group tree": groupTreeOf,
	"collection": collectionOf, "chunk tree": chunkTreeOf,
	"header": headerOf, "attributes": attributesOf, "members": membersOf,
}

// attributesOf is attribute A of an object whose one message is an
// Attribute message of the bytes.
func attributesOf(raw []byte) (bool, error) {
	found := &object{messages: []message{{kind: messageAttribute,
		data: raw}}}
	made, err := attributeNamed(found, attributeA)
	return made != nil, err
}

// membersOf is the members of the group of the tree at the start of a
// file of the bytes and the heap at 96.
func membersOf(raw []byte) (bool, error) {
	table := groupTable{tree: start, heap: groupHeap}
	links, err := fileOf(raw).members(table, leafOfOne)
	return links != nil, err
}

// dataspaceOf parses a Dataspace message.
func dataspaceOf(raw []byte) (bool, error) {
	shape, err := readDataspace(raw, start)
	return shape != nil, err
}

// datatypeOf parses a Datatype message.
func datatypeOf(raw []byte) (bool, error) {
	kind, err := readDatatype(raw, start)
	return kind != nil, err
}

// filtersOf parses a Filter Pipeline message.
func filtersOf(raw []byte) (bool, error) {
	pipeline, err := readFilters(message{data: raw})
	return pipeline != nil, err
}

// layoutOf parses a Data Layout message for one dimension of eight-byte
// elements.
func layoutOf(raw []byte) (bool, error) {
	where, err := readLayout(message{data: raw}, oneDimension, eightBytes)
	return where != nil, err
}

// symbolTableOf parses a Symbol Table message.
func symbolTableOf(raw []byte) (bool, error) {
	table, err := parseSymbolTable(message{data: raw})
	return table != nil, err
}

// continuationOf parses an Object Header Continuation message.
func continuationOf(raw []byte) (bool, error) {
	next, err := continuation(raw, start)
	return next != nil, err
}

// chunkKeyOf parses a chunk B-tree key of one dimension.
func chunkKeyOf(raw []byte) (bool, error) {
	entry, err := chunkKey(raw, oneDimension, start)
	return entry != nil, err
}

// objectsOf parses the objects of a global heap collection's body.
func objectsOf(raw []byte) (bool, error) {
	made, err := parseObjects(raw, start)
	return made != nil, err
}

// fletcherOf checks the Fletcher-32 sum that ends the bytes.
func fletcherOf(raw []byte) (bool, error) {
	body, err := verifyFletcher32(raw, chunkEntry{})
	return body != nil, err
}

// inflateOf inflates a zlib stream of at most eight bytes.
func inflateOf(raw []byte) (bool, error) {
	out, err := inflate(raw, inflateLimit, chunkEntry{})
	return out != nil, err
}

// attributeOf parses an Attribute message.
func attributeOf(raw []byte) (bool, error) {
	made, err := readAttribute(message{data: raw})
	return made != nil, err
}

// heapOf reads the local heap at the start of a file of the bytes.
func heapOf(raw []byte) (bool, error) {
	store, err := fileOf(raw).readHeap(start)
	return store != nil, err
}

// nameOf is the name at the start of a heap of the bytes.
func nameOf(raw []byte) (bool, error) {
	name, err := (&heap{data: raw}).name(start, start)
	return name != "", err
}

// nodeOf reads the symbol table node at the start of a file of the
// bytes.
func nodeOf(raw []byte) (bool, error) {
	symbols, err := fileOf(raw).readNode(start, leafOfOne)
	return symbols != nil, err
}

// groupTreeOf reads the group B-tree at the start of a file of the
// bytes, at whatever level it says.
func groupTreeOf(raw []byte) (bool, error) {
	symbols, err := fileOf(raw).groupSymbols(start, noLevel, leafOfOne)
	return symbols != nil, err
}

// collectionOf is object 1 of the global heap collection at the start
// of a file of the bytes.
func collectionOf(raw []byte) (bool, error) {
	store := newGlobalHeap(fileOf(raw))
	data, err := store.object(start, firstObject)
	return data != nil, err
}

// chunkTreeOf reads the chunk B-tree of one dimension at the start of a
// file of the bytes, at whatever level it says.
func chunkTreeOf(raw []byte) (bool, error) {
	entries, err := fileOf(raw).chunkEntries(start, noLevel, oneDimension)
	return entries != nil, err
}

// headerOf reads the object header at the start of a file of the bytes.
func headerOf(raw []byte) (bool, error) {
	found, err := readObject(fileOf(raw), start)
	return found != nil, err
}

// The test and the anti-test, for each parser: bytes built from the
// specification give their result, and bytes that break it are refused
// for the cause the case names, and give nothing.
func TestEachParserTakesItsBytesAndRefusesTheRest(test *testing.T) {
	made := builtBytes(test)
	for _, each := range made.Cases {
		parse, known := parsers[each.Parser]
		if !known {
			test.Fatalf("no parser %q", each.Parser)
		}
		gave, err := parse(made.bytesOf(test, each))
		if !judged(each.Cause, gave, err) {
			test.Errorf("%s, %s: gave %v, %v", each.Parser, each.Why, gave,
				err)
		}
	}
}

// The test and the anti-test, for whole datasets: each case's changes
// written over its base read as the case says, or are refused for its
// cause and give no values; and its elements, read as the case's kind,
// are what it says, or are refused for its cause and give none.
func TestTheBuiltDatasetsReadAsTheirChangesSay(test *testing.T) {
	made := builtBytes(test)
	for base, cases := range made.Datasets {
		for _, each := range cases {
			from := fileOf(made.based(test, base, each.Changes))
			found, err := readObject(from, start)
			if err != nil {
				test.Fatalf("%s, %s: the header is refused: %v", base,
					each.Why, err)
			}
			held, err := (&File{from: from}).valuesOf(found)
			if each.Cause != "" || err != nil {
				if !judged(each.Cause, held != nil, err) {
					test.Errorf("%s, %s: %v, %v", base, each.Why, held, err)
				}
				continue
			}
			each.check(test, held)
		}
	}
}

// The anti-test: chunks of more elements than a uint64 counts, or of
// more bytes than a file holds, are refused as damaged before any chunk
// is read, and give no bytes.
func TestChunksTooLargeToCountAreRefused(test *testing.T) {
	large := builtBytes(test).TooLarge
	from := fileOf(unhex(test, large.Tree))
	for _, chunk := range large.Chunks {
		shape := slices.Repeat([]uint64{oneElement}, len(chunk))
		set := &dataset{description: description{shape: shape,
			kind: datatype{class: integerClass, size: byteWide}},
			layout: &layout{chunked: true, chunk: chunk},
			count:  oneElement}
		raw, err := set.rawChunked(from)
		if raw != nil || !errors.Is(err, errCorrupt) {
			test.Errorf("chunks of %v: %v, %v", chunk, raw, err)
		}
	}
}

// check holds the elements, read as the case's kind, to what it says.
func (want changed) check(test *testing.T, held *Values) {
	test.Helper()
	var gave, same bool
	var err error
	switch want.As {
	case "integers":
		got, refused := held.Integers()
		gave, same, err = got != nil, slices.Equal(got, want.Integers),
			refused
	case "floats":
		got, refused := held.Floats()
		gave, same, err = got != nil, slices.Equal(got, want.Floats),
			refused
	case "texts":
		got, refused := held.Texts()
		gave, same, err = got != nil, slices.Equal(got, want.Texts), refused
	default:
		test.Fatalf("%s: no kind %q", want.Why, want.As)
	}
	if want.AsCause != "" {
		if !judged(want.AsCause, gave, err) {
			test.Errorf("%s: as %s, %v, %v", want.Why, want.As, gave, err)
		}
		return
	}
	if err != nil || !same {
		test.Errorf("%s: as %s, same %v, %v", want.Why, want.As, same, err)
	}
}

// The test and the anti-test: each shape holds the elements it says, and
// one whose elements are more than a uint64 holds is not counted.
func TestEachShapeHoldsItsElements(test *testing.T) {
	for _, want := range builtBytes(test).Shapes {
		count, counted := elementCount(want.Shape)
		if count != want.Count || counted != want.Counted {
			test.Errorf("%v holds %d, %v", want.Shape, count, counted)
		}
	}
}

// backwards is a count below nothing. Defined.
const backwards = -1

// The anti-test: a cursor asked for fewer bytes than none gives none,
// and is short.
func TestACursorRefusesACountBelowNothing(test *testing.T) {
	walk := &cursor{data: signature}
	if got := walk.take(backwards); got != nil || !walk.short {
		test.Errorf("took %v, short %v", got, walk.short)
	}
}

// The draws the properties make, each defined: how many bytes a draw
// changes in a built file, and how many words a summed body holds.
const (
	fewestChanges = 1  // defined: at least one byte changed
	mostChanges   = 4  // defined: at most four
	fewestWords   = 1  // defined: a body of at least one word
	mostWords     = 32 // defined: and at most thirty-two
	mostRank      = 4  // defined: shapes of up to four dimensions
	firstByte     = 0  // defined: a body's first byte
	someBit       = 1  // defined: the least change a byte can take
	everyBit      = math.MaxUint8
)

// The property: any bytes written over a built file, anywhere, read
// without a panic, and every refusal gives nothing.
func TestChangedBytesNeverPanicAndRefusalsGiveNothing(test *testing.T) {
	made := builtBytes(test)
	bases := map[string][]byte{}
	for name := range made.Bases {
		bases[name] = made.based(test, name, nil)
	}
	rapid.Check(test, scrambling{bases: bases}.check)
}

// scrambling writes drawn bytes over a drawn base and reads it whole.
type scrambling struct{ bases map[string][]byte }

// check draws a base and the bytes to change, and reads it as a group
// or as a dataset.
func (scramble scrambling) check(test *rapid.T) {
	names := slices.Sorted(maps.Keys(scramble.bases))
	name := rapid.SampledFrom(names).Draw(test, "base")
	raw := slices.Clone(scramble.bases[name])
	count := rapid.IntRange(fewestChanges, mostChanges).Draw(test, "count")
	last := len(raw) - oneElement
	for range count {
		at := rapid.IntRange(start, last).Draw(test, "at")
		raw[at] = rapid.Byte().Draw(test, "byte")
	}
	if name == "group" {
		table := groupTable{tree: start, heap: groupHeap}
		links, err := fileOf(raw).members(table, leafOfOne)
		either(test, "members", links, err)
		return
	}
	readAll(test, fileOf(raw))
}

// readAll reads the dataset at the start of a file as every kind.
func readAll(test *rapid.T, from *source) {
	found, err := readObject(from, start)
	eitherOne(test, "header", found, err)
	if err != nil {
		return
	}
	held, err := (&File{from: from}).valuesOf(found)
	eitherOne(test, "values", held, err)
	if err != nil {
		return
	}
	floats, err := held.Floats()
	either(test, "floats", floats, err)
	integers, err := held.Integers()
	either(test, "integers", integers, err)
	texts, err := held.Texts()
	either(test, "texts", texts, err)
}

// either fails a draw whose read gave both elements and an error, or
// neither.
func either[Element any](test *rapid.T, what string, got []Element,
	err error) {
	if (got == nil) == (err == nil) {
		test.Fatalf("%s: gave %v and %v", what, got, err)
	}
}

// eitherOne fails a draw whose read gave both a result and an error, or
// neither.
func eitherOne[Result any](test *rapid.T, what string, got *Result,
	err error) {
	if (got == nil) == (err == nil) {
		test.Fatalf("%s: gave %v and %v", what, got, err)
	}
}

// The property: a Fletcher-32 sum checks the body it was taken over,
// and any one byte of that body changed is refused as damaged.
func TestAnyByteChangedUnderASumIsFound(test *testing.T) {
	rapid.Check(test, summing{}.check)
}

// summing draws a body, sums it, and changes a byte of it.
type summing struct{}

// check draws an even body, sums it as the filter stores a sum,
// little-endian after it, and draws the byte to change.
func (summing) check(test *rapid.T) {
	words := rapid.IntRange(fewestWords, mostWords).Draw(test, "words")
	size := words * evenWords
	body := rapid.SliceOfN(rapid.Byte(), size, size).Draw(test, "body")
	summed := binary.LittleEndian.AppendUint32(slices.Clone(body),
		fletcher32(body))
	got, err := verifyFletcher32(summed, chunkEntry{})
	if err != nil || !slices.Equal(got, body) {
		test.Fatalf("a body and its own sum: %v, %v", got, err)
	}
	at := rapid.IntRange(firstByte, size-oneElement).Draw(test, "at")
	turn := rapid.ByteRange(someBit, everyBit).Draw(test, "turn")
	summed[at] ^= turn
	if got, err := verifyFletcher32(summed, chunkEntry{}); got != nil ||
		!errors.Is(err, errCorrupt) {
		test.Fatalf("byte %d turned by %#x: %v, %v", at, turn, got, err)
	}
}

// The property: the elements a shape counts are the product of its
// sizes, exactly, and the count says it is not counted only when that
// product is more than a uint64 holds.
func TestAShapeCountsItsProduct(test *testing.T) {
	rapid.Check(test, counting{}.check)
}

// counting draws a shape and counts it two ways.
type counting struct{}

// check draws a shape and holds its count to the product of big
// integers.
func (counting) check(test *rapid.T) {
	shape := rapid.SliceOfN(rapid.Uint64(), noBytes, mostRank).Draw(test,
		"shape")
	product := big.NewInt(oneElement)
	for _, size := range shape {
		product.Mul(product, new(big.Int).SetUint64(size))
	}
	fits := product.IsUint64()
	count, counted := elementCount(shape)
	if counted != fits || (fits && count != product.Uint64()) {
		test.Fatalf("%v counts %d, %v; its product is %v", shape, count,
			counted, product)
	}
}
