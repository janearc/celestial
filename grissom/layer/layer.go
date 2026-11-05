// Package layer lays an image onto the sky's triangles. Each cell of the
// mesh at the layer's depth takes the mean of the image's pixels whose
// centres fall in it. The cells are in ICRS whatever frame the image is
// in. The layer keeps where its image came from.
package layer

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/janearc/celestial/fits"
	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/sky"
)

// The counts a layer is laid with.
const (
	// first is the number of the first pixel, as the FITS standard counts
	// them. It's defined.
	first = 1
	// none is no pixels, and either sign of infinity. It's defined.
	none = 0
)

// errRefused is what Refused gives.
var errRefused = errors.New("refused")

// Refused is wrapped by each refusal of this package's own: no image, or
// an image with no pixel on the sky. Depths the mesh has no room for are
// refused as the mesh refuses them.
func Refused() error { return errRefused }

// Source is where a layer's image came from: its entry in the fetch
// ledger.
type Source struct {
	url, licence, sum string
	fetched           time.Time
	size              int64
}

// NewSource is a ledger entry. It is where an image was fetched from and
// the terms it was published under, in the publisher's words. Then come
// when it was fetched, its length in bytes and its SHA-256 in hex.
func NewSource(url, licence string, fetched time.Time, size int64,
	sum string) Source {
	return Source{url: url, licence: licence, fetched: fetched, size: size,
		sum: sum}
}

// URL is where the image was fetched from.
func (from Source) URL() string { return from.url }

// Licence is the terms the image was published under, in the publisher's
// words.
func (from Source) Licence() string { return from.licence }

// Fetched is when the image was fetched.
func (from Source) Fetched() time.Time { return from.fetched }

// Size is the image's length in bytes.
func (from Source) Size() int64 { return from.size }

// Sum is the image's SHA-256, in hex.
func (from Source) Sum() string { return from.sum }

// Value is what a layer holds at one cell.
type Value struct {
	mean   float64
	pixels int
}

// Mean is the mean of the pixels that fell in the cell.
func (held Value) Mean() float64 { return held.mean }

// Pixels is how many pixels fell in the cell.
func (held Value) Pixels() int { return held.pixels }

// Layer is an image laid onto the mesh at one depth.
type Layer struct {
	name   string
	depth  int
	source Source
	cells  map[uint64]Value
}

// tally is the pixels counted into one cell so far.
type tally struct {
	sum    float64
	pixels int
}

// Lay lays an image onto the mesh at a depth. Each pixel counts in the cell
// its centre falls in. Pixels that are not a finite number are left out,
// and so are pixels whose place is off the sky.
//
// It refuses no image, an image whose place on the sky is not read, and
// an image with no pixel left. Rows cut short are an error.
func Lay(name string, image *fits.Image, depth int,
	source Source) (*Layer, error) {
	if image == nil {
		return nil, fmt.Errorf("%w: layer %s has no image", errRefused,
			name)
	}
	where, err := image.Sky()
	if err != nil {
		return nil, fmt.Errorf("layer %s: %w", name, err)
	}
	tallies := make(map[uint64]tally)
	for y := range image.Height() {
		if err := layRow(image, where, y, depth, tallies); err != nil {
			return nil, fmt.Errorf("layer %s: %w", name, err)
		}
	}
	if len(tallies) == none {
		return nil, fmt.Errorf("%w: no pixel of layer %s lies on the "+
			"sky", errRefused, name)
	}
	cells := make(map[uint64]Value, len(tallies))
	for key, kept := range tallies {
		mean := kept.sum / float64(kept.pixels)
		cells[key] = Value{mean: mean, pixels: kept.pixels}
	}
	return &Layer{name: name, depth: depth, source: source,
		cells: cells}, nil
}

// layRow counts one row's pixels into their cells.
func layRow(image *fits.Image, where *fits.Sky, y, depth int,
	tallies map[uint64]tally) error {
	row, err := image.Row(y)
	if err != nil {
		return err
	}
	for x, value := range row {
		if math.IsNaN(value) || math.IsInf(value, none) {
			continue
		}
		lon, lat := where.Place(float64(x+first), float64(y+first))
		place, err := icrs(where.Frame(), lon, lat)
		if err != nil {
			continue
		}
		cell, err := mesh.Locate(place, depth)
		if err != nil {
			return err
		}
		kept := tallies[cell.Key()]
		kept.sum += value
		kept.pixels++
		tallies[cell.Key()] = kept
	}
	return nil
}

// icrs is a place in an image's frame, as it lies in ICRS. Places off the
// sky are refused.
func icrs(frame fits.Frame, lon, lat float64) (*mesh.Point, error) {
	if frame == fits.Galactic {
		seen, err := sky.NewGalactic(measure.Degrees(lon),
			measure.Degrees(lat))
		if err != nil {
			return nil, err
		}
		turned := seen.Equatorial()
		lon = turned.Ascension().InDegrees()
		lat = turned.Declination().InDegrees()
	}
	return mesh.NewPoint(lon, lat)
}

// Name is what the layer is called.
func (layer *Layer) Name() string { return layer.name }

// Depth is the depth of the cells the layer holds.
func (layer *Layer) Depth() int { return layer.depth }

// Source is where the layer's image came from.
func (layer *Layer) Source() Source { return layer.source }

// At is the layer's value at a cell. There is none at a cell no pixel fell
// in, nor at a cell of another depth.
func (layer *Layer) At(cell mesh.Cell) (Value, bool) {
	value, held := layer.cells[cell.Key()]
	return value, held
}

// Cells are the keys of the cells the layer holds, in order.
func (layer *Layer) Cells() []uint64 {
	return slices.Sorted(maps.Keys(layer.cells))
}
