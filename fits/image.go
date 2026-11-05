package fits

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// The kinds of pixel, by BITPIX. Each is defined by the FITS Standard
// 4.0, table 8.
const (
	unsigned8 = 8   // defined: an unsigned byte
	signed16  = 16  // defined: a 16-bit integer
	signed32  = 32  // defined: a 32-bit integer
	signed64  = 64  // defined: a 64-bit integer
	single    = -32 // defined: an IEEE single float
	double    = -64 // defined: an IEEE double float
)

// The counts an HDU's size is reckoned with.
const (
	plane      = 2 // defined: an image has two axes that are not one
	firstAxis  = 1 // defined: FITS counts axes from one
	flat       = 1 // defined: an axis of one holds nothing more
	byteBits   = 8 // defined: the bits in a byte
	noScale    = 1 // defined by section 4.4.2.5: BSCALE when absent
	noZero     = 0 // defined by section 4.4.2.5: BZERO when absent
	oneGroup   = 1 // defined by section 7.1.1: GCOUNT when absent
	noHeap     = 0 // defined by section 7.1.1: PCOUNT when absent
	firstRow   = 0 // defined: rows are counted here from none
	noPixels   = 0 // defined: an image of no pixels
	xAxis      = 0 // defined: the place of the first axis, along a row
	yAxis      = 1 // defined: the place of the second axis, up the rows
	firstByte  = 0 // defined: the place of a pixel's only byte
	shortOfOne = 1 // defined: what rounds a size up to whole blocks
)

// Image is a two-dimensional image in a FITS file. Its rows are read as
// they are asked for.
type Image struct {
	file          io.ReaderAt
	start         int64
	width, height int
	bitpix        int
	scale, zero   float64
	blank         int64
	hasBlank      bool
	header        *Header
}

// Open is the first two-dimensional image in a file: the primary array or
// an IMAGE extension. HDUs that hold no image are stepped over. Tables
// and empty primaries are such. Files with no image are refused.
func Open(file io.ReaderAt) (*Image, error) {
	var at int64
	for {
		section := io.NewSectionReader(file, at, math.MaxInt64-at)
		header, err := ReadHeader(section)
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: no two-dimensional image "+
				"in the file", errRefused)
		}
		if err != nil {
			return nil, err
		}
		at += headerSize(section)
		image, size, err := imageOf(file, at, header)
		if err != nil || image != nil {
			return image, err
		}
		at += padded(size)
	}
}

// headerSize is how far a header read has gone through its section.
func headerSize(section *io.SectionReader) int64 {
	read, _ := section.Seek(noBytes, io.SeekCurrent)
	return read
}

// padded is a size made up to whole blocks.
func padded(size int64) int64 {
	whole := (size + block - shortOfOne) / block
	return whole * block
}

// imageOf is the image an HDU holds, if it holds one. It gives the size
// of its data either way.
func imageOf(file io.ReaderAt, at int64, header *Header) (*Image, int64,
	error) {
	bitpix, bitpixErr := header.Number("BITPIX")
	axes, axesErr := header.Number("NAXIS")
	if bitpixErr != nil || axesErr != nil {
		return nil, noPixels, fmt.Errorf("%w: an HDU without "+
			"BITPIX and NAXIS", errRefused)
	}
	lengths, err := axisLengths(header, int(axes))
	if err != nil {
		return nil, noPixels, err
	}
	width, err := widthOf(int(bitpix))
	if err != nil {
		return nil, noPixels, err
	}
	size := dataSize(header, lengths, width)
	if !isImage(header, lengths) {
		return nil, size, nil
	}
	image := &Image{file: file, start: at, width: lengths[xAxis],
		height: lengths[yAxis], bitpix: int(bitpix),
		scale: numberOr(header, "BSCALE", noScale),
		zero:  numberOr(header, "BZERO", noZero), header: header}
	if header.Has("BLANK") && bitpix > noPixels {
		blank, err := header.Number("BLANK")
		if err != nil {
			return nil, noPixels, err
		}
		image.blank, image.hasBlank = int64(blank), true
	}
	return image, size, nil
}

// axisLengths are an HDU's NAXISn, in order.
func axisLengths(header *Header, axes int) ([]int, error) {
	lengths := make([]int, axes)
	for i := range lengths {
		key := fmt.Sprintf("NAXIS%d", i+firstAxis)
		length, err := header.Number(key)
		if err != nil || length < noPixels {
			return nil, fmt.Errorf("%w: %s is not a length",
				errRefused, key)
		}
		lengths[i] = int(length)
	}
	return lengths, nil
}

// widthOf is the bytes in one pixel of a BITPIX.
func widthOf(bitpix int) (int, error) {
	switch bitpix {
	case unsigned8, signed16, signed32, signed64, single, double:
		return abs(bitpix) / byteBits, nil
	}
	return noPixels, fmt.Errorf("%w: BITPIX %d", errRefused, bitpix)
}

// abs is a whole number's size, without its sign.
func abs(number int) int { return max(number, -number) }

// dataSize is the bytes an HDU's data takes before its padding, as section
// 7.1.1 reckons it: PCOUNT more than GCOUNT times the axes' product.
func dataSize(header *Header, lengths []int, width int) int64 {
	if len(lengths) == noPixels {
		return noPixels
	}
	product := int64(width)
	for _, length := range lengths {
		product *= int64(length)
	}
	if header.Has("XTENSION") {
		groups := numberOr(header, "GCOUNT", oneGroup)
		heap := numberOr(header, "PCOUNT", noHeap)
		return int64(heap) + int64(groups)*product
	}
	return product
}

// isImage is whether an HDU holds a two-dimensional image: the primary
// array or an IMAGE extension, its first two axes not empty and any more
// axes of one.
func isImage(header *Header, lengths []int) bool {
	if header.Has("XTENSION") {
		kind, err := header.Text("XTENSION")
		if err != nil || kind != "IMAGE" {
			return false
		}
	}
	if len(lengths) < plane || lengths[xAxis] == noPixels ||
		lengths[yAxis] == noPixels {
		return false
	}
	for _, length := range lengths[plane:] {
		if length != flat {
			return false
		}
	}
	return true
}

// numberOr is a keyword's number, or otherwise when the header has none.
func numberOr(header *Header, key string, otherwise float64) float64 {
	number, err := header.Number(key)
	if err != nil {
		return otherwise
	}
	return number
}

// Width is the image's pixels a row.
func (image *Image) Width() int { return image.width }

// Height is the image's rows.
func (image *Image) Height() int { return image.height }

// Header is the image's header.
func (image *Image) Header() *Header { return image.header }

// Row is the values of one row. The first row is the bottom one, as the
// standard counts them. Each is BZERO more than BSCALE times the stored
// value. Stored BLANKs are NaN, and so are floats that are not a number.
func (image *Image) Row(y int) ([]float64, error) {
	if y < firstRow || y >= image.height {
		return nil, fmt.Errorf("there is no row %d of %d", y,
			image.height)
	}
	width, _ := widthOf(image.bitpix)
	raw := make([]byte, image.width*width)
	at := image.start + int64(y)*int64(len(raw))
	if _, err := image.file.ReadAt(raw, at); err != nil {
		return nil, fmt.Errorf("%w: row %d cut short: %v", errRefused,
			y,
			err)
	}
	values := make([]float64, image.width)
	for x := range values {
		values[x] = image.pixel(raw[x*width:])
	}
	return values, nil
}

// pixel is the value of the pixel at the front of its bytes.
func (image *Image) pixel(raw []byte) float64 {
	var stored int64
	switch image.bitpix {
	case single:
		bits := binary.BigEndian.Uint32(raw)
		narrow := math.Float32frombits(bits)
		return image.scaled(float64(narrow))
	case double:
		bits := binary.BigEndian.Uint64(raw)
		return image.scaled(math.Float64frombits(bits))
	case unsigned8:
		stored = int64(raw[firstByte])
	case signed16:
		bits := binary.BigEndian.Uint16(raw)
		stored = int64(int16(bits))
	case signed32:
		bits := binary.BigEndian.Uint32(raw)
		stored = int64(int32(bits))
	default:
		stored = int64(binary.BigEndian.Uint64(raw))
	}
	if image.hasBlank && stored == image.blank {
		return math.NaN()
	}
	return image.scaled(float64(stored))
}

// scaled is a stored value as its physical one.
func (image *Image) scaled(stored float64) float64 {
	return image.zero + image.scale*stored
}
