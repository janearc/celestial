// Package picture builds small FITS files for grissom's tests: one image
// of 64-bit floats, laid on the sky by the cards given.
package picture

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// The sizes and kinds of a FITS file. Each is defined by the FITS
// Standard 4.0.
const (
	cardWide = 80   // defined: a header card's characters
	block    = 2880 // defined: a block's bytes
	doubles  = -64  // defined: the BITPIX of 64-bit floats
	axes     = 2    // defined: an image's axes
	first    = 1    // defined: the first pixel, as FITS counts them
	none     = 0    // defined: the reference longitude and latitude
	degree   = 1    // defined: a degree a pixel, on the plain grids
)

// Block is a FITS block's size in bytes. It's defined by the standard.
const Block = block

// card is one header card.
func card(key string, value any) string {
	line := fmt.Sprintf("%-8s= %20v", key, value)
	return fmt.Sprintf("%-*s", cardWide, line)
}

// padding is the bytes that bring a size up to whole blocks.
func padding(size int) int { return (block - size%block) % block }

// File is a FITS file holding one image of 64-bit floats, laid on the sky
// by the cards given. Its values go row by row from the bottom.
func File(width, height int, placed []string, values []float64) []byte {
	cards := []string{card("SIMPLE", "T"), card("BITPIX", doubles),
		card("NAXIS", axes), card("NAXIS1", width),
		card("NAXIS2", height)}
	cards = append(cards, placed...)
	ending := fmt.Sprintf("%-*s", cardWide, "END")
	header := strings.Join(append(cards, ending), "")
	short := padding(len(header))
	file := []byte(header + strings.Repeat(" ", short))
	for _, value := range values {
		bits := math.Float64bits(value)
		file = binary.BigEndian.AppendUint64(file, bits)
	}
	rest := padding(len(file))
	return append(file, make([]byte, rest)...)
}

// Tangent is the cards that lay an image about a centre in ICRS by the
// gnomonic projection. Its first pixel is at the centre, and each pixel
// spans scale degrees.
func Tangent(ra, dec, scale float64) []string {
	return []string{card("CTYPE1", "'RA---TAN'"),
		card("CTYPE2", "'DEC--TAN'"), card("CRVAL1", ra),
		card("CRVAL2", dec), card("CRPIX1", first),
		card("CRPIX2", first),
		card("CDELT1", -scale), card("CDELT2", scale)}
}

// Plate is the cards that lay an image on a plain grid in a frame's axes,
// a degree a pixel. Its first pixel is at longitude none and a latitude.
// The axes are quoted CTYPE values such as 'GLON-CAR'.
func Plate(lonAxis, latAxis string, lat float64) []string {
	return []string{card("CTYPE1", lonAxis), card("CTYPE2", latAxis),
		card("CRVAL1", none), card("CRVAL2", none),
		card("CRPIX1", first),
		card("CRPIX2", first-lat), card("CDELT1", degree),
		card("CDELT2", degree)}
}
