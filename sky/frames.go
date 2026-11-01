package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Equatorial is a direction on the sky's own frame: right ascension
// east along the equator, and declination north of it.
type Equatorial struct{ ascension, declination measure.Angle }

// NewEquatorial is the direction at a right ascension and declination.
// returns nil and error for an ascension more than a turn either way,
// or a declination past a pole.
func NewEquatorial(ascension, declination measure.Angle) (*Equatorial,
	error) {
	east, north := ascension.InDegrees(), declination.InDegrees()
	if !within(east, -degreesInATurn, degreesInATurn) ||
		!within(north, -pole, pole) {
		return nil, fmt.Errorf("sky: ascension %g and declination %g, "+
			"past a turn or a pole", east, north)
	}
	return &Equatorial{ascension: ascension, declination: declination},
		nil
}

// Ascension is the direction's right ascension, east along the equator.
func (seen Equatorial) Ascension() measure.Angle { return seen.ascension }

// Declination is the direction's declination, north of the equator.
func (seen Equatorial) Declination() measure.Angle {
	return seen.declination
}

// String is the direction in degrees, for a person reading an error.
func (seen Equatorial) String() string {
	return fmt.Sprintf("ascension %g, declination %g degrees",
		seen.ascension.InDegrees(), seen.declination.InDegrees())
}

// Horizontal is a direction in the local sky: azimuth from north
// round toward east, and altitude above the horizon.
type Horizontal struct{ azimuth, altitude measure.Angle }

// Azimuth is how far round from north, toward east.
func (seen Horizontal) Azimuth() measure.Angle { return seen.azimuth }

// Altitude is how far above the horizon.
func (seen Horizontal) Altitude() measure.Angle { return seen.altitude }

// Local is a direction on the sky's frame as it stands in the place's
// sky, when greenwich has turned so far: its sidereal time. apparent
// directions take apparent sidereal time. by meeus, astronomical
// algorithms, equations 13.5 and 13.6, with azimuth from north.
func (there Place) Local(at Equatorial,
	sidereal measure.Angle) Horizontal {
	hour := sidereal.InRadians() + there.longitude.InRadians() -
		at.ascension.InRadians()
	north := there.latitude.InRadians()
	up := at.declination.InRadians()
	fromSouth := math.Atan2(math.Sin(hour)*math.Cos(up),
		math.Cos(hour)*math.Cos(up)*math.Sin(north)-
			math.Sin(up)*math.Cos(north))
	turn := measure.Degrees(degreesInATurn).InRadians()
	azimuth := intoATurn(fromSouth+math.Pi, turn)
	altitude := math.Asin(math.Sin(north)*math.Sin(up) +
		math.Cos(north)*math.Cos(up)*math.Cos(hour))
	return Horizontal{azimuth: measure.Radians(azimuth),
		altitude: measure.Radians(altitude)}
}
