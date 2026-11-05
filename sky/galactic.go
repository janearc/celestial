package sky

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// The galactic frame's place on the equatorial one in degrees, by the
// Hipparcos definition. It is set out in ESA SP-1200 volume 1 section
// 1.5.3.
const (
	// galacticPoleAscension is the right ascension of the north galactic
	// pole. It's defined.
	galacticPoleAscension = 192.85948
	// galacticPoleDeclination is the declination of the north galactic
	// pole. It's defined.
	galacticPoleDeclination = 27.12825
	// celestialPoleLongitude is the galactic longitude of the north
	// celestial pole. It's defined.
	celestialPoleLongitude = 122.93192
)

// clamp is the most a sine or a cosine can be. It's defined.
const clamp = 1

// Galactic is a direction in the galactic frame: longitude along the
// galactic plane from the centre, and latitude north of it.
type Galactic struct{ longitude, latitude measure.Angle }

// NewGalactic is the direction at a galactic longitude and latitude.
// returns nil and error for a longitude more than a turn either way, or a
// latitude past a pole.
func NewGalactic(longitude, latitude measure.Angle) (*Galactic, error) {
	along, up := longitude.InDegrees(), latitude.InDegrees()
	if !within(along, -degreesInATurn, degreesInATurn) ||
		!within(up, -pole, pole) {
		return nil, fmt.Errorf("sky: galactic longitude %g and "+
			"latitude %g, past a turn or a pole", along, up)
	}
	return &Galactic{longitude: longitude, latitude: latitude}, nil
}

// Longitude is the direction's galactic longitude, from the galactic
// centre along the plane.
func (seen Galactic) Longitude() measure.Angle { return seen.longitude }

// Latitude is the direction's galactic latitude, north of the plane.
func (seen Galactic) Latitude() measure.Angle { return seen.latitude }

// Galactic is the direction in the galactic frame, its longitude within
// one turn.
func (seen Equatorial) Galactic() Galactic {
	along, up := turned(seen.ascension.InRadians(),
		seen.declination.InRadians(), galacticPoleAscension,
		galacticPoleDeclination, celestialPoleLongitude)
	return Galactic{longitude: along, latitude: up}
}

// Equatorial is the direction in the equatorial frame, its right
// ascension within one turn.
func (seen Galactic) Equatorial() Equatorial {
	east, north := turned(seen.longitude.InRadians(),
		seen.latitude.InRadians(), celestialPoleLongitude,
		galacticPoleDeclination, galacticPoleAscension)
	return Equatorial{ascension: east, declination: north}
}

// turned is a direction in one frame as it lies in another. The other
// frame's pole lies at a longitude and latitude in this one, and this
// one's pole at a longitude in the other: all in degrees. The turn is the
// same rotation either way round, only with its two poles swapped.
func turned(lon, lat, poleLon, poleLat, backLon float64) (measure.Angle,
	measure.Angle) {
	sinLat, cosLat := math.Sincos(lat)
	poleAt := measure.Degrees(poleLat).InRadians()
	sinPole, cosPole := math.Sincos(poleAt)
	apart := lon - measure.Degrees(poleLon).InRadians()
	sinApart, cosApart := math.Sincos(apart)
	up := sinLat*sinPole + cosLat*cosPole*cosApart
	across := cosLat * sinApart
	along := sinLat*cosPole - cosLat*sinPole*cosApart
	back := measure.Degrees(backLon).InRadians()
	turn := measure.Degrees(degreesInATurn).InRadians()
	longitude := intoATurn(back-math.Atan2(across, along), turn)
	sine := math.Max(-clamp, math.Min(clamp, up))
	latitude := math.Asin(sine)
	return measure.Radians(longitude), measure.Radians(latitude)
}
