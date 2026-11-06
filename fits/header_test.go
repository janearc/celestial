package fits_test

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/celestial/fits"
	"github.com/janearc/puzzlemouth/numbers"
)

// The FITS Standard's units, as the tests write files with them.
const (
	block     = 2880 // defined: FITS 4.0 section 3.1, a block's bytes
	cardWidth = 80   // defined: FITS 4.0 section 4.1, a card's width
)

// card is one header card: a keyword and its value, as the standard
// writes them.
func card(key, value string) string {
	return fmt.Sprintf("%-8s= %s", key, value)
}

// hdu is one header and data unit: its cards and an END card, then its
// data. Each is padded to whole blocks. The header pads with spaces and
// the data with zeros, as the standard says.
func hdu(cards []string, data []byte) []byte {
	var written bytes.Buffer
	for _, each := range append(cards, "END") {
		written.WriteString(fmt.Sprintf("%-80s", each))
	}
	pad(&written, ' ')
	if len(data) > none {
		start := written.Len()
		written.Write(data)
		pad(&written, zero)
		if written.Len() == start {
			written.WriteByte(zero)
		}
	}
	return written.Bytes()
}

// pad fills a buffer to whole blocks with a byte.
func pad(written *bytes.Buffer, with byte) {
	for written.Len()%block != none {
		written.WriteByte(with)
	}
}

// The counts the tests write files with.
const (
	none = 0 // defined: no bytes, no rows
	zero = 0 // defined: the byte the standard pads data with
)

// headerOf is the header read from cards.
func headerOf(t *testing.T, cards ...string) *fits.Header {
	t.Helper()
	header, err := fits.ReadHeader(fileOf(cards, nil))
	if err != nil {
		t.Fatal(err)
	}
	return header
}

// Strings lose their quotes and their trailing spaces, and keep a doubled
// quote as one. Comments go, even one with a slash in a string before it.
func TestTextIsReadAsTheStandardWritesIt(t *testing.T) {
	header := headerOf(t, card("OBJECT", "'O''HARA  '         / who"),
		card("ORIGIN", "'a/b'   / where"), card("EMPTY", "''"))
	for key, want := range map[string]string{"OBJECT": "O'HARA",
		"ORIGIN": "a/b", "EMPTY": ""} {
		if got, err := header.Text(key); err != nil || got != want {
			t.Errorf("%s: %q, %v", key, got, err)
		}
	}
}

// Numbers are read with their comments off, and a D exponent as an E.
func TestNumbersAreReadAsTheStandardWritesThem(t *testing.T) {
	header := headerOf(t, card("CDELT1", "-2.5D-03 / degrees"),
		card("NAXIS", "2"))
	if got, err := header.Number("CDELT1"); err != nil || got != written {
		t.Errorf("%v, %v", got, err)
	}
	if !header.Has("NAXIS") || header.Has("NAXIS3") {
		t.Error("Has says what is not there")
	}
}

// The whole numbers the header tests read, each defined.
const (
	largestWhole = 1<<63 - 1 // defined: the largest int64, which a float rounds
	minusOne     = -1        // defined: a whole number below none
)

// the test and the anti-test: whole numbers are read whole, one too
// large for a float exactly; a fraction, text and a missing card are
// refused by name.
func TestWholeNumbersAreReadWhole(t *testing.T) {
	header := headerOf(t, card("TNULL1", "9223372036854775807"),
		card("TNULL2", "-1"), card("TNULL3", "1.5"), card("TNULL4", "'x'"))
	if got, err := header.Integer("TNULL1"); err != nil ||
		got != largestWhole {
		t.Errorf("%v, %v", got, err)
	}
	if got, err := header.Integer("TNULL2"); err != nil || got != minusOne {
		t.Errorf("%v, %v", got, err)
	}
	for key, said := range map[string]string{"TNULL3": "TNULL3 is",
		"TNULL4": "TNULL4 is", "TNULL5": "no TNULL5 card"} {
		if _, err := header.Integer(key); !errors.Is(err, fits.Refused()) ||
			!strings.Contains(err.Error(), said) {
			t.Errorf("%s: %v", key, err)
		}
	}
}

// Anti-tests: missing cards are refused by name. So are numbers asked as
// text and text asked as a number.
func TestAskingForWhatIsNotThereIsRefused(t *testing.T) {
	header := headerOf(t, card("NAXIS", "2"), card("CTYPE1", "'RA---TAN'"))
	_, missing := header.Number("NAXIS3")
	_, notText := header.Text("NAXIS")
	_, notNumber := header.Number("CTYPE1")
	_, noText := header.Text("CTYPE9")
	for want, err := range map[string]error{"NAXIS3": missing,
		"NAXIS is not": notText, "CTYPE1 is": notNumber,
		"CTYPE9 is not": noText} {
		if !errors.Is(err, fits.Refused()) ||
			!strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
}

// Anti-test: a header cut short before its END card is refused.
func TestAHeaderWithNoEndIsRefused(t *testing.T) {
	whole := hdu([]string{card("NAXIS", "0")}, nil)
	_, err := fits.ReadHeader(bytes.NewReader(whole[:block-cardWidth]))
	if !errors.Is(err, fits.Refused()) ||
		!strings.Contains(err.Error(), "no END") {
		t.Errorf("%v", err)
	}
}

// picked is a picked number from the tests' numbers file.
func picked(t *testing.T, name string) float64 {
	t.Helper()
	raw, err := os.ReadFile("testdata/numbers.yaml")
	if err != nil {
		t.Fatal(err)
	}
	held, err := numbers.Read("numbers.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	number, err := held.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// near is whether two numbers agree to within a tolerance. NaN agrees
// with nothing.
func near(got, want, within float64) bool {
	return math.Abs(got-want) <= within
}

// fileOf is one HDU of cards and data, as a file to read.
func fileOf(cards []string, data []byte) *bytes.Reader {
	whole := hdu(cards, data)
	return bytes.NewReader(whole)
}

// written is the number the CDELT1 card writes with a D. It's defined.
const written = -2.5e-3
