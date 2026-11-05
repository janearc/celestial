package layer_test

import (
	"bytes"
	"errors"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/fits"
	"github.com/janearc/celestial/grissom/internal/picture"
	"github.com/janearc/celestial/grissom/layer"
	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/puzzlemouth/numbers"
	"pgregory.net/rapid"
)

// The counts and angles the tests work with.
const (
	none     = 0   // defined: no pixels, no depth, either infinity
	first    = 1   // defined: the first pixel, as FITS counts them
	second   = 2   // defined: the second pixel
	degree   = 1   // defined: a degree a pixel, on the plain grids
	pole     = 90  // defined: the latitude of a pole, in degrees
	fullTurn = 360 // defined: a turn, in degrees
	side     = 8   // defined: the most pixels a drawn image is on a side
	deepest  = 12  // defined: the deepest the properties draw, for speed
	spread   = 1e6 // defined: the most a drawn pixel is, either way
	low      = 3   // defined: one pixel's value
	high     = 8   // defined: another's
	halves   = 2.0 // defined: a mean of two is their sum halved
	// pinDepth is the depth places are checked at. It's defined: cells
	// there are about a third of a degree across, so a pixel a degree
	// off lands elsewhere.
	pinDepth = 8
)

// The north galactic pole in ICRS, in degrees. Each is defined by the
// Hipparcos definition, ESA SP-1200 volume 1 section 1.5.3.
const (
	poleAscension   = 192.85948 // defined: its right ascension
	poleDeclination = 27.12825  // defined: its declination
)

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

// stopper is what the helpers stop a test or a draw with.
type stopper interface {
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// open is the image in a file.
func open(test stopper, file []byte) *fits.Image {
	image, err := fits.Open(bytes.NewReader(file))
	if err != nil {
		test.Fatal(err)
	}
	return image
}

// lay is a file's image laid at a depth, from a source of its own.
func lay(test stopper, file []byte, depth int) *layer.Layer {
	laid, err := layer.Lay("test", open(test, file), depth, layer.Source{})
	if err != nil {
		test.Fatal(err)
	}
	return laid
}

// locate is the cell a place falls in at a depth.
func locate(test stopper, lon, lat float64, depth int) *mesh.Cell {
	place, err := mesh.NewPoint(lon, lat)
	if err != nil {
		test.Fatal(err)
	}
	cell, err := mesh.Locate(place, depth)
	if err != nil {
		test.Fatal(err)
	}
	return cell
}

// The numbers the properties are drawn with. Each test that runs one
// fills them.
var scale, rounding float64

// drawn is an image the properties draw: its size, where it lies and its
// pixels, some of them blank.
type drawn struct {
	width, height int
	ra, dec       float64
	values        []float64
}

// draw draws an image. Every pixel is even when even is.
func draw(t *rapid.T, even bool) drawn {
	image := drawn{width: rapid.IntRange(first, side).Draw(t, "width"),
		height: rapid.IntRange(first, side).Draw(t, "height"),
		ra:     rapid.Float64Range(none, fullTurn).Draw(t, "ra"),
		dec:    rapid.Float64Range(-pole, pole).Draw(t, "dec")}
	image.values = make([]float64, image.width*image.height)
	level := rapid.Float64Range(-spread, spread).Draw(t, "level")
	for i := range image.values {
		image.values[i] = level
		if !even {
			image.values[i] = rapid.Float64Range(-spread,
				spread).Draw(t, "value")
		}
		if rapid.Bool().Draw(t, "blank") {
			image.values[i] = math.NaN()
		}
	}
	return image
}

// laidOut is a drawn image laid at a drawn depth. It is nil when every
// pixel is blank, and then the refusal is checked.
func laidOut(t *rapid.T, image drawn) *layer.Layer {
	depth := rapid.IntRange(none, deepest).Draw(t, "depth")
	file := picture.File(image.width, image.height,
		picture.Tangent(image.ra, image.dec, scale), image.values)
	laid, err := layer.Lay("drawn", open(t, file), depth, layer.Source{})
	if !slices.ContainsFunc(image.values, isMeasured) {
		if !errors.Is(err, layer.Refused()) {
			t.Fatalf("all blank, and %v", err)
		}
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return laid
}

// isMeasured is whether a pixel holds a value.
func isMeasured(value float64) bool { return !math.IsNaN(value) }

// keepsEverything is the property of TestEveryPixelIsCountedOnce, for one
// draw.
func keepsEverything(t *rapid.T) {
	image := draw(t, false)
	laid := laidOut(t, image)
	if laid == nil {
		return
	}
	var pixels, counted int
	var sum, size, kept float64
	for _, value := range image.values {
		if isMeasured(value) {
			pixels++
			sum += value
			size += math.Abs(value)
		}
	}
	keys := laid.Cells()
	for _, key := range keys {
		value := heldAt(t, laid, key)
		counted += value.Pixels()
		kept += value.Mean() * float64(value.Pixels())
	}
	if counted != pixels || !(math.Abs(kept-sum) <= rounding*size) ||
		!slices.IsSorted(keys) {
		t.Fatalf("%d pixels summing %v kept as %d summing %v in %v",
			pixels, sum, counted, kept, keys)
	}
}

// heldAt is the value a layer holds at a key, which names a cell at its
// depth.
func heldAt(test stopper, laid *layer.Layer, key uint64) layer.Value {
	cell, err := mesh.FromKey(key)
	if err != nil || cell.Depth() != laid.Depth() {
		test.Fatalf("%#x at depth %d: %v", key, laid.Depth(), err)
	}
	value, held := laid.At(*cell)
	if !held || value.Pixels() <= none {
		test.Fatalf("%#x holds %v, %v", key, value, held)
	}
	return value
}

// Property: every pixel that holds a value is counted in one cell, and
// the cells' means, each times its count, sum to the pixels' sum.
func TestEveryPixelIsCountedOnce(t *testing.T) {
	scale = picked(t, "pixel-scale")
	rounding = picked(t, "sum-rounding")
	rapid.Check(t, keepsEverything)
}

// staysEven is the property of TestAnEvenImageIsEvenEverywhere, for one
// draw.
func staysEven(t *rapid.T) {
	image := draw(t, true)
	laid := laidOut(t, image)
	if laid == nil {
		return
	}
	level := image.values[slices.IndexFunc(image.values, isMeasured)]
	for _, key := range laid.Cells() {
		value := heldAt(t, laid, key)
		if !(math.Abs(value.Mean()-level) <= rounding*math.Abs(level)) {
			t.Fatalf("%#x holds %v of an even %v", key, value.Mean(), level)
		}
	}
}

// Property: an image the same everywhere is the same in every cell.
func TestAnEvenImageIsEvenEverywhere(t *testing.T) {
	scale = picked(t, "pixel-scale")
	rounding = picked(t, "sum-rounding")
	rapid.Check(t, staysEven)
}

// Two pixels in one cell make their mean there. The layer keeps its name,
// its depth and its source, and holds nothing at another cell or depth.
func TestTwoPixelsMakeTheirMean(t *testing.T) {
	ra, dec := picked(t, "centre-ra"), picked(t, "centre-dec")
	placed := picture.Tangent(ra, dec, picked(t, "pixel-scale"))
	file := picture.File(second, first, placed, []float64{low, high})
	size := int64(len(file))
	source := layer.NewSource("https://example.org/image.fits",
		"as the publisher says", time.Unix(none, none), size,
		"hex")
	laid, err := layer.Lay("orion", open(t, file), none, source)
	if err != nil {
		t.Fatal(err)
	}
	cell := locate(t, ra, dec, none)
	value, held := laid.At(*cell)
	if !held || value.Mean() != (low+high)/halves ||
		value.Pixels() != second {
		t.Errorf("%v, %v", value, held)
	}
	if laid.Name() != "orion" || laid.Depth() != none ||
		laid.Source() != source ||
		!slices.Equal(laid.Cells(), []uint64{cell.Key()}) {
		t.Errorf("%s at %d from %v holds %v", laid.Name(), laid.Depth(),
			laid.Source(), laid.Cells())
	}
	elsewhere := locate(t, ra+fullTurn/halves, -dec, none)
	deeper := locate(t, ra, dec, first)
	for _, other := range []*mesh.Cell{elsewhere, deeper} {
		if _, held := laid.At(*other); held {
			t.Errorf("%#x is held", other.Key())
		}
	}
}

// A source gives back each part of its ledger entry.
func TestASourceGivesBackItsEntry(t *testing.T) {
	fetched := time.Unix(none, none)
	source := layer.NewSource("https://example.org/image.fits",
		"as the publisher says", fetched, high, "hex")
	if source.URL() != "https://example.org/image.fits" ||
		source.Licence() != "as the publisher says" ||
		!source.Fetched().Equal(fetched) || source.Size() != high ||
		source.Sum() != "hex" {
		t.Errorf("%v", source)
	}
}

// Each pixel falls in the cell its centre lies in: the first at the
// image's reference point, the second a degree east of it.
func TestEachPixelFallsWhereItLies(t *testing.T) {
	ra, dec := picked(t, "centre-ra"), picked(t, "centre-dec")
	file := picture.File(second, first, picture.Tangent(ra, dec, degree),
		[]float64{low, high})
	image := open(t, file)
	where, err := image.Sky()
	if err != nil {
		t.Fatal(err)
	}
	east, north := where.Place(second, first)
	laid := lay(t, file, pinDepth)
	for want, cell := range map[float64]*mesh.Cell{
		low:  locate(t, ra, dec, pinDepth),
		high: locate(t, east, north, pinDepth),
	} {
		if value, held := laid.At(*cell); !held || value.Mean() != want {
			t.Errorf("%#x holds %v, %v, not %v", cell.Key(), value, held,
				want)
		}
	}
}

// A galactic image is laid in ICRS: its pixel at the galactic north pole
// lies in the cell the pole's ICRS place falls in.
func TestGalacticImagesLieInICRS(t *testing.T) {
	file := picture.File(first, first, picture.Plate("'GLON-CAR'", "'GLAT-CAR'", pole),
		[]float64{low})
	laid := lay(t, file, pinDepth)
	cell := locate(t, poleAscension, poleDeclination, pinDepth)
	if !slices.Equal(laid.Cells(), []uint64{cell.Key()}) {
		t.Errorf("the pole lies in %v, not %#x", laid.Cells(), cell.Key())
	}
}

// Pixels whose place is past a pole are left out, in either frame. The
// pixel at the pole is kept.
func TestPlacesOffTheSkyAreLeftOut(t *testing.T) {
	for _, frame := range [][]string{picture.Plate("'RA---CAR'", "'DEC--CAR'", pole),
		picture.Plate("'GLON-CAR'", "'GLAT-CAR'", pole)} {
		laid := lay(t, picture.File(first, second, frame, []float64{low, high}),
			none)
		keys := laid.Cells()
		value := heldAt(t, laid, keys[none])
		if len(keys) != first || value.Pixels() != first || value.Mean() != low {
			t.Errorf("%v holds %v", keys, value)
		}
	}
}

// Pixels that are not a finite number are left out.
func TestNonFinitePixelsAreLeftOut(t *testing.T) {
	ra, dec := picked(t, "centre-ra"), picked(t, "centre-dec")
	values := []float64{math.NaN(), math.Inf(first), math.Inf(-first), low}
	placed := picture.Tangent(ra, dec, picked(t, "pixel-scale"))
	file := picture.File(len(values), first, placed, values)
	laid := lay(t, file, none)
	keys := laid.Cells()
	value := heldAt(t, laid, keys[none])
	if len(keys) != first || value.Pixels() != first || value.Mean() != low {
		t.Errorf("%v holds %v", keys, value)
	}
}

// Anti-tests: no image is refused, and so is an image with no place on
// the sky or no pixel left. A depth the mesh has no room for is refused as
// the mesh refuses it. A row cut short is an error.
func TestWhatCannotBeLaidIsRefused(t *testing.T) {
	if _, err := layer.Lay("none", nil, none, layer.Source{}); !errors.Is(
		err, layer.Refused()) || !strings.Contains(err.Error(), "none") {
		t.Errorf("no image: %v", err)
	}
	ra, dec := picked(t, "centre-ra"), picked(t, "centre-dec")
	placed := picture.Tangent(ra, dec, degree)
	nowhere := picture.File(first, first, nil, []float64{low})
	blank := picture.File(first, first, placed, []float64{math.NaN()})
	whole := picture.File(first, first, placed, []float64{low})
	for name, each := range map[string]struct {
		file  []byte
		depth int
		why   error
	}{
		"nowhere":   {nowhere, none, fits.Refused()},
		"blank":     {blank, none, layer.Refused()},
		"shallow":   {whole, -first, mesh.Refused()},
		"deep":      {whole, mesh.Deepest + first, mesh.Refused()},
		"cut short": {whole[:picture.Block], none, fits.Refused()},
	} {
		_, err := layer.Lay(name, open(t, each.file), each.depth,
			layer.Source{})
		if !errors.Is(err, each.why) ||
			!strings.Contains(err.Error(), "layer "+name) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
