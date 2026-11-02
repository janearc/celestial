//ninja:vow dojo each parser is held to bytes built by hand from the specification, and to those bytes changed at random, through the parsers, which hdf5 does not export
package hdf5

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/big"
	"os"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// built describes parser byte cases, hand-built files, dataset mutations,
// oversized-chunk cases, and the cases in pieces_test.go from
// testdata/parsers.json.
type built struct {
	Cases      []parsed             `json:"cases"`
	Bases      map[string]string    `json:"bases"`
	Datasets   map[string][]changed `json:"datasets"`
	TooLarge   tooLarge             `json:"too large"`
	Shapes     []shapeCount         `json:"shapes"`
	Short      shortReads           `json:"short reads"`
	Kept       []keptKinds          `json:"kept"`
	ByteCounts []byteCounted        `json:"byte counts"`
	Grids      []gridded            `json:"grids"`
	Placements []placement          `json:"placements"`
}

// shapeCount is a shape, the elements it holds, and whether a uint64
// counts them.
type shapeCount struct {
	Shape   []uint64 `json:"shape"`
	Count   uint64   `json:"count"`
	Counted bool     `json:"counted"`
}

// parsed is bytes for one parser, spelled out or a base with changes
// written over it; the cause it must refuse them for, or none; and where
// and in what words, if the case says.
type parsed struct {
	Parser  string   `json:"parser"`
	Why     string   `json:"why"`
	Bytes   string   `json:"bytes"`
	Base    string   `json:"base"`
	Changes []change `json:"changes"`
	Cause   string   `json:"cause"`
	Refusal *placed  `json:"refusal"`
}

// placed is where a refusal must say it met the bytes, and what it must
// say of them: an address, a section and words, each if given.
type placed struct {
	At      *uint64 `json:"at"`
	Section string  `json:"section"`
	Says    string  `json:"says"`
}

// tooLarge is chunk shapes that count too many elements or bytes, and a
// chunk tree for them to be read against.
type tooLarge struct {
	Chunks [][]uint64 `json:"chunks"`
	Tree   string     `json:"tree"`
}

// changed is a base with bytes written over it; the cause reading it
// must be refused for, or none, and where; and the kind its elements are
// read as, with what that gives or the cause it is refused for.
type changed struct {
	Why      string    `json:"why"`
	Changes  []change  `json:"changes"`
	Cause    string    `json:"cause"`
	Refusal  *placed   `json:"refusal"`
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

// causes is the words each case's cause puts in its refusal.
var causes = map[string]string{"corrupt": damaged,
	"unsupported": "hdf5: part of the format not read",
	"wrong type":  "hdf5: not that kind of value",
	"not found":   "hdf5: not found"}

// damaged is the words of a refusal of a damaged file.
const damaged = "hdf5: file is damaged"

// says is whether an error is one and holds the words.
func says(err error, words string) bool {
	return err != nil && strings.Contains(err.Error(), words)
}

// judged checks a parser’s result against the expected outcome. Success
// requires a result without an error; failure requires no result and an
// error containing the expected words.
func judged(cause string, gave bool, err error) bool {
	if cause == "" {
		return gave && err == nil
	}
	return !gave && says(err, causes[cause])
}

// placedAs is whether an error is the refusal a case places: with none
// placed, any error; with one, a refusal naming its address and its
// section and holding its words, each where the case gives it.
func placedAs(want *placed, err error) bool {
	if want == nil {
		return true
	}
	return err != nil &&
		(want.At == nil || says(err, fmt.Sprintf("at address %#x",
			*want.At))) &&
		(want.Section == "" || says(err, "(spec "+want.Section+")")) &&
		says(err, want.Says)
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

// parsers is each parser the cases name that takes bytes, saying whether
// it gave its result.
var parsers = map[string]func([]byte) (bool, error){
	"dataspace": dataspaceOf, "datatype": datatypeOf,
	"filters": filtersOf, "layout": layoutOf,
	"symbol table": symbolTableOf, "continuation": continuationOf,
	"chunk key": chunkKeyOf, "objects": objectsOf,
	"fletcher32": fletcherOf, "inflate": inflateOf,
	"attribute": attributeOf, "name": nameOf, "attributes": attributesOf,
}

// filed is each parser the cases name that reads through a file, saying
// whether it gave its result.
var filed = map[string]func(*source) (bool, error){
	"heap": heapOf, "node": nodeOf, "group tree": groupTreeOf,
	"collection": collectionOf, "chunk tree": chunkTreeOf,
	"header": headerOf, "members": membersOf,
}

// parseWith gives a case's bytes to the parser it names, one that takes
// bytes or one that reads a file of them; and says whether it gave its
// result, and whether a parser has that name.
func parseWith(name string, raw []byte) (gave, known bool, err error) {
	if parse, found := parsers[name]; found {
		gave, err = parse(raw)
		return gave, found, err
	}
	if read, found := filed[name]; found {
		gave, err = read(fileOf(raw))
		return gave, found, err
	}
	return false, false, nil
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
// file and the heap at 96.
func membersOf(from *source) (bool, error) {
	table := groupTable{tree: start, heap: groupHeap}
	links, err := from.members(table, leafOfOne)
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

// heapOf reads the local heap at the start of a file.
func heapOf(from *source) (bool, error) {
	store, err := from.readHeap(start)
	return store != nil, err
}

// nameOf is the name at the start of a heap of the bytes.
func nameOf(raw []byte) (bool, error) {
	name, err := (&heap{data: raw}).name(start, start)
	return name != "", err
}

// nodeOf reads the symbol table node at the start of a file.
func nodeOf(from *source) (bool, error) {
	symbols, err := from.readNode(start, leafOfOne)
	return symbols != nil, err
}

// groupTreeOf reads the group B-tree at the start of a file, at whatever
// level it says.
func groupTreeOf(from *source) (bool, error) {
	symbols, err := from.groupSymbols(start, noLevel, leafOfOne)
	return symbols != nil, err
}

// collectionOf is object 1 of the global heap collection at the start
// of a file.
func collectionOf(from *source) (bool, error) {
	store := newGlobalHeap(from)
	data, err := store.object(start, firstObject)
	return data != nil, err
}

// chunkTreeOf reads the chunk B-tree of one dimension at the start of a
// file, at whatever level it says.
func chunkTreeOf(from *source) (bool, error) {
	entries, err := from.chunkEntries(start, noLevel, oneDimension)
	return entries != nil, err
}

// headerOf reads the object header at the start of a file.
func headerOf(from *source) (bool, error) {
	found, err := readObject(from, start)
	return found != nil, err
}

// The test and anti-test: each parser accepts specification-built bytes and
// rejects malformed cases with no result. Errors must identify the expected
// address, specification section, and reason in words.
func TestEachParserTakesItsBytesAndRefusesTheRest(test *testing.T) {
	made := builtBytes(test)
	for _, each := range made.Cases {
		gave, known, err := parseWith(each.Parser, made.bytesOf(test, each))
		if !known {
			test.Fatalf("no parser %q", each.Parser)
		}
		if !judged(each.Cause, gave, err) || !placedAs(each.Refusal, err) {
			test.Errorf("%s, %s: gave %v, %v", each.Parser, each.Why, gave,
				err)
		}
	}
}

// The test and the anti-test, for whole datasets: each case's changes
// written over its base read as the case says, or are refused for its
// cause, where it places the refusal, and give no values; and its
// elements, read as the case's kind, are what it says, or are refused
// for its cause and give none.
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
				if !judged(each.Cause, held != nil, err) ||
					!placedAs(each.Refusal, err) {
					test.Errorf("%s, %s: %v, %v", base, each.Why, held, err)
				}
				continue
			}
			each.check(test, held)
		}
	}
}

// The anti-test: reject overflowing element counts and byte counts larger than
// the file before reading or allocating chunk data.
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
		if raw != nil || !says(err, damaged) {
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

// The property: arbitrary mutations of a built file never panic, and failed
// reads return no partial result.
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
		!says(err, damaged) {
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
