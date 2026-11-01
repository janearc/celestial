package sky

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
)

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

// OnEarth is the place at a latitude, a longitude and a height above
// sea level. returns nil and error for:
//   - a latitude past a pole, or a longitude past the date line
//   - a height below the earth's centre, or above the highest a place
//     on the earth stands, the top of the 1976 standard atmosphere
func OnEarth(latitude, longitude measure.Angle,
	height measure.Length) (*Place, error) {
	north, east := latitude.InDegrees(), longitude.InDegrees()
	if !within(north, -pole, pole) || !within(east, -dateLine, dateLine) {
		return nil, fmt.Errorf("sky: latitude %g and longitude %g, "+
			"past a pole or the date line", north, east)
	}
	metres := height.InMetres()
	if !within(metres, -earthRadius(), highestPlace()) {
		return nil, fmt.Errorf("sky: a height of %v m, outside %v "+
			"to %v", metres, -earthRadius(), highestPlace())
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

// within is whether a number lies from low to high, both ends
// included. NaN lies nowhere, since it fails every comparison.
func within(x, low, high float64) bool { return low <= x && x <= high }

// earthRadius is the earth's mean radius in metres, as celestial's
// research data holds it: no place on the earth is farther below sea
// level than its centre.
func earthRadius() float64 { return research.Value("earth-mean-radius-m") }

// highestPlace is the highest a place on the earth stands, in metres
// above sea level, as celestial's research data picks it: above it is
// orbit.
func highestPlace() float64 { return research.Value("highest-place-m") }
