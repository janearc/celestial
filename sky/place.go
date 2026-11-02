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

// Place describes a site on Earth. Latitude is positive north and longitude
// positive east, following the IAU convention; height is measured above sea
// level.
type Place struct {
	latitude, longitude measure.Angle
	height              measure.Length
}

// OnEarth constructs a site from latitude, longitude, and height. It rejects
// coordinates beyond the poles or date line, heights below the centre of Earth,
// and heights above the chosen 1976-atmosphere limit.
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

// Latitude returns angular distance north of the equator.
func (there Place) Latitude() measure.Angle { return there.latitude }

// Longitude returns angular distance east of Greenwich.
func (there Place) Longitude() measure.Angle { return there.longitude }

// Height returns elevation above sea level.
func (there Place) Height() measure.Length { return there.height }

// within checks the inclusive interval [low, high]. NaN fails the comparisons
// and is outside the interval.
func within(x, low, high float64) bool { return low <= x && x <= high }

// earthRadius reads Earth’s mean radius in metres from the research data,
// setting the lower height bound at Earth’s centre.
func earthRadius() float64 { return research.Value("earth-mean-radius-m") }

// highestPlace reads the chosen maximum site height in metres from research
// data. Higher locations are outside this Earth-site model.
func highestPlace() float64 { return research.Value("highest-place-m") }
