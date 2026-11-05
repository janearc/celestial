package fits_test

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/celestial/fits"
	"pgregory.net/rapid"
)

// The angles the sky tests work with, in degrees.
const (
	halfTurn = 180  // defined: half a turn
	fullTurn = 360  // defined: a turn
	step     = 0.01 // defined: the test grids' pixel, a hundredth of a degree
	centre   = 51   // defined: the test grids' reference pixel
	along    = 100  // defined: how many pixels along the tests step
	// reference is the test grids' reference longitude. It's defined.
	reference = 10
)

// real is one of the mosaics fetched on the night of the Sgr A* page. It
// is named by its header in testdata, and says its frame.
type real struct {
	name  string
	frame fits.Frame
}

// reals are the four mosaics: the projections and frames read here.
var reals = []real{{"glimpse-ii-I4", fits.Galactic},
	{"hpsgc-palpha", fits.Galactic}, {"meerkat-stokesi", fits.Equatorial},
	{"ppmap-l000-cdens", fits.Galactic}}

// stopper is a test or a property's draw that can be stopped.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// opened is a real mosaic's image, from its header alone.
func opened(test stopper, name string) *fits.Image {
	test.Helper()
	file, err := os.Open("testdata/" + name + ".header")
	if err != nil {
		test.Fatal(err)
	}
	read, err := fits.Open(file)
	if err != nil {
		test.Fatal(err)
	}
	return read
}

// The reference pixel of each mosaic lies at its reference value, in the
// frame its axes name.
func TestTheReferencePixelIsTheReferenceValue(t *testing.T) {
	within := picked(t, "degree-rounding")
	for _, each := range reals {
		read := opened(t, each.name)
		sky, err := read.Sky()
		if err != nil {
			t.Fatal(each.name, err)
		}
		header := read.Header()
		x, _ := header.Number("CRPIX1")
		y, _ := header.Number("CRPIX2")
		lon0, _ := header.Number("CRVAL1")
		lat0, _ := header.Number("CRVAL2")
		lon, lat := sky.Place(x, y)
		if sky.Frame() != each.frame ||
			!near(math.Remainder(lon-lon0, fullTurn), none, within) ||
			!near(lat, lat0, within) {
			t.Errorf("%s: %v at %v, %v", each.name, sky.Frame(), lon, lat)
		}
	}
}

// returns is the property of TestEveryPixelComesBack, for one draw.
func returns(t *rapid.T) {
	each := rapid.SampledFrom(reals).Draw(t, "mosaic")
	read := opened(t, each.name)
	sky, err := read.Sky()
	if err != nil {
		t.Fatal(err)
	}
	width, height := float64(read.Width()), float64(read.Height())
	x := rapid.Float64Range(none, width).Draw(t, "x")
	y := rapid.Float64Range(none, height).Draw(t, "y")
	lon, lat := sky.Place(x, y)
	backX, backY, seen := sky.Pixel(lon, lat)
	if !seen || !near(backX, x, rounding) || !near(backY, y, rounding) {
		t.Fatalf("%s: %v, %v came back %v, %v", each.name, x, y, backX,
			backY)
	}
}

// rounding is the pixel rounding for the property. The property has no
// *testing.T to read it with. TestEveryPixelComesBack fills it.
var rounding float64

// Property: every pixel of every mosaic goes to the sky and comes back to
// itself.
func TestEveryPixelComesBack(t *testing.T) {
	rounding = picked(t, "pixel-rounding")
	rapid.Check(t, returns)
}

// grid is a test image's header. Its frame's axes are in a projection
// about a place, a hundredth of a degree a pixel. More cards come after.
func grid(t *testing.T, lon, lat, projection string,
	more ...string) *fits.Sky {
	t.Helper()
	sky, err := gridOf(lon, lat, projection, more...)
	if err != nil {
		t.Fatal(err)
	}
	return sky
}

// gridOf is grid's sky, or why there is none.
func gridOf(lon, lat, projection string, more ...string) (*fits.Sky,
	error) {
	cards := append([]string{card("SIMPLE", "T"), card("BITPIX", "-32"),
		card("NAXIS", "2"), card("NAXIS1", "101"), card("NAXIS2", "101"),
		card("CTYPE1", "'"+lon+"-"+projection+"'"),
		card("CTYPE2", "'"+lat+"-"+projection+"'"),
		card("CRPIX1", "51"), card("CRPIX2", "51")}, more...)
	header, err := fits.ReadHeader(fileOf(cards, nil))
	if err != nil {
		return nil, err
	}
	return fits.SkyOf(header)
}

// The cards that put a grid at a place a hundredth of a degree a pixel.
// East is to the left, as on the sky.
var at10 = []string{card("CRVAL1", "10"), card("CRVAL2", "0"),
	card("CDELT1", "-0.01"), card("CDELT2", "0.01")}

// Along the equator the pixels of tangent grids are the tangent of their
// longitude from the centre. One degree west of the centre's pixel is the
// arctangent of one degree's worth.
func TestTangentAlongTheEquatorIsTheArctangent(t *testing.T) {
	sky := grid(t, "RA--", "DEC-", "TAN", at10...)
	lon, lat := sky.Place(centre+along, centre)
	want := reference - math.Atan(along*step*math.Pi/halfTurn)*halfTurn/
		math.Pi
	within := picked(t, "degree-rounding")
	if !near(lon, want, within) || !near(lat, none, within) {
		t.Errorf("%v, %v; want %v", lon, lat, want)
	}
}

// CAR grids are plain: each pixel is a hundredth of a degree everywhere.
func TestCarIsAPlainGrid(t *testing.T) {
	sky := grid(t, "GLON", "GLAT", "CAR", at10...)
	lon, lat := sky.Place(centre+along, centre+along)
	within := picked(t, "degree-rounding")
	if !near(lon, reference-along*step, within) || !near(lat, along*step, within) {
		t.Errorf("%v, %v", lon, lat)
	}
}

// The pixel matrix is read however it is written. CD is read. So is PC
// times CDELT, and so is CDELT turned by CROTA2. Each here turns the grid
// a quarter. So a pixel up the image lies along the equator.
func TestThePixelMatrixIsReadEveryWay(t *testing.T) {
	place := []string{card("CRVAL1", "10"), card("CRVAL2", "0")}
	spacing := []string{card("CDELT1", "-0.01"), card("CDELT2", "0.01")}
	for way, cards := range map[string][]string{
		"CD": append(place, card("CD1_1", "0"), card("CD1_2", "-0.01"),
			card("CD2_1", "-0.01"), card("CD2_2", "0")),
		"PC": append(append(place, spacing...), card("PC1_1", "0"),
			card("PC1_2", "1"), card("PC2_1", "1"), card("PC2_2", "0")),
		"CROTA2": append(append(place, spacing...), card("CROTA2", "90")),
	} {
		sky := grid(t, "GLON", "GLAT", "CAR", cards...)
		lon, lat := sky.Place(centre, centre+along)
		within := picked(t, "degree-rounding")
		if !near(lon, reference-along*step, within) || !near(lat, none, within) {
			t.Errorf("%s: %v, %v", way, lon, lat)
		}
	}
}

// Tangent grids cannot show the half of the sky behind them.
func TestATangentGridCannotSeeBehindIt(t *testing.T) {
	sky := grid(t, "RA--", "DEC-", "TAN", at10...)
	if x, y, seen := sky.Pixel(reference+halfTurn, none); seen ||
		!math.IsNaN(x) || !math.IsNaN(y) {
		t.Errorf("behind: %v, %v", x, y)
	}
}

// Anti-tests: what this package does not read is refused, by name.
func TestWhatIsNotReadIsRefused(t *testing.T) {
	for want, cards := range map[string]struct {
		lon, lat, projection string
		more                 []string
	}{
		"the projection":       {"RA--", "DEC-", "SIN", at10},
		"the axes":             {"RA--", "GLAT", "TAN", at10},
		`RADESYS "FK4"`:        {"RA--", "DEC-", "TAN", append(at10, card("RADESYS", "'FK4'"))},
		"other than J2000":     {"RA--", "DEC-", "TAN", append(at10, card("EQUINOX", "1950"))},
		"not the equator":      {"GLON", "GLAT", "CAR", []string{card("CRVAL1", "0"), card("CRVAL2", "5"), card("CDELT1", "1"), card("CDELT2", "1")}},
		"LONPOLE moved":        {"RA--", "DEC-", "TAN", append(at10, card("LONPOLE", "90"))},
		"no inverse":           {"RA--", "DEC-", "TAN", []string{card("CRVAL1", "10"), card("CRVAL2", "0"), card("CD1_1", "0")}},
		"no CRVAL2":            {"RA--", "DEC-", "TAN", []string{card("CRVAL1", "10")}},
		"neither CD nor CDELT": {"RA--", "DEC-", "TAN", []string{card("CRVAL1", "10"), card("CRVAL2", "0")}},
		`CUNIT1 "arcsec"`:      {"RA--", "DEC-", "TAN", append(at10, card("CUNIT1", "'arcsec'"))},
		`CUNIT2 "rad"`:         {"GLON", "GLAT", "CAR", append(at10, card("CUNIT2", "'rad'"))},
	} {
		_, err := gridOf(cards.lon, cards.lat, cards.projection,
			cards.more...)
		if !errors.Is(err, fits.Refused()) ||
			!strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
	simple := []string{card("SIMPLE", "T")}
	header, err := fits.ReadHeader(fileOf(simple, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fits.SkyOf(header); !errors.Is(err, fits.Refused()) {
		t.Errorf("no CTYPE: %v", err)
	}
}

// ICRS and FK5 at J2000 are both read, as the same frame.
func TestICRSAndFK5AreRead(t *testing.T) {
	for _, system := range []string{"'ICRS'", "'FK5'"} {
		cards := append(at10, card("RADESYS", system),
			card("EQUINOX", "2000"))
		sky := grid(t, "RA--", "DEC-", "TAN", cards...)
		if sky.Frame() != fits.Equatorial {
			t.Errorf("%s: %v", system, sky.Frame())
		}
	}
}

// Each projection's own pole is read; poles moved from it are not.
// Galactic grids' equinoxes are no concern of their frame.
func TestEachProjectionsOwnPoleIsRead(t *testing.T) {
	for why, cards := range map[string]struct {
		lon, lat, projection string
		more                 []string
		read                 bool
	}{
		"TAN at 180": {"RA--", "DEC-", "TAN", append(at10,
			card("LONPOLE", "180")), true},
		"CAR at 0": {"GLON", "GLAT", "CAR", append(at10,
			card("LONPOLE", "0")), true},
		"CAR at 180": {"GLON", "GLAT", "CAR", append(at10,
			card("LONPOLE", "180")), false},
		"galactic in 1950": {"GLON", "GLAT", "TAN", append(at10,
			card("EQUINOX", "1950"), card("RADESYS", "'FK4'")), true},
		"one CDELT": {"GLON", "GLAT", "TAN", []string{card("CRVAL1", "0"),
			card("CRVAL2", "0"), card("CDELT1", "1")}, false},
	} {
		_, err := gridOf(cards.lon, cards.lat, cards.projection,
			cards.more...)
		if (err == nil) != cards.read {
			t.Errorf("%s: %v", why, err)
		}
	}
}

// Anti-test: a header with one CTYPE says it lacks the other.
func TestOneCtypeIsNotEnough(t *testing.T) {
	cards := []string{card("SIMPLE", "T"), card("CTYPE1", "'RA---TAN'")}
	header, err := fits.ReadHeader(fileOf(cards, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fits.SkyOf(header); err == nil ||
		!strings.Contains(err.Error(), "no CTYPE1 and CTYPE2") {
		t.Errorf("%v", err)
	}
}

// Grids turned a quarter by CROTA2 lay their rows up the sky: pixels
// along the row lie south of the centre.
func TestATurnedGridsRowsRunNorthAndSouth(t *testing.T) {
	cards := append(at10, card("CROTA2", "90"))
	sky := grid(t, "GLON", "GLAT", "CAR", cards...)
	lon, lat := sky.Place(centre+along, centre)
	within := picked(t, "degree-rounding")
	if !near(lon, reference, within) || !near(lat, -along*step, within) {
		t.Errorf("%v, %v", lon, lat)
	}
}

// Anti-test: axes that say they are in degrees, padded as writers pad
// them, are read, and lie where axes that do not say so lie.
func TestAxesInDegreesAreRead(t *testing.T) {
	plain := grid(t, "GLON", "GLAT", "CAR", at10...)
	units := []string{card("CUNIT1", "'deg     '"), card("CUNIT2", "'deg'")}
	said := grid(t, "GLON", "GLAT", "CAR", append(at10, units...)...)
	lon, lat := said.Place(centre+along, centre+along)
	wantLon, wantLat := plain.Place(centre+along, centre+along)
	if lon != wantLon || lat != wantLat {
		t.Errorf("%v, %v; want %v, %v", lon, lat, wantLon, wantLat)
	}
}

// A grid turned half round by CROTA2 faces the other way: its rows run
// east and its columns south. The matrix CDELT1 cos, -CDELT2 sin, CDELT1
// sin, CDELT2 cos is 0.01, 0, 0, -0.01 there, worked by hand. A quarter
// turn has no cosine, so this holds the cosine's sign; the quarter, the
// sine's.
func TestAGridTurnedHalfRoundFacesTheOtherWay(t *testing.T) {
	cards := append(at10, card("CROTA2", "180"))
	sky := grid(t, "GLON", "GLAT", "CAR", cards...)
	within := picked(t, "degree-rounding")
	lon, lat := sky.Place(centre+along, centre)
	if !near(lon, reference+along*step, within) || !near(lat, none, within) {
		t.Errorf("along the row: %v, %v", lon, lat)
	}
	lon, lat = sky.Place(centre, centre+along)
	if !near(lon, reference, within) || !near(lat, -along*step, within) {
		t.Errorf("up the column: %v, %v", lon, lat)
	}
}

// turned are test grids whose matrices are turned and sheared. They sit
// away from the zero of longitude, as rotated survey tiles do.
func turned(t *testing.T) []*fits.Sky {
	t.Helper()
	place := []string{card("CRVAL1", "10"), card("CRVAL2", "20")}
	spacing := []string{card("CDELT1", "-0.01"), card("CDELT2", "0.01"),
		card("CROTA2", "30")}
	sheared := []string{card("CD1_1", "-0.009"), card("CD1_2", "0.003"),
		card("CD2_1", "0.002"), card("CD2_2", "0.008")}
	equator := []string{card("CRVAL1", "10"), card("CRVAL2", "0"),
		card("CDELT1", "-0.01"), card("CDELT2", "0.01"),
		card("CROTA2", "30")}
	return []*fits.Sky{
		grid(t, "RA--", "DEC-", "TAN", append(place, spacing...)...),
		grid(t, "RA--", "DEC-", "TAN", append(place, sheared...)...),
		grid(t, "GLON", "GLAT", "CAR", equator...),
	}
}

// turnedSkies are the turned grids, for the property. TestTurnedPixels
// fills them.
var turnedSkies []*fits.Sky

// turnedReturns is the property of TestTurnedPixelsComeBack, for one
// draw.
func turnedReturns(t *rapid.T) {
	sky := rapid.SampledFrom(turnedSkies).Draw(t, "grid")
	x := rapid.Float64Range(none, gridSide).Draw(t, "x")
	y := rapid.Float64Range(none, gridSide).Draw(t, "y")
	lon, lat := sky.Place(x, y)
	backX, backY, seen := sky.Pixel(lon, lat)
	if !seen || !near(backX, x, rounding) || !near(backY, y, rounding) {
		t.Fatalf("%v, %v came back %v, %v", x, y, backX, backY)
	}
}

// gridSide is the test grids' pixels a side. It's defined by their cards.
const gridSide = 101

// Property: every pixel of a turned or sheared grid goes to the sky and
// comes back to itself.
func TestTurnedPixelsComeBack(t *testing.T) {
	rounding = picked(t, "pixel-rounding")
	turnedSkies = turned(t)
	rapid.Check(t, turnedReturns)
}

// Anti-test: one CDELT without the other is refused as no matrix at all.
func TestOneCdeltIsNotEnough(t *testing.T) {
	_, err := gridOf("GLON", "GLAT", "TAN", card("CRVAL1", "0"),
		card("CRVAL2", "0"), card("CDELT1", "1"))
	if !errors.Is(err, fits.Refused()) ||
		!strings.Contains(err.Error(), "neither CD nor CDELT") {
		t.Errorf("%v", err)
	}
}
