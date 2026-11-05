package picture_test

import (
	"bytes"
	"testing"

	"github.com/janearc/celestial/fits"
	"github.com/janearc/celestial/grissom/internal/picture"
)

// The values and places the tests build with.
const (
	wide  = 2   // defined: the test image's pixels a row
	tall  = 1   // defined: its rows
	low   = 3   // defined: one pixel's value
	high  = 8   // defined: another's
	east  = 10  // defined: a right ascension, in degrees
	north = 0   // defined: the equator, where the projection is exact
	scale = 1   // defined: a degree a pixel
	pole  = 90  // defined: the north pole's latitude
	at    = 1.0 // defined: the first pixel, as FITS counts them
	row   = 0   // defined: the bottom row
	next  = 2.0 // defined: the second pixel, as FITS counts them
)

// A file opens as an image of its size and values, in whole blocks, and
// lies where its cards lay it.
func TestAFileIsTheImageItSays(t *testing.T) {
	for _, placed := range [][]string{picture.Tangent(east, north, scale),
		picture.Plate("'GLON-CAR'", "'GLAT-CAR'", pole)} {
		file := picture.File(wide, tall, placed, []float64{low, high})
		if len(file)%picture.Block != row {
			t.Errorf("%d bytes", len(file))
		}
		image, err := fits.Open(bytes.NewReader(file))
		if err != nil {
			t.Fatal(err)
		}
		values, err := image.Row(row)
		if err != nil || image.Width() != wide || image.Height() != tall ||
			values[row] != low || values[tall] != high {
			t.Errorf("%dx%d of %v, %v", image.Width(), image.Height(),
				values, err)
		}
		checkPlace(t, image, placed)
	}
}

// checkPlace checks that an image's first pixel lies where its cards say.
// On the sky in ICRS, right ascension runs east to the left, so the second
// pixel lies west of the first.
func checkPlace(t *testing.T, image *fits.Image, placed []string) {
	t.Helper()
	where, err := image.Sky()
	if err != nil {
		t.Fatal(err)
	}
	lon, lat := where.Place(at, at)
	want := []float64{east, north}
	if where.Frame() == fits.Galactic {
		want = []float64{row, pole}
	} else if west, _ := where.Place(next, at); !(west < east) {
		t.Errorf("the second pixel at %v, not west of %v", west, east)
	}
	if lon != want[row] || lat != want[tall] {
		t.Errorf("%v: the first pixel at %v, %v", placed, lon, lat)
	}
}
