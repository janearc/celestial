package fits

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// The binary table's numbers, each defined by the FITS Standard 4.0,
// section 7.3, as recalled, or here.
const (
	tableAxes   = 2    // defined: a table has two axes
	rowAxis     = 0    // defined: the first axis is a row's bytes, NAXIS1
	rowsAxis    = 1    // defined: the second is the rows, NAXIS2
	oneValue    = 1    // defined: one value a column, the count when absent
	noValues    = 0    // defined: a column of no values
	oneLetter   = 1    // defined: a form has one letter
	firstColumn = 1    // defined: TTYPEn counts columns from one
	logicalTrue = 'T'  // defined: the byte of a true logical value
	logicalNull = 0    // defined: the byte of a logical that is neither
	rowsAtOnce  = 4096 // defined: the rows one read serves, half a megabyte
	noScaling   = 0    // defined: a TSCAL of none, which no writer means
)

// The widths of the kinds of column, in bytes, each defined by section
// 7.3.1's table, as recalled.
const (
	logicalWidth = 1 // defined: L, a byte
	byteWidth    = 1 // defined: B, a byte
	shortWidth   = 2 // defined: I, two bytes
	intWidth     = 4 // defined: J, four bytes
	longWidth    = 8 // defined: K, eight bytes
	singleWidth  = 4 // defined: E, four bytes
	doubleWidth  = 8 // defined: D, eight bytes
)

// Column is one column of a binary table: its name and unit as the
// header gives them, the kind of value it holds, how many a row, and
// where it lies in a row.
type Column struct {
	name, unit  string
	kind        byte
	count       int
	at, width   int
	scale, zero float64
	null        int64
	hasNull     bool
}

// Name is the column's name, its TTYPE.
func (column Column) Name() string { return column.name }

// Unit is the column's unit as the header writes it, its TUNIT, or
// nothing when the header gives none.
func (column Column) Unit() string { return column.unit }

// Kind is the letter of the column's TFORM: what each value is and
// how many bytes it takes, L, B, I, J, K, E, D or A.
func (column Column) Kind() byte { return column.kind }

// Count is how many values a row holds in the column: one, or more in a
// column of many, which is stepped over and not read. Text is one value
// however many characters it holds.
func (column Column) Count() int { return column.count }

// Table is a binary table extension in a FITS file: columns of numbers
// and text in rows of one width, read as they are asked for, a few
// thousand rows at a time.
type Table struct {
	file    io.ReaderAt
	start   int64
	width   int
	rows    int64
	columns []Column
	header  *Header
	buffer  []byte
	from    int64
}

// OpenTable is the first binary table in a file. HDUs that hold no table
// are stepped over: the primary array, images, and anything else the
// standard sizes. Files with no table are refused, and so is a table
// whose columns this package does not read.
func OpenTable(file io.ReaderAt) (*Table, error) {
	var at int64
	for {
		section := io.NewSectionReader(file, at, math.MaxInt64)
		header, err := ReadHeader(section)
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: no binary table in the file",
				errRefused)
		}
		if err != nil {
			return nil, err
		}
		at += headerSize(section)
		if kind, err := header.Text("XTENSION"); err == nil &&
			kind == "BINTABLE" {
			return tableOf(file, at, header)
		}
		size, err := sizeOf(header)
		if err != nil {
			return nil, err
		}
		at += padded(size)
	}
}

// sizeOf is the bytes an HDU's data takes, from its cards.
func sizeOf(header *Header) (int64, error) {
	bitpix, err := header.Number("BITPIX")
	if err != nil {
		return noPixels, fmt.Errorf("%w: an HDU without BITPIX", errRefused)
	}
	axes, err := header.Number("NAXIS")
	if err != nil {
		return noPixels, fmt.Errorf("%w: an HDU without NAXIS", errRefused)
	}
	lengths, err := axisLengths(header, int(axes))
	if err != nil {
		return noPixels, err
	}
	width, err := widthOf(int(bitpix))
	if err != nil {
		return noPixels, err
	}
	return dataSize(header, lengths, width), nil
}

// tableOf is the table an HDU holds, from its cards: the row's width
// and the rows from the axes, and each column from its TTYPE, TFORM and
// the rest.
func tableOf(file io.ReaderAt, at int64, header *Header) (*Table, error) {
	axes, err := header.Number("NAXIS")
	if err != nil || int(axes) != tableAxes {
		return nil, fmt.Errorf("%w: a binary table with NAXIS %g, not 2",
			errRefused, axes)
	}
	lengths, err := axisLengths(header, tableAxes)
	if err != nil {
		return nil, err
	}
	fields, err := header.Number("TFIELDS")
	if err != nil {
		return nil, fmt.Errorf("%w: a binary table without TFIELDS",
			errRefused)
	}
	if fields < firstColumn {
		return nil, fmt.Errorf("%w: a binary table of %g columns", errRefused,
			fields)
	}
	table := &Table{file: file, start: at, width: lengths[rowAxis],
		rows: int64(lengths[rowsAxis]), header: header}
	offset := noBytes
	for number := firstColumn; number < int(fields)+firstColumn; number++ {
		column, err := columnOf(header, number, offset)
		if err != nil {
			return nil, err
		}
		table.columns = append(table.columns, *column)
		offset += column.width
	}
	if offset != table.width {
		return nil, fmt.Errorf("%w: columns of %d bytes in rows of %d",
			errRefused, offset, table.width)
	}
	return table, nil
}

// columnOf is a table's column from its cards, beginning so many bytes
// into a row. It refuses a column without a name or a form, a form this
// package does not read, and a column of none or of many values.
func columnOf(header *Header, number int, at int) (*Column, error) {
	name, err := header.Text(fmt.Sprintf("TTYPE%d", number))
	if err != nil {
		return nil, fmt.Errorf("%w: column %d has no name", errRefused,
			number)
	}
	form, err := header.Text(fmt.Sprintf("TFORM%d", number))
	if err != nil {
		return nil, fmt.Errorf("%w: column %s has no form", errRefused, name)
	}
	kind, count, width, err := formOf(form)
	if err != nil {
		return nil, fmt.Errorf("%w: column %s: %w", errRefused, name, err)
	}
	column := &Column{name: name, kind: kind, count: count, at: at,
		width: width, scale: noScale, zero: noZero}
	unit := fmt.Sprintf("TUNIT%d", number)
	if header.Has(unit) {
		column.unit, _ = header.Text(unit)
	}
	if err := column.scaled(header, number); err != nil {
		return nil, err
	}
	return column, column.nulled(header, number)
}

// scaled takes the column's TSCAL and TZERO when the header has them.
// One that is not a number is refused, since it changes every value.
func (column *Column) scaled(header *Header, number int) error {
	scale, zero := fmt.Sprintf("TSCAL%d", number), fmt.Sprintf("TZERO%d",
		number)
	var err error
	if header.Has(scale) {
		if column.scale, err = header.Number(scale); err != nil {
			return err
		}
		// astrogo/fitsio writes a TSCAL of none on every column: none
		// was set; no writer means a scale that zeroes every value
		if column.scale == noScaling {
			column.scale = noScale
		}
	}
	if header.Has(zero) {
		if column.zero, err = header.Number(zero); err != nil {
			return err
		}
	}
	return nil
}

// nulled takes an integer column's TNULL when the header has one, read
// whole. A float column's is ignored: archives write one on every
// column, and a float's null is not a number whatever the card says.
func (column *Column) nulled(header *Header, number int) error {
	null := fmt.Sprintf("TNULL%d", number)
	if !header.Has(null) || !holdsIntegers(column.kind) {
		return nil
	}
	value, err := header.Integer(null)
	if err != nil {
		return err
	}
	column.null, column.hasNull = value, true
	return nil
}

// holdsIntegers is whether a kind of column holds whole numbers.
func holdsIntegers(kind byte) bool {
	switch kind {
	case 'B', 'I', 'J', 'K':
		return true
	}
	return false
}

// formOf is a column's kind, its count of values and its bytes in a
// row, from its TFORM: a repeat count, one when absent, and a letter.
// Text is one value at any count, a byte a character; a number is as
// many values as its count, each its kind's width.
func formOf(form string) (byte, int, int, error) {
	digits := strings.TrimRight(form, "ABDEIJKL")
	letter := strings.TrimPrefix(form, digits)
	if len(letter) != oneLetter {
		return logicalNull, noValues, noValues,
			fmt.Errorf("a form %q is not read", form)
	}
	count := oneValue
	if digits != "" {
		repeat, err := strconv.Atoi(digits)
		if err != nil {
			return logicalNull, noValues, noValues,
				fmt.Errorf("a form %q is not read", form)
		}
		count = repeat
	}
	kind := letter[noBytes]
	if count < oneValue {
		return kind, noValues, noValues,
			fmt.Errorf("a column of %d values, %q", count, form)
	}
	if kind == 'A' {
		return kind, oneValue, count, nil
	}
	return kind, count, count * widthOfKind(kind), nil
}

// ofOne refuses a column of many values a row, which is not read.
func ofOne(column Column) error {
	if column.count != oneValue {
		return fmt.Errorf("%w: column %s holds %d values a row, and one is "+
			"read", errRefused, column.name, column.count)
	}
	return nil
}

// widthOfKind is the bytes one value of a kind takes.
func widthOfKind(kind byte) int {
	switch kind {
	case 'L':
		return logicalWidth
	case 'B':
		return byteWidth
	case 'I':
		return shortWidth
	case 'J':
		return intWidth
	case 'K':
		return longWidth
	case 'E':
		return singleWidth
	}
	return doubleWidth
}

// Rows is how many rows the table has.
func (table *Table) Rows() int64 { return table.rows }

// Columns is the table's columns, in the file's order.
func (table *Table) Columns() []Column { return table.columns }

// Column is the column of a name. It refuses a name the table lacks.
func (table *Table) Column(name string) (*Column, error) {
	for _, column := range table.columns {
		if column.name == name {
			return &column, nil
		}
	}
	return nil, fmt.Errorf("%w: no column %s", errRefused, name)
}

// Header is the table's header.
func (table *Table) Header() *Header { return table.header }

// Row is one row of a table, counted from none. It is read from the
// file when a cell of it is asked for. It refuses a row the table does
// not have.
type Row struct {
	table *Table
	index int64
}

// Row is a row, counted from none. It refuses a row past the end.
func (table *Table) Row(index int64) (*Row, error) {
	if index < firstRow || index >= table.rows {
		return nil, fmt.Errorf("%w: row %d of %d", errRefused, index,
			table.rows)
	}
	return &Row{table, index}, nil
}

// cell is a column's bytes in a row, read from the file through the
// table's buffer, which is filled from the row on when the row is not
// in it. A file that says it ends with the last bytes asked for, as a
// ReaderAt may, has given them; one that ends before them is refused.
func (table *Table) cell(index int64, column Column) ([]byte, error) {
	rows := int64(len(table.buffer)) / int64(table.width)
	if index < table.from || index >= table.from+rows {
		count := min(int64(rowsAtOnce), table.rows-index)
		table.buffer = make([]byte, count*int64(table.width))
		at := table.start + index*int64(table.width)
		read, err := table.file.ReadAt(table.buffer, at)
		if err != nil && !(errors.Is(err, io.EOF) && read == len(table.buffer)) {
			table.buffer = nil
			return nil, fmt.Errorf("%w: row %d: %w", errRefused, index, err)
		}
		table.from = index
	}
	begin := (index-table.from)*int64(table.width) + int64(column.at)
	return table.buffer[begin : begin+int64(column.width)], nil
}

// Float is a cell's value as a number, scaled by the column's TSCAL
// and TZERO. A float that is not a number, and an integer equal to the
// column's TNULL, read as not a number. It refuses a column of text or
// of logical values, and a column of many values a row.
func (row Row) Float(column Column) (float64, error) {
	if err := ofOne(column); err != nil {
		return math.NaN(), err
	}
	raw, err := row.table.cell(row.index, column)
	if err != nil {
		return math.NaN(), err
	}
	var stored float64
	switch column.kind {
	case 'E':
		single := math.Float32frombits(binary.BigEndian.Uint32(raw))
		stored = float64(single)
	case 'D':
		stored = math.Float64frombits(binary.BigEndian.Uint64(raw))
	case 'B', 'I', 'J', 'K':
		whole, null := integerOf(raw, column)
		if null {
			return math.NaN(), nil
		}
		stored = float64(whole)
	default:
		return math.NaN(), fmt.Errorf("%w: column %s holds no numbers",
			errRefused, column.name)
	}
	return column.scale*stored + column.zero, nil
}

// integerOf is an integer cell's value, and whether it is the column's
// TNULL.
func integerOf(raw []byte, column Column) (int64, bool) {
	var whole int64
	switch column.kind {
	case 'B':
		whole = int64(raw[firstByte])
	case 'I':
		short := int16(binary.BigEndian.Uint16(raw))
		whole = int64(short)
	case 'J':
		whole32 := int32(binary.BigEndian.Uint32(raw))
		whole = int64(whole32)
	default:
		whole = int64(binary.BigEndian.Uint64(raw))
	}
	return whole, column.hasNull && whole == column.null
}

// Int is a cell's value as a whole number, unscaled, and whether it is
// one: false for the column's TNULL. It refuses a column that holds no
// integers, and a column of many values a row.
func (row Row) Int(column Column) (int64, bool, error) {
	if !holdsIntegers(column.kind) {
		return noValues, false, fmt.Errorf("%w: column %s holds no integers",
			errRefused, column.name)
	}
	if err := ofOne(column); err != nil {
		return noValues, false, err
	}
	raw, err := row.table.cell(row.index, column)
	if err != nil {
		return noValues, false, err
	}
	whole, null := integerOf(raw, column)
	return whole, !null, nil
}

// Logical is a cell's logical value, and whether it is one: a null
// logical is neither. It refuses a column that holds no logical values,
// and a column of many values a row.
func (row Row) Logical(column Column) (bool, bool, error) {
	if column.kind != 'L' {
		return false, false, fmt.Errorf("%w: column %s holds no logical"+
			" values", errRefused, column.name)
	}
	if err := ofOne(column); err != nil {
		return false, false, err
	}
	raw, err := row.table.cell(row.index, column)
	if err != nil {
		return false, false, err
	}
	value := raw[firstByte]
	return value == logicalTrue, value != logicalNull, nil
}

// Text is a cell's text, with the spaces after it dropped. It refuses a
// column that holds no text.
func (row Row) Text(column Column) (string, error) {
	if column.kind != 'A' {
		return "", fmt.Errorf("%w: column %s holds no text", errRefused,
			column.name)
	}
	raw, err := row.table.cell(row.index, column)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), " \x00"), nil
}
