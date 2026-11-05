package fits_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/fits"
)

// The values the pixel tests store and read back.
const (
	scale  = 2   // defined: the BSCALE the integer images carry
	offset = 10  // defined: the BZERO the integer images carry
	blank  = -1  // defined: the BLANK the integer images carry
	stored = 7   // defined: the value the first pixel stores
	wide   = 2   // defined: the pixels a row in the test images
	tall   = 1   // defined: the rows in the test images
	row    = 0   // defined: the only row
	first  = 0   // defined: the first pixel
	second = 1   // defined: the second pixel
	half   = 0.5 // defined: a float the float images store
)

// image is a test image of one row: the first pixel stores a value and
// the second the BLANK, both written as the BITPIX says.
func image(t *testing.T, bitpix int) *fits.Image {
	t.Helper()
	var data bytes.Buffer
	cards := []string{card("SIMPLE", "T"),
		card("BITPIX", strconv.Itoa(bitpix)), card("NAXIS", "2"),
		card("NAXIS1", strconv.Itoa(wide)),
		card("NAXIS2", strconv.Itoa(tall))}
	switch bitpix {
	case asSingle:
		nan := float32(math.NaN())
		binary.Write(&data, binary.BigEndian, []float32{half,
			nan})
	case asDouble:
		binary.Write(&data, binary.BigEndian, []float64{half,
			math.NaN()})
	default:
		blankText := strconv.Itoa(blankOf(bitpix))
		scaleText, zeroText := strconv.Itoa(scale), strconv.Itoa(offset)
		cards = append(cards, card("BSCALE", scaleText),
			card("BZERO", zeroText),
			card("BLANK", blankText))
		for _, value := range []int64{stored, int64(blankOf(bitpix))} {
			writeStored(&data, bitpix, value)
		}
	}
	raw := data.Bytes()
	read, err := fits.Open(fileOf(cards, raw))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// blankOf is the BLANK a test image of a BITPIX carries. Unsigned bytes
// cannot hold -1, so it carries its largest value.
func blankOf(bitpix int) int {
	if bitpix == byteWide {
		return math.MaxUint8
	}
	return blank
}

// byteWide is the BITPIX of an unsigned byte. It's defined by the FITS
// Standard 4.0, table 8.
const byteWide = 8

// The other kinds of pixel, by BITPIX. Each is defined by the FITS
// Standard 4.0, table 8.
const (
	asShort  = 16  // defined: a 16-bit integer
	asInt    = 32  // defined: a 32-bit integer
	asLong   = 64  // defined: a 64-bit integer
	asSingle = -32 // defined: an IEEE single float
	asDouble = -64 // defined: an IEEE double float
)

// The sizes of the test files' data, in bytes.
const (
	tableBytes = 35 // defined: ten bytes by three rows, and a heap of five
	cubeBytes  = 8  // defined: a cube of two by two by two bytes
)

// halves is a row of two single floats, each a half.
func halves() []byte {
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, []float32{half, half})
	return data.Bytes()
}

// writeStored writes one stored integer at a BITPIX's width.
func writeStored(data *bytes.Buffer, bitpix int, value int64) {
	switch bitpix {
	case byteWide:
		data.WriteByte(byte(value))
	case asShort:
		binary.Write(data, binary.BigEndian, int16(value))
	case asInt:
		binary.Write(data, binary.BigEndian, int32(value))
	default:
		binary.Write(data, binary.BigEndian, value)
	}
}

// Every kind of pixel is read and scaled. Stored BLANKs are NaN. So are
// floats that are not a number.
func TestEveryKindOfPixelIsRead(t *testing.T) {
	for _, bitpix := range []int{byteWide, asShort, asInt, asLong,
		asSingle, asDouble} {
		values, err := image(t, bitpix).Row(row)
		if err != nil || len(values) != wide {
			t.Fatalf("BITPIX %d: %v, %v", bitpix, values, err)
		}
		want := float64(offset + scale*stored)
		if bitpix < none {
			want = half
		}
		if values[first] != want || !math.IsNaN(values[second]) {
			t.Errorf("BITPIX %d: %v", bitpix, values)
		}
	}
}

// Anti-tests: rows that are not there are errors. So is data cut short.
func TestARowThatIsNotThereIsAnError(t *testing.T) {
	read := image(t, asSingle)
	for _, y := range []int{-tall, tall} {
		if _, err := read.Row(y); err == nil {
			t.Errorf("row %d was read", y)
		}
	}
	cards := []string{card("SIMPLE", "T"), card("BITPIX", "-64"),
		card("NAXIS", "2"), card("NAXIS1", "4000"), card("NAXIS2", "1")}
	cut, err := fits.Open(fileOf(cards, []byte{zero}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cut.Row(row); !errors.Is(err, fits.Refused()) {
		t.Errorf("a cut row: %v", err)
	}
}

// Empty primaries and tables before the image are stepped over. Cubes of
// one plane are images.
func TestTheFirstImageIsFoundPastWhatIsNot(t *testing.T) {
	empty := hdu([]string{card("SIMPLE", "T"), card("BITPIX", "8"),
		card("NAXIS", "0"), card("EXTEND", "T")}, nil)
	table := hdu([]string{card("XTENSION", "'BINTABLE'"),
		card("BITPIX", "8"), card("NAXIS", "2"), card("NAXIS1", "10"),
		card("NAXIS2", "3"), card("PCOUNT", "5"), card("GCOUNT", "1")},
		make([]byte, tableBytes))
	plane := hdu([]string{card("XTENSION", "'IMAGE   '"),
		card("BITPIX", "-32"), card("NAXIS", "3"), card("NAXIS1", "2"),
		card("NAXIS2", "1"), card("NAXIS3", "1"), card("PCOUNT", "0"),
		card("GCOUNT", "1"), card("OBJECT", "'found'")},
		halves())
	file := append(append(empty, table...), plane...)
	read, err := openedWithin(t, file)
	if err != nil {
		t.Fatal(err)
	}
	object, _ := read.Header().Text("OBJECT")
	values, err := read.Row(row)
	if object != "found" || read.Width() != wide || read.Height() != tall ||
		err != nil || values[first] != half {
		t.Errorf("%q %dx%d %v %v", object, read.Width(), read.Height(),
			values, err)
	}
}

// Anti-tests: files with no two-dimensional image are refused. So are
// kinds of pixel the standard does not have and axes with no length.
func TestWhatIsNotAnImageIsRefused(t *testing.T) {
	emptyPrimary := []string{card("SIMPLE", "T"), card("BITPIX", "8"),
		card("NAXIS", "0")}
	for want, file := range map[string][]byte{
		"no two-dimensional image": hdu([]string{card("SIMPLE", "T"),
			card("BITPIX", "8"), card("NAXIS", "3"), card("NAXIS1", "2"),
			card("NAXIS2", "2"), card("NAXIS3", "2")},
			make([]byte, cubeBytes)),
		"BITPIX 12": hdu([]string{card("SIMPLE", "T"),
			card("BITPIX", "12"), card("NAXIS", "2"), card("NAXIS1", "1"),
			card("NAXIS2", "1")}, nil),
		"NAXIS2 is not": hdu([]string{card("SIMPLE", "T"),
			card("BITPIX", "8"), card("NAXIS", "2"),
			card("NAXIS1", "1")}, nil),
		"no END": append(hdu(emptyPrimary, nil),
			make([]byte, cardWidth)...),
		"without BITPIX": hdu([]string{card("SIMPLE", "T"),
			card("NAXIS", "0")}, nil),
		"BLANK is": hdu([]string{card("SIMPLE", "T"),
			card("BITPIX", "16"), card("NAXIS", "2"), card("NAXIS1", "1"),
			card("NAXIS2", "1"), card("BLANK", "'none'")}, nil),
	} {
		_, err := openedWithin(t, file)
		if !errors.Is(err, fits.Refused()) ||
			!strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
}

// The sizes of the HDUs the stepping test writes, in bytes. Each is
// defined by its cards: a table's heap and rows, or one byte alone.
const (
	heapBytes   = 2000 // defined: the first table's PCOUNT
	rowBytes    = 100  // defined: the first table's NAXIS1
	rowsOf      = 40   // defined: the first table's NAXIS2
	oneBlock    = 2880 // defined: the second table's one row, a block exactly
	oneByte     = 1    // defined: the third table, of one byte
	lastRow     = 2    // defined: the third row, counted from none
	rowsInImage = 3    // defined: the image's rows
)

// steppedFile is a file whose image comes after HDUs of every awkward
// size: one crossing blocks, one filling a block and one of a byte. Then
// comes an empty image. The image is two wide and three tall. Each
// row's values are its own number.
func steppedFile() []byte {
	primary := hdu([]string{card("SIMPLE", "T"), card("BITPIX", "8"),
		card("NAXIS", "0")}, nil)
	crossing := hdu([]string{card("XTENSION", "'BINTABLE'"),
		card("BITPIX", "8"), card("NAXIS", "2"), card("NAXIS1", "100"),
		card("NAXIS2", "40"), card("PCOUNT", "2000")},
		decoy(heapBytes+rowBytes*rowsOf))
	filling := hdu([]string{card("XTENSION", "'BINTABLE'"),
		card("BITPIX", "8"), card("NAXIS", "2"), card("NAXIS1", "2880"),
		card("NAXIS2", "1"), card("GCOUNT", "1")},
		decoy(oneBlock))
	single := hdu([]string{card("XTENSION", "'BINTABLE'"),
		card("BITPIX", "8"), card("NAXIS", "2"), card("NAXIS1", "1"),
		card("NAXIS2", "1"), card("PCOUNT", "0"), card("GCOUNT", "1")},
		decoy(oneByte))
	empty := hdu([]string{card("XTENSION", "'IMAGE'"),
		card("BITPIX", "-32"), card("NAXIS", "2"), card("NAXIS1", "2"),
		card("NAXIS2", "0")}, nil)
	var values bytes.Buffer
	for y := range rowsInImage {
		binary.Write(&values, binary.BigEndian, []float32{float32(y),
			float32(y)})
	}
	image := hdu([]string{card("XTENSION", "'IMAGE'"),
		card("BITPIX", "-32"), card("NAXIS", "2"), card("NAXIS1", "2"),
		card("NAXIS2", "3")}, values.Bytes())
	var file bytes.Buffer
	for _, each := range [][]byte{primary, crossing, filling, single, empty,
		image} {
		file.Write(each)
	}
	return file.Bytes()
}

// openedWithin is Open of a file, or a failed test when Open does not end
// within the picked patience.
func openedWithin(t *testing.T, file []byte) (*fits.Image, error) {
	t.Helper()
	patience := web(picked(t, "patience"))
	ended := make(chan opening, oneByte)
	go openInto(ended, file)
	select {
	case done := <-ended:
		return done.image, done.err
	case <-time.After(patience):
		t.Fatal("Open did not end")
		return nil, nil
	}
}

// opening is what Open gave.
type opening struct {
	image *fits.Image
	err   error
}

// openInto opens a file and sends what Open gave.
func openInto(ended chan<- opening, file []byte) {
	read, err := fits.Open(bytes.NewReader(file))
	ended <- opening{image: read, err: err}
}

// web is a count of seconds as a duration.
func web(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// HDUs of every awkward size are stepped over to the image after them.
// So is an empty image. The image's last row is its own.
func TestSizesOfEveryShapeAreSteppedOver(t *testing.T) {
	read, err := openedWithin(t, steppedFile())
	if err != nil {
		t.Fatal(err)
	}
	values, err := read.Row(lastRow)
	if err != nil || values[first] != lastRow || values[second] != lastRow {
		t.Errorf("%v, %v", values, err)
	}
}

// Float images may carry a BLANK, which the standard says to pass over.
// Integer images need carry none.
func TestBlankIsReadOnlyForIntegers(t *testing.T) {
	for _, cards := range [][]string{
		{card("SIMPLE", "T"), card("BITPIX", "-32"), card("NAXIS", "2"),
			card("NAXIS1", "1"), card("NAXIS2", "1"),
			card("BLANK", "'none'")},
		{card("SIMPLE", "T"), card("BITPIX", "16"), card("NAXIS", "2"),
			card("NAXIS1", "1"), card("NAXIS2", "1")},
	} {
		data := make([]byte, cardWidth)
		read, err := fits.Open(fileOf(cards, data))
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if _, err := read.Row(row); err != nil {
			t.Errorf("%v", err)
		}
	}
}

// decoy is data that reads as a header no image has: BITPIX 99 and an END
// card after card, cut to a size. Walks that land where they should never
// read it. Any walk that lands in it is refused.
func decoy(size int) []byte {
	cards := []string{card("BITPIX", "99"), card("NAXIS", "2"),
		card("NAXIS1", "1"), card("NAXIS2", "1"), "END"}
	var written bytes.Buffer
	for written.Len() < size {
		for _, each := range cards {
			written.WriteString(fmt.Sprintf("%-80s", each))
		}
	}
	return written.Bytes()[:size]
}
