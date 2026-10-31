package sky

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// errPlace is a place or a direction that is not one.
var errPlace = errors.New("celestial: no such place")

// ErrPlace is a place or a direction that is not one: past a pole, past
// the date line, or not a number.
func ErrPlace() error { return errPlace }

// The ends of latitude and longitude, in degrees, each defined: a pole
// is a quarter turn from the equator, and the date line half a turn from
// greenwich.
const (
	pole     = 90.0  // defined by the degree
	dateLine = 180.0 // defined by the degree
)

// Place is a place on the earth: its latitude, north above nought; its
// longitude, east above nought, as the iau counts it; and its height.
type Place struct {
	latitude, longitude measure.Angle
	height              measure.Length
}

// OnEarth is the place at a latitude, a longitude and a height. It
// refuses a latitude past a pole, a longitude past the date line, and a
// height that is not a finite number, and then gives no place.
func OnEarth(latitude, longitude measure.Angle,
	height measure.Length) (*Place, error) {
	north, east := latitude.InDegrees(), longitude.InDegrees()
	if !(math.Abs(north) <= pole) || !(math.Abs(east) <= dateLine) ||
		!finiteNumber(height.InMetres()) {
		return nil, fmt.Errorf("%w: latitude %g, longitude %g",
			errPlace, north, east)
	}
	return &Place{latitude: latitude, longitude: longitude,
		height: height}, nil
}

// Latitude is how far north the place is.
func (there Place) Latitude() measure.Angle { return there.latitude }

// Longitude is how far east the place is.
func (there Place) Longitude() measure.Angle { return there.longitude }

// Height is how high the place stands.
func (there Place) Height() measure.Length { return there.height }

// finiteNumber is whether a number is one and has an end.
func finiteNumber(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, eitherSign)
}
