package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Equatorial stores right ascension east along the celestial equator and
// declination north of it.
type Equatorial struct{ ascension, declination measure.Angle }

// NewEquatorial constructs a direction from right ascension and declination. It
// rejects ascension outside one turn in either direction and declination beyond
// a pole.
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

// Ascension returns right ascension, positive east along the equator.
func (seen Equatorial) Ascension() measure.Angle { return seen.ascension }

// Declination returns declination, positive north of the equator.
func (seen Equatorial) Declination() measure.Angle {
	return seen.declination
}

// String formats the direction in degrees for diagnostics.
func (seen Equatorial) String() string {
	return fmt.Sprintf("ascension %g, declination %g degrees",
		seen.ascension.InDegrees(), seen.declination.InDegrees())
}

// Horizontal stores local azimuth from north toward east and altitude above the
// horizon.
type Horizontal struct{ azimuth, altitude measure.Angle }

// Azimuth returns the angle from north toward east.
func (seen Horizontal) Azimuth() measure.Angle { return seen.azimuth }

// Altitude returns angular height above the horizon.
func (seen Horizontal) Altitude() measure.Angle { return seen.altitude }

// Local converts an equatorial direction to a site’s horizontal frame using the
// supplied Greenwich sidereal angle. Use apparent sidereal time for apparent
// directions. The transform follows Meeus, Astronomical Algorithms, equations
// 13.5–13.6, with azimuth from north.
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
