package fits_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/fits"
)

// The values the table tests store and read back, each defined.
const (
	flagByte      = 'T'              // defined: a true logical's byte
	falseByte     = 'F'              // defined: a false logical's byte
	neitherByte   = 0                // defined: a logical that is neither
	byteValue     = 200              // defined: a byte past a signed one's reach
	zeroByte      = 0                // defined: a byte of nothing
	shortValue    = -3               // defined: a short
	shortEdge     = 5                // defined: the third row's short
	shortNull     = -1               // defined: the short column's TNULL
	intStored     = 7                // defined: the int the scaled column stores
	intNegative   = -1               // defined: the third row's stored int
	intScale      = 2                // defined: the scaled column's TSCAL
	intZero       = 10               // defined: the scaled column's TZERO
	longValue     = 1<<60 + 5        // defined: a long no float holds whole
	singleValue   = 1.5              // defined: a single
	singleEdge    = 2.5              // defined: the third row's single
	doubleValue   = -2.25            // defined: a double
	doubleEdge    = 1e300            // defined: a double past a single's reach
	nameValue     = "abc"            // defined: the text column's text
	nullName      = " \x00 "         // defined: text of spaces and NULs, null
	nameWithNUL   = "a\x00c"         // defined: text with a NUL inside it
	nameWidth     = 3                // defined: the text column's characters
	rowWidth      = 31               // defined: the bytes of a row of every kind
	firstRow      = 0                // defined: the first row
	secondRow     = 1                // defined: the second row
	thirdRow      = 2                // defined: the third row
	threeRows     = 3                // defined: the rows of the kinds table
	manyRows      = 10000            // defined: more rows than one read serves
	lastOfMany    = manyRows - 1     // defined: the last of them
	intBytes      = 4                // defined: the bytes of a row of one int
	shortBytes    = 2                // defined: the bytes of a row of one short
	singleBytes   = 4                // defined: the bytes of a row of one single
	twoRows       = 2                // defined: the rows of the small tables
	decoyWide     = 100              // defined: the decoy image's row of bytes
	decoyTall     = 40               // defined: the decoy image's rows
	shortHeader   = 100              // defined: a header cut before its END card
	noSuchRow     = -1               // defined: a row before the first
	someRoom      = 1e-6             // defined: a single's room, read back
	largestValue  = math.MaxFloat64  // defined: the largest double
	kinds         = 8                // defined: the columns of the kinds table
	singleCell    = 1                // defined: one value a cell
	noColumns     = 0                // defined: a table of no columns, refused
	fewerThanNone = -1               // defined: a count of columns below none
	bufferedRows  = 4096             // defined: the rows one read serves, table.go's
	lastBuffered  = bufferedRows - 1 // defined: the last row the first read serves
	oneRead       = 1                // defined: one read, for a buffer's rows
	twoReads      = 2                // defined: a second read, past the buffer
	noReads       = 0                // defined: no reads yet
	unsignedZero  = 1 << 15          // defined: the TZERO of an unsigned short
	floatNull     = 7                // defined: a float column's TNULL, ignored
)

// The files tableFile builds hold five blocks before the table's rows: a
// primary header, an image header, two blocks of decoy and the table's
// header. A cut file ends inside its rows or where they begin.
const (
	cutAt         = 6 * block // defined: the cut file ends a block into its rows
	cutBeforeRows = 5 * block // defined: the cut file ends where its rows begin
)

// kindsCards are the cards of a table with a column of every kind read:
// a logical, a byte, a short with a TNULL, a scaled int, a long, a
// single, a double and three characters of text, with a unit on one.
func kindsCards(rows int) []string {
	return []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", fmt.Sprint(rowWidth)),
		card("NAXIS2", fmt.Sprint(rows)), card("PCOUNT", "0"),
		card("GCOUNT", "1"), card("TFIELDS", fmt.Sprint(kinds)),
		card("TTYPE1", "'flag'"), card("TFORM1", "'L'"),
		card("TTYPE2", "'byte'"), card("TFORM2", "'B'"),
		card("TTYPE3", "'short'"), card("TFORM3", "'I'"),
		card("TNULL3", fmt.Sprint(shortNull)),
		card("TTYPE4", "'int'"), card("TFORM4", "'1J'"),
		card("TSCAL4", fmt.Sprint(intScale)), card("TZERO4", fmt.Sprint(intZero)),
		card("TTYPE5", "'long'"), card("TFORM5", "'K'"), card("TUNIT5", "'pc'"),
		card("TTYPE6", "'single'"), card("TFORM6", "'E'"),
		card("TTYPE7", "'double'"), card("TFORM7", "'D'"),
		card("TTYPE8", "'name'"), card("TFORM8", fmt.Sprintf("'%dA'", nameWidth))}
}

// kindsRows are three rows of the kinds table: the first with a value in
// every column, the second with a null in every column that can hold
// one, the third at each kind's edges.
func kindsRows() []byte {
	var data bytes.Buffer
	data.WriteByte(flagByte)
	data.WriteByte(byteValue)
	binary.Write(&data, binary.BigEndian, int16(shortValue))
	binary.Write(&data, binary.BigEndian, int32(intStored))
	binary.Write(&data, binary.BigEndian, int64(longValue))
	binary.Write(&data, binary.BigEndian, float32(singleValue))
	binary.Write(&data, binary.BigEndian, float64(doubleValue))
	data.WriteString(nameValue)
	data.WriteByte(neitherByte)
	data.WriteByte(byteValue)
	binary.Write(&data, binary.BigEndian, int16(shortNull))
	binary.Write(&data, binary.BigEndian, int32(intStored))
	binary.Write(&data, binary.BigEndian, int64(longValue))
	notASingle := float32(math.NaN())
	binary.Write(&data, binary.BigEndian, notASingle)
	binary.Write(&data, binary.BigEndian, math.NaN())
	data.WriteString(nullName)
	data.WriteByte(falseByte)
	data.WriteByte(zeroByte)
	binary.Write(&data, binary.BigEndian, int16(shortEdge))
	binary.Write(&data, binary.BigEndian, int32(intNegative))
	binary.Write(&data, binary.BigEndian, int64(-longValue))
	binary.Write(&data, binary.BigEndian, float32(singleEdge))
	binary.Write(&data, binary.BigEndian, float64(doubleEdge))
	data.WriteString(nameWithNUL)
	return data.Bytes()
}

// decoyImage is an image of decoy cards, for a walk to step over: a walk
// that mis-sizes it lands on a decoy, which is refused.
func decoyImage() []byte {
	wide, tall := fmt.Sprint(decoyWide), fmt.Sprint(decoyTall)
	cards := []string{card("XTENSION", "'IMAGE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", wide), card("NAXIS2", tall)}
	return hdu(cards, decoy(decoyWide*decoyTall))
}

// tableFile is a file whose table comes after a primary array of
// nothing and the decoy image, which are stepped over.
func tableFile(cards []string, rows []byte) []byte {
	primary := hdu([]string{card("SIMPLE", "T"), card("BITPIX", "8"),
		card("NAXIS", "0")}, nil)
	var file bytes.Buffer
	for _, each := range [][]byte{primary, decoyImage(), hdu(cards, rows)} {
		file.Write(each)
	}
	return file.Bytes()
}

// tabling is what OpenTable gave.
type tabling struct {
	table *fits.Table
	err   error
}

// tableInto opens a file's table and sends what OpenTable gave.
func tableInto(ended chan<- tabling, file io.ReaderAt) {
	table, err := fits.OpenTable(file)
	ended <- tabling{table: table, err: err}
}

// tabledWithin is OpenTable of a file, or a failed test when it does
// not end within the picked patience: a walk through the file's HDUs
// that a slip makes endless ends the test, not the run.
func tabledWithin(test *testing.T, file io.ReaderAt) (*fits.Table, error) {
	test.Helper()
	patience := web(picked(test, "patience"))
	ended := make(chan tabling, oneRead)
	go tableInto(ended, file)
	select {
	case done := <-ended:
		return done.table, done.err
	case <-time.After(patience):
		test.Fatal("OpenTable did not end")
		return nil, nil
	}
}

// tabled is the table in a file, or the test stops.
func tabled(test *testing.T, file []byte) *fits.Table {
	test.Helper()
	table, err := tabledWithin(test, bytes.NewReader(file))
	if err != nil {
		test.Fatal(err)
	}
	return table
}

// kindsTable is the kinds table of three rows, opened, or the test
// stops.
func kindsTable(test *testing.T) *fits.Table {
	test.Helper()
	file := tableFile(kindsCards(threeRows), kindsRows())
	return tabled(test, file)
}

// cell is a row and a column of a table, or the test stops.
func cell(test *testing.T, table *fits.Table, row int64,
	name string) (fits.Row, fits.Column) {
	test.Helper()
	read, err := table.Row(row)
	if err != nil {
		test.Fatal(err)
	}
	column, err := table.Column(name)
	if err != nil {
		test.Fatal(err)
	}
	return *read, *column
}

// every kind of cell is read: a logical, a byte, a short, a scaled int,
// a long too big for a float, a single, a double and text; a column's
// name and unit are the header's; the table is found past the primary
// array and an image.
func TestEveryKindOfCellIsRead(test *testing.T) {
	table := kindsTable(test)
	columns := table.Columns()
	if table.Rows() != threeRows || len(columns) != kinds ||
		!table.Header().Has("TFIELDS") {
		test.Fatalf("the table has %d rows and %d columns", table.Rows(),
			len(columns))
	}
	row, flag := cell(test, table, firstRow, "flag")
	if set, held, err := row.Logical(flag); !set || !held || err != nil {
		test.Errorf("the flag is %v, %v, %v", set, held, err)
	}
	_, long := cell(test, table, firstRow, "long")
	if long.Name() != "long" || long.Unit() != "pc" {
		test.Errorf("the long column is %q in %q", long.Name(), long.Unit())
	}
	if whole, held, err := row.Int(long); whole != longValue || !held ||
		err != nil {
		test.Errorf("the long is %d, %v, %v", whole, held, err)
	}
	for name, want := range map[string]float64{"byte": byteValue,
		"short": shortValue, "int": intScale*intStored + intZero,
		"double": doubleValue} {
		_, column := cell(test, table, firstRow, name)
		if got, err := row.Float(column); got != want || err != nil {
			test.Errorf("the %s is %g, %v, want %g", name, got, err, want)
		}
	}
	_, single := cell(test, table, firstRow, "single")
	if got, err := row.Float(single); !(math.Abs(got-singleValue) <=
		someRoom) || err != nil {
		test.Errorf("the single is %g, %v", got, err)
	}
	_, name := cell(test, table, firstRow, "name")
	if text, err := row.Text(name); text != nameValue || err != nil {
		test.Errorf("the name is %q, %v", text, err)
	}
	if _, column := cell(test, table, firstRow, "int"); column.Unit() != "" {
		test.Errorf("the int column has a unit %q", column.Unit())
	}
}

// the anti-test: a null is read as none: a logical that is neither, a
// short at its TNULL, singles and doubles that are not numbers, and
// text of spaces and NULs; a byte and a long, with no TNULL, are never
// null.
func TestANullIsReadAsNone(test *testing.T) {
	table := kindsTable(test)
	row, flag := cell(test, table, secondRow, "flag")
	if set, held, err := row.Logical(flag); set || held || err != nil {
		test.Errorf("a null flag is %v, %v, %v", set, held, err)
	}
	_, short := cell(test, table, secondRow, "short")
	if whole, held, err := row.Int(short); held || err != nil {
		test.Errorf("a null short is %d, %v, %v", whole, held, err)
	}
	for _, name := range []string{"short", "single", "double"} {
		_, column := cell(test, table, secondRow, name)
		if got, err := row.Float(column); !math.IsNaN(got) || err != nil {
			test.Errorf("a null %s is %g, %v", name, got, err)
		}
	}
	for _, name := range []string{"byte", "long"} {
		_, column := cell(test, table, secondRow, name)
		if _, held, err := row.Int(column); !held || err != nil {
			test.Errorf("a %s with no TNULL is null, %v", name, err)
		}
	}
	_, name := cell(test, table, secondRow, "name")
	if text, err := row.Text(name); text != "" || err != nil {
		test.Errorf("null text is %q, %v", text, err)
	}
}

// the test and the anti-test: each kind at its edges: a false logical
// is held, not null; a byte of nothing is nothing; a negative stored
// int is scaled; a negative long is read whole; a double past a
// single's reach is whole; a NUL inside text stays.
func TestEachKindIsReadAtItsEdges(test *testing.T) {
	table := kindsTable(test)
	row, flag := cell(test, table, thirdRow, "flag")
	if set, held, err := row.Logical(flag); set || !held || err != nil {
		test.Errorf("a false flag is %v, %v, %v", set, held, err)
	}
	_, long := cell(test, table, thirdRow, "long")
	if whole, held, err := row.Int(long); whole != -longValue || !held ||
		err != nil {
		test.Errorf("the negative long is %d, %v, %v", whole, held, err)
	}
	for name, want := range map[string]float64{"byte": zeroByte,
		"short": shortEdge, "int": intScale*intNegative + intZero,
		"single": singleEdge, "double": doubleEdge} {
		_, column := cell(test, table, thirdRow, name)
		if got, err := row.Float(column); got != want || err != nil {
			test.Errorf("the %s is %g, %v, want %g", name, got, err, want)
		}
	}
	_, name := cell(test, table, thirdRow, "name")
	if text, err := row.Text(name); text != nameWithNUL || err != nil {
		test.Errorf("the name with a NUL inside is %q, %v", text, err)
	}
}

// the anti-test: a cell is read only as what its column holds: no
// number from text or a logical, no integer from a float, no logical
// or text from a number.
func TestACellIsReadOnlyAsWhatItHolds(test *testing.T) {
	table := kindsTable(test)
	row, name := cell(test, table, firstRow, "name")
	_, flag := cell(test, table, firstRow, "flag")
	_, double := cell(test, table, firstRow, "double")
	if _, err := row.Float(name); err == nil {
		test.Error("text read as a number")
	}
	if _, err := row.Float(flag); err == nil {
		test.Error("a logical read as a number")
	}
	if _, _, err := row.Int(double); err == nil {
		test.Error("a double read as an integer")
	}
	if _, _, err := row.Logical(double); err == nil {
		test.Error("a double read as a logical")
	}
	if _, err := row.Text(double); err == nil {
		test.Error("a double read as text")
	}
}

// oneColumn are the cards of a table of one column of a form, with any
// more cards given, so many rows.
func oneColumn(form string, rows int, width int, more ...string) []string {
	cards := []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", fmt.Sprint(width)),
		card("NAXIS2", fmt.Sprint(rows)), card("TFIELDS", "1"),
		card("TTYPE1", "'value'"), card("TFORM1", form)}
	return append(cards, more...)
}

// the unsigned convention: a short whose TZERO is 2^15 reads back
// through Float as an unsigned short, the least stored as nothing and
// the most as 65535.
func TestTheUnsignedConventionIsReadThroughFloat(test *testing.T) {
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, int16(math.MinInt16))
	binary.Write(&data, binary.BigEndian, int16(math.MaxInt16))
	zero := card("TZERO1", fmt.Sprint(unsignedZero))
	file := hdu(oneColumn("'I'", twoRows, shortBytes, zero), data.Bytes())
	table := tabled(test, file)
	for index, want := range map[int64]float64{firstRow: none,
		secondRow: math.MaxUint16} {
		row, column := cell(test, table, index, "value")
		if got, err := row.Float(column); got != want || err != nil {
			test.Errorf("row %d is %g, %v, want %g", index, got, err, want)
		}
	}
}

// a TNULL on a float column is ignored: the value it names reads as
// itself, since a float's null is not a number whatever the card says.
func TestATNULLOnAFloatColumnIsIgnored(test *testing.T) {
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, float32(floatNull))
	null := card("TNULL1", fmt.Sprint(floatNull))
	file := hdu(oneColumn("'E'", singleCell, singleBytes, null), data.Bytes())
	table := tabled(test, file)
	row, column := cell(test, table, firstRow, "value")
	if got, err := row.Float(column); got != floatNull || err != nil {
		test.Errorf("the value TNULL names is %g, %v", got, err)
	}
}

// intRows is so many rows of one int, each its own row number.
func intRows(rows int) []byte {
	var data bytes.Buffer
	for row := range rows {
		binary.Write(&data, binary.BigEndian, int32(row))
	}
	return data.Bytes()
}

// intCards are the cards of a table of one int column, so many rows.
func intCards(rows int) []string {
	return []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", fmt.Sprint(intBytes)),
		card("NAXIS2", fmt.Sprint(rows)), card("TFIELDS", "1"),
		card("TTYPE1", "'row'"), card("TFORM1", "'J'")}
}

// intFile is a file whose table is one int column of so many rows.
func intFile(rows int) []byte {
	return tableFile(intCards(rows), intRows(rows))
}

// rows are read in any order across more of them than one read of the
// file serves: forward through every row, then back to the first.
func TestRowsAreReadAcrossTheBuffer(test *testing.T) {
	table := tabled(test, intFile(manyRows))
	_, column := cell(test, table, firstRow, "row")
	for _, index := range []int64{firstRow, lastOfMany, firstRow} {
		row, err := table.Row(index)
		if err != nil {
			test.Fatal(err)
		}
		if got, held, err := row.Int(column); got != index || !held ||
			err != nil {
			test.Fatalf("row %d holds %d, %v, %v", index, got, held, err)
		}
	}
	for index := range int64(manyRows) {
		row, _ := table.Row(index)
		if got, _, _ := row.Int(column); got != index {
			test.Fatalf("row %d holds %d", index, got)
		}
	}
}

// counting is a file that counts its reads.
type counting struct {
	file  *bytes.Reader
	reads int
}

// ReadAt reads from the file and counts the read.
func (file *counting) ReadAt(into []byte, at int64) (int, error) {
	file.reads++
	return file.file.ReadAt(into, at)
}

// the anti-test: a row is read from the file once: one read serves 4096
// rows from the row asked for, so that row asked again, the next, and
// the last of the 4096 read nothing more; the row past them is a second
// read.
func TestOneReadServesTheBufferedRows(test *testing.T) {
	file := &counting{file: bytes.NewReader(intFile(manyRows))}
	table, err := tabledWithin(test, file)
	if err != nil {
		test.Fatal(err)
	}
	_, column := cell(test, table, firstRow, "row")
	file.reads = noReads
	for _, index := range []int64{firstRow, firstRow, secondRow, lastBuffered} {
		row, _ := table.Row(index)
		if got, _, err := row.Int(column); got != index || err != nil {
			test.Fatalf("row %d holds %d, %v", index, got, err)
		}
	}
	if file.reads != oneRead {
		test.Errorf("four cells of the first %d rows took %d reads of the "+
			"file", bufferedRows, file.reads)
	}
	row, _ := table.Row(bufferedRows)
	if got, _, err := row.Int(column); got != bufferedRows || err != nil {
		test.Fatalf("row %d holds %d, %v", bufferedRows, got, err)
	}
	if file.reads != twoReads {
		test.Errorf("the row past the first %d took %d reads of the file",
			bufferedRows, file.reads)
	}
}

// the anti-test: a row the table does not have is refused, before the
// first and past the last, and so is a column the table lacks.
func TestARowOrAColumnTheTableLacksIsRefused(test *testing.T) {
	table := kindsTable(test)
	for _, index := range []int64{noSuchRow, threeRows} {
		if _, err := table.Row(index); !errors.Is(err, fits.Refused()) {
			test.Errorf("row %d of %d gave %v", index, threeRows, err)
		}
	}
	if _, err := table.Column("nothing"); !errors.Is(err, fits.Refused()) {
		test.Errorf("a column the table lacks gave %v", err)
	}
}

// brokenTable is a table of one int column with some of its cards
// replaced or left out: what the name says is wrong with it.
func brokenTable(why string, cards []string) []byte {
	whole := []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", fmt.Sprint(intBytes)),
		card("NAXIS2", "1")}
	if why != "a table without TFIELDS" {
		whole = append(whole, card("TFIELDS", "1"))
	}
	if why != "a column without a name" && why != "a table without TFIELDS" {
		whole = append(whole, card("TTYPE1", "'row'"))
	}
	if why != "a column without a form" && why != "a table without TFIELDS" {
		whole = append(whole, card("TFORM1", "'J'"))
	}
	return hdu(append(whole, cards...), intRows(singleCell))
}

// refusal is a broken table or file: the cards that break it, and the
// words its refusal must say.
type refusal struct {
	cards []string
	said  string
}

// brokenForms are the tables whose column's form is wrong: a kind not
// read, a form that is not one, a count of values below one, or
// no form at all, each with the words its refusal says.
func brokenForms() map[string]refusal {
	return map[string]refusal{
		"a form not read":             {[]string{card("TFORM1", "'X'")}, `"X" is not read`},
		"complex numbers":             {[]string{card("TFORM1", "'C'")}, `"C" is not read`},
		"a kind not read":             {[]string{card("TFORM1", "'F'")}, `"F" is not read`},
		"a form that is not a form":   {[]string{card("TFORM1", "'ZZ'")}, `"ZZ" is not read`},
		"a form of nothing":           {[]string{card("TFORM1", "''")}, `"" is not read`},
		"a form of letters":           {[]string{card("TFORM1", "'abE'")}, `"abE" is not read`},
		"a column of no values":       {[]string{card("TFORM1", "'0D'")}, "0 values"},
		"a column of fewer than none": {[]string{card("TFORM1", "'-5A'")}, "-5 values"},
		"a column without a form":     {[]string{card("TTYPE1", "'row'")}, "no form"},
		"a column without a name":     {[]string{card("TFORM1", "'E'")}, "no name"},
		"columns short of the row":    {[]string{card("NAXIS1", "5")}, "rows of 5"},
	}
}

// brokenCards are the tables whose other cards are wrong: the axes, the
// count of columns, or a number that is not one, each with the words
// its refusal says.
func brokenCards() map[string]refusal {
	return map[string]refusal{
		"a table of no columns": {[]string{card("TFIELDS", fmt.Sprint(noColumns)),
			card("NAXIS1", "0")}, "0 columns"},
		"a count of columns below none": {[]string{card("TFIELDS",
			fmt.Sprint(fewerThanNone))}, "-1 columns"},
		"a table of three axes":        {[]string{card("NAXIS", "3")}, "not 2"},
		"an axis that is not a length": {[]string{card("NAXIS2", "'x'")}, "not a length"},
		"a table without TFIELDS":      {[]string{card("TTYPE1", "'row'")}, "TFIELDS"},
		"a TNULL that is not a number": {[]string{card("TNULL1", "'x'")}, "not a whole number"},
		"a TNULL with a fraction":      {[]string{card("TNULL1", "1.5")}, "not a whole number"},
		"a TSCAL that is not a number": {[]string{card("TSCAL1", "'x'")}, "TSCAL1 is"},
		"a TZERO that is not a number": {[]string{card("TZERO1", "'x'")}, "TZERO1 is"},
	}
}

// the anti-test: a table this package does not read is refused, saying
// why: each of the broken forms and the broken cards.
func TestATableNotReadIsRefused(test *testing.T) {
	for _, tables := range []map[string]refusal{brokenForms(), brokenCards()} {
		for why, broken := range tables {
			file := brokenTable(why, broken.cards)
			_, err := tabledWithin(test, bytes.NewReader(file))
			if !errors.Is(err, fits.Refused()) ||
				!strings.Contains(err.Error(), broken.said) {
				test.Errorf("%s: %v, which does not say %q", why, err, broken.said)
			}
		}
	}
}

// the anti-test: a file with no table is refused, and so is one whose
// HDUs before a table cannot be stepped over: without BITPIX or NAXIS,
// with an axis that is not a length, of a BITPIX not read, or with a
// header cut before its END card; each refusal says which.
func TestAFileWithNoTableIsRefused(test *testing.T) {
	files := map[string]refusal{
		"an image alone": {[]string{card("SIMPLE", "T"), card("BITPIX", "-32"),
			card("NAXIS", "2"), card("NAXIS1", "2"), card("NAXIS2", "1")},
			"no binary table"},
		"an HDU without BITPIX": {[]string{card("SIMPLE", "T"),
			card("NAXIS", "0")}, "without BITPIX"},
		"an HDU without NAXIS": {[]string{card("SIMPLE", "T"),
			card("BITPIX", "8")}, "without NAXIS"},
		"an axis that is not a length": {[]string{card("SIMPLE", "T"),
			card("BITPIX", "8"), card("NAXIS", "1"), card("NAXIS1", "'x'")},
			"not a length"},
		"a BITPIX not read": {[]string{card("SIMPLE", "T"),
			card("BITPIX", "99"), card("NAXIS", "0")}, "BITPIX 99"},
	}
	for why, broken := range files {
		file := hdu(broken.cards, decoy(decoyWide))
		_, err := tabledWithin(test, bytes.NewReader(file))
		if !errors.Is(err, fits.Refused()) ||
			!strings.Contains(err.Error(), broken.said) {
			test.Errorf("%s: %v, which does not say %q", why, err, broken.said)
		}
	}
	short := hdu([]string{card("SIMPLE", "T"), card("BITPIX", "8"),
		card("NAXIS", "0")}, nil)[:shortHeader]
	_, err := tabledWithin(test, bytes.NewReader(short))
	if !errors.Is(err, fits.Refused()) || !strings.Contains(err.Error(),
		"END") {
		test.Errorf("a header cut before its END gave %v", err)
	}
}

// cut is a file cut short: its bytes end inside the table's rows, so a
// row past the cut cannot be read.
type cut struct{ bytes []byte }

// ReadAt reads what is there and fails past the cut.
func (file cut) ReadAt(into []byte, at int64) (int, error) {
	if at+int64(len(into)) > int64(len(file.bytes)) {
		return none, io.ErrUnexpectedEOF
	}
	return copy(into, file.bytes[at:]), nil
}

// the anti-test: a row the file cannot give is refused, not a number:
// through a file that fails past its cut, and through one that gives
// what it has with EOF, as bytes.Reader does.
func TestARowTheFileCannotGiveIsRefused(test *testing.T) {
	whole := intFile(manyRows)
	for _, file := range []io.ReaderAt{cut{whole[:cutAt]},
		bytes.NewReader(whole[:cutAt])} {
		table, err := tabledWithin(test, file)
		if err != nil {
			test.Fatal(err)
		}
		row, column := cell(test, table, lastOfMany, "row")
		if _, _, err := row.Int(column); !errors.Is(err, fits.Refused()) {
			test.Errorf("a row past the cut gave %v", err)
		}
		if _, err := row.Float(column); !errors.Is(err, fits.Refused()) {
			test.Errorf("a row past the cut gave %v as a number", err)
		}
	}
	flagged := tableFile(kindsCards(threeRows), kindsRows())
	flags, err := tabledWithin(test, cut{flagged[:cutBeforeRows]})
	if err != nil {
		test.Fatal(err)
	}
	row, flag := cell(test, flags, firstRow, "flag")
	if _, _, err := row.Logical(flag); !errors.Is(err, fits.Refused()) {
		test.Errorf("a flag past the cut gave %v", err)
	}
	_, name := cell(test, flags, firstRow, "name")
	if _, err := row.Text(name); !errors.Is(err, fits.Refused()) {
		test.Errorf("text past the cut gave %v", err)
	}
}

// ending is a file that says io.EOF with the last bytes it has, as a
// ReaderAt may, instead of after them.
type ending struct{ bytes []byte }

// ReadAt reads what is there, saying EOF with the last byte.
func (file ending) ReadAt(into []byte, at int64) (int, error) {
	if at >= int64(len(file.bytes)) {
		return none, io.EOF
	}
	read := copy(into, file.bytes[at:])
	if at+int64(read) == int64(len(file.bytes)) {
		return read, io.EOF
	}
	if read < len(into) {
		return read, io.ErrUnexpectedEOF
	}
	return read, nil
}

// a file that says it ends with its last row's bytes, as a ReaderAt
// may, gives the last row whole.
func TestAFileEndingWithItsLastRowGivesIt(test *testing.T) {
	rows := intRows(manyRows)
	exact := hdu(intCards(manyRows), rows)[:block+len(rows)]
	table, err := tabledWithin(test, ending{exact})
	if err != nil {
		test.Fatal(err)
	}
	row, column := cell(test, table, lastOfMany, "row")
	if got, held, err := row.Int(column); got != lastOfMany || !held ||
		err != nil {
		test.Errorf("the last row is %d, %v, %v", got, held, err)
	}
}

// the property: a number stored in a cell is read back as it was
// stored, a double exactly and a single within rounding, whatever the
// number.
func TestANumberStoredIsReadBack(test *testing.T) {
	rapid.Check(test, storedIsReadBack)
}

// storedIsReadBack is the property on one drawn double and single.
func storedIsReadBack(draw *rapid.T) {
	double := rapid.Float64Range(-largestValue, largestValue).Draw(draw,
		"double")
	single := rapid.Float32().Draw(draw, "single")
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, double)
	binary.Write(&data, binary.BigEndian, single)
	cards := []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", "12"), card("NAXIS2", "1"),
		card("TFIELDS", "2"), card("TTYPE1", "'double'"), card("TFORM1", "'D'"),
		card("TTYPE2", "'single'"), card("TFORM2", "'E'")}
	file := hdu(cards, data.Bytes())
	table, err := fits.OpenTable(bytes.NewReader(file))
	if err != nil {
		draw.Fatal(err)
	}
	row, err := table.Row(firstRow)
	if err != nil {
		draw.Fatal(err)
	}
	doubles, _ := table.Column("double")
	singles, _ := table.Column("single")
	if got, err := row.Float(*doubles); got != double || err != nil {
		draw.Errorf("a double %g is read as %g, %v", double, got, err)
	}
	if got, err := row.Float(*singles); float32(got) != single || err != nil {
		draw.Errorf("a single %g is read as %g, %v", single, got, err)
	}
}

// a TSCAL of none is read as no scaling, not as a scale that zeroes
// every value: astrogo/fitsio writes TSCALn and TZEROn of none on
// every column it writes, meaning neither was set.
func TestATSCALOfNoneIsNoScaling(test *testing.T) {
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, float32(floatNull))
	unset := []string{card("TSCAL1", "0.00000"), card("TZERO1", "0.00000")}
	file := hdu(oneColumn("'E'", singleCell, singleBytes, unset...),
		data.Bytes())
	table := tabled(test, file)
	row, column := cell(test, table, firstRow, "value")
	if got, err := row.Float(column); got != floatNull || err != nil {
		test.Errorf("the value under a TSCAL of none is %g, %v", got, err)
	}
}

// each column's kind is its form's letter.
func TestEachColumnsKindIsItsFormsLetter(test *testing.T) {
	table := kindsTable(test)
	for name, kind := range map[string]byte{"flag": 'L', "byte": 'B',
		"short": 'I', "int": 'J', "long": 'K', "single": 'E', "double": 'D',
		"name": 'A'} {
		_, column := cell(test, table, firstRow, name)
		if column.Kind() != kind {
			test.Errorf("%s is of kind %q, want %q", name, column.Kind(), kind)
		}
	}
}

// The many table: a long, three singles, two ints, two logicals, a
// double and three characters of text a row, each defined.
const (
	manySingles  = 3  // defined: the singles a row of the many table
	manyInts     = 2  // defined: the ints a row of the many table
	manyLogicals = 2  // defined: the logicals a row of the many table
	manyColumns  = 6  // defined: the many table's columns
	manyWidth    = 41 // defined: 8 + 3*4 + 2*4 + 2*1 + 8 + 3 bytes a row
)

// manyCards are the cards of a table with columns of many values a row
// between columns of one: a long, three singles, two ints, two
// logicals, a double and text.
func manyCards(rows int) []string {
	return []string{card("XTENSION", "'BINTABLE'"), card("BITPIX", "8"),
		card("NAXIS", "2"), card("NAXIS1", fmt.Sprint(manyWidth)),
		card("NAXIS2", fmt.Sprint(rows)), card("PCOUNT", "0"),
		card("GCOUNT", "1"), card("TFIELDS", fmt.Sprint(manyColumns)),
		card("TTYPE1", "'long'"), card("TFORM1", "'K'"),
		card("TTYPE2", "'singles'"),
		card("TFORM2", fmt.Sprintf("'%dE'", manySingles)),
		card("TTYPE3", "'ints'"), card("TFORM3", fmt.Sprintf("'%dJ'", manyInts)),
		card("TTYPE4", "'flags'"),
		card("TFORM4", fmt.Sprintf("'%dL'", manyLogicals)),
		card("TTYPE5", "'double'"), card("TFORM5", "'D'"),
		card("TTYPE6", "'name'"), card("TFORM6", fmt.Sprintf("'%dA'", nameWidth))}
}

// manyRow writes a row of the many table: the long and the double
// given, the columns of many between them filled, and the text.
func manyRow(data *bytes.Buffer, long int64, double float64) {
	binary.Write(data, binary.BigEndian, long)
	for range manySingles {
		binary.Write(data, binary.BigEndian, float32(singleValue))
	}
	for range manyInts {
		binary.Write(data, binary.BigEndian, int32(intStored))
	}
	for range manyLogicals {
		data.WriteByte(flagByte)
	}
	binary.Write(data, binary.BigEndian, double)
	data.WriteString(nameValue)
}

// manyTable is the many table of two rows, the second's long and double
// unlike the first's, opened, or the test stops.
func manyTable(test *testing.T) *fits.Table {
	test.Helper()
	var rows bytes.Buffer
	manyRow(&rows, longValue, doubleValue)
	manyRow(&rows, -longValue, doubleEdge)
	file := tableFile(manyCards(twoRows), rows.Bytes())
	return tabled(test, file)
}

// the test: a table with columns of many values a row opens; each
// column counts its values, text as one; and the columns of one after
// the columns of many are read where they lie, in either row.
func TestColumnsOfManyAreSteppedOver(test *testing.T) {
	table := manyTable(test)
	counts := map[string]int{"long": singleCell, "singles": manySingles,
		"ints": manyInts, "flags": manyLogicals, "double": singleCell,
		"name": singleCell}
	for name, want := range counts {
		_, column := cell(test, table, firstRow, name)
		if column.Count() != want {
			test.Errorf("%s counts %d values, not %d", name, column.Count(),
				want)
		}
	}
	first, long := cell(test, table, firstRow, "long")
	second, double := cell(test, table, secondRow, "double")
	_, name := cell(test, table, secondRow, "name")
	if got, _, err := first.Int(long); err != nil || got != longValue {
		test.Errorf("the first long read %d, %v", got, err)
	}
	if got, _, err := second.Int(long); err != nil || got != -longValue {
		test.Errorf("the second long read %d, %v", got, err)
	}
	if got, err := first.Float(double); err != nil || got != doubleValue {
		test.Errorf("the first double read %g, %v", got, err)
	}
	if got, err := second.Float(double); err != nil || got != doubleEdge {
		test.Errorf("the second double read %g, %v", got, err)
	}
	if got, err := second.Text(name); err != nil || got != nameValue {
		test.Errorf("the second text read %q, %v", got, err)
	}
}

// the anti-test: a column of many values a row is not read as one, as
// a number, an integer or a logical; each is refused, saying how many
// values the column holds.
func TestAColumnOfManyIsNotReadAsOne(test *testing.T) {
	table := manyTable(test)
	row, singles := cell(test, table, firstRow, "singles")
	_, ints := cell(test, table, firstRow, "ints")
	_, flags := cell(test, table, firstRow, "flags")
	_, err := row.Float(singles)
	refusedSaying(test, err, "singles holds 3 values a row")
	_, err = row.Float(ints)
	refusedSaying(test, err, "ints holds 2 values a row")
	_, _, err = row.Int(ints)
	refusedSaying(test, err, "ints holds 2 values a row")
	_, _, err = row.Logical(flags)
	refusedSaying(test, err, "flags holds 2 values a row")
}

// refusedSaying fails the test unless err is a refusal whose words hold
// those said.
func refusedSaying(test *testing.T, err error, said string) {
	test.Helper()
	if !errors.Is(err, fits.Refused()) || !strings.Contains(err.Error(), said) {
		test.Errorf("refused with %v, not saying %q", err, said)
	}
}
