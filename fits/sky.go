package fits

import (
	"fmt"
	"math"
	"strings"
)

// Frame is the coordinates an image's axes are in.
type Frame string

// The frames an image may be in.
const (
	// Equatorial is right ascension and declination, ICRS. FK5 at the
	// J2000 equinox is read as ICRS: the two differ by tens of
	// milliarcseconds, far below any pixel here.
	Equatorial Frame = "equatorial"
	// Galactic is galactic longitude and latitude.
	Galactic Frame = "galactic"
)

// The projections read, by the code CTYPE ends in. Each is defined by
// Calabretta and Greisen 2002, the world coordinates paper II.
const (
	tangent     = "TAN" // defined: the gnomonic projection, from the centre
	cylindrical = "CAR" // defined: the plate carree, a plain grid
)

// The angles the projections work with, in degrees.
const (
	halfTurn = 180  // defined: half a turn
	fullTurn = 360  // defined: a turn
	equator  = 0    // defined: the latitude of the equator
	j2000    = 2000 // defined: the equinox FK5 and ICRS agree at
	// tangentPole is LONPOLE's default for a zenithal projection. It's
	// defined by paper II, section 2.4.
	tangentPole = 180
	// carPole is LONPOLE's default for CAR about the equator. It's
	// defined by paper II, section 2.4.
	carPole = 0
)

// Sky is where an image's pixels lie on the sky: a projection about a
// reference point, and the matrix that turns pixels into degrees.
type Sky struct {
	frame      Frame
	projection string
	lon0, lat0 float64
	px0, py0   float64
	m11, m12   float64
	m21, m22   float64
	in11, in12 float64
	in21, in22 float64
}

// Sky is where the image's pixels lie on the sky.
func (image *Image) Sky() (*Sky, error) { return SkyOf(image.header) }

// SkyOf is where a header says its image lies on the sky. It refuses a
// projection or a frame it does not read, and axes not in degrees.
//
// It refuses a CAR grid not about the equator and a pole moved from where
// the projection puts it. It refuses an equinox other than J2000 and a
// pixel matrix with no inverse.
func SkyOf(header *Header) (*Sky, error) {
	frame, projection, err := axesOf(header)
	if err != nil {
		return nil, err
	}
	if err := equinoxOf(header, frame); err != nil {
		return nil, err
	}
	if err := unitsOf(header); err != nil {
		return nil, err
	}
	read := &Sky{frame: frame, projection: projection}
	values := []*float64{&read.lon0, &read.lat0, &read.px0, &read.py0}
	for i, key := range []string{"CRVAL1", "CRVAL2", "CRPIX1", "CRPIX2"} {
		value, err := header.Number(key)
		if err != nil {
			return nil, err
		}
		*values[i] = value
	}
	if projection == cylindrical && read.lat0 != equator {
		return nil, fmt.Errorf("%w: a CAR grid about latitude %v, not "+
			"the equator", errRefused, read.lat0)
	}
	if err := poleOf(header, projection); err != nil {
		return nil, err
	}
	return read, read.matrix(header)
}

// axesOf is the frame and the projection CTYPE1 and CTYPE2 name.
func axesOf(header *Header) (Frame, string, error) {
	first, firstErr := header.Text("CTYPE1")
	second, secondErr := header.Text("CTYPE2")
	if firstErr != nil || secondErr != nil {
		return "", "", fmt.Errorf("%w: no CTYPE1 and CTYPE2",
			errRefused)
	}
	lon, lonCode := split(first)
	lat, latCode := split(second)
	if lonCode != latCode ||
		(lonCode != tangent && lonCode != cylindrical) {
		return "", "", fmt.Errorf("%w: the projection %q, %q",
			errRefused, first, second)
	}
	switch {
	case lon == "RA" && lat == "DEC":
		return Equatorial, lonCode, nil
	case lon == "GLON" && lat == "GLAT":
		return Galactic, lonCode, nil
	}
	return "", "", fmt.Errorf("%w: the axes %q, %q", errRefused, first,
		second)
}

// split is a CTYPE's axis and its projection's code. Some writers pad
// between them with more hyphens than the standard's, as GLON---TAN.
func split(ctype string) (string, string) {
	axis, _, _ := strings.Cut(ctype, "-")
	code := ctype[strings.LastIndex(ctype, "-")+len("-"):]
	return strings.TrimSpace(axis), strings.TrimSpace(code)
}

// equinoxOf refuses an equatorial frame at another equinox than J2000, or
// in FK4.
func equinoxOf(header *Header, frame Frame) error {
	if frame != Equatorial {
		return nil
	}
	system, err := header.Text("RADESYS")
	if err == nil && system != "ICRS" && system != "FK5" {
		return fmt.Errorf("%w: RADESYS %q", errRefused, system)
	}
	if header.Has("EQUINOX") &&
		numberOr(header, "EQUINOX", j2000) != j2000 {
		return fmt.Errorf("%w: an equinox other than J2000", errRefused)
	}
	return nil
}

// unitsOf refuses axes in any unit but degrees. `CRVAL`, `CDELT` and `CD`
// are read as degrees; an axis without `CUNIT` is in degrees by default,
// and one whose `CUNIT` says otherwise, as arcseconds, would be read wrong.
func unitsOf(header *Header) error {
	for _, key := range []string{"CUNIT1", "CUNIT2"} {
		if !header.Has(key) {
			continue
		}
		unit, err := header.Text(key)
		if err != nil || unit != degreeUnit {
			return fmt.Errorf("%w: %s %q, not degrees", errRefused,
				key, unit)
		}
	}
	return nil
}

// degreeUnit is how `CUNIT` names a degree. It's defined by the standard.
const degreeUnit = "deg"

// poleOf refuses a LONPOLE other than the projection's own.
func poleOf(header *Header, projection string) error {
	want := float64(carPole)
	if projection == tangent {
		want = tangentPole
	}
	if numberOr(header, "LONPOLE", want) != want {
		return fmt.Errorf("%w: LONPOLE moved from %v", errRefused, want)
	}
	return nil
}

// matrix reads the pixel matrix. Paper II's section 6.1 gives three ways
// to write it, read here in turn: CD; PC times CDELT; CDELT and CROTA2.
func (read *Sky) matrix(header *Header) error {
	cd := []string{"CD1_1", "CD1_2", "CD2_1", "CD2_2"}
	elements := []*float64{&read.m11, &read.m12, &read.m21, &read.m22}
	if anyOf(header, cd) {
		for i, key := range cd {
			*elements[i] = numberOr(header, key, equator)
		}
		return read.inverted()
	}
	across, acrossErr := header.Number("CDELT1")
	up, upErr := header.Number("CDELT2")
	if acrossErr != nil || upErr != nil {
		return fmt.Errorf("%w: neither CD nor CDELT", errRefused)
	}
	pc := []string{"PC1_1", "PC1_2", "PC2_1", "PC2_2"}
	if anyOf(header, pc) {
		identity := []float64{flat, equator, equator, flat}
		scale := []float64{across, across, up, up}
		for i, key := range pc {
			element := numberOr(header, key, identity[i])
			*elements[i] = scale[i] * element
		}
		return read.inverted()
	}
	turn := numberOr(header, "CROTA2", equator) * math.Pi / halfTurn
	sin, cos := math.Sincos(turn)
	read.m11, read.m12 = across*cos, -up*sin
	read.m21, read.m22 = across*sin, up*cos
	return read.inverted()
}

// anyOf is whether a header has any of some keywords.
func anyOf(header *Header, keys []string) bool {
	for _, key := range keys {
		if header.Has(key) {
			return true
		}
	}
	return false
}

// inverted keeps the matrix's inverse, or refuses one with none.
func (read *Sky) inverted() error {
	determinant := read.m11*read.m22 - read.m12*read.m21
	if determinant == equator || math.IsNaN(determinant) {
		return fmt.Errorf("%w: a pixel matrix with no inverse",
			errRefused)
	}
	read.in11, read.in12 = read.m22/determinant, -read.m12/determinant
	read.in21, read.in22 = -read.m21/determinant, read.m11/determinant
	return nil
}

// Frame is the coordinates the image's axes are in.
func (read *Sky) Frame() Frame { return read.frame }

// Place is the longitude and latitude at a pixel, in degrees in the
// image's frame. Pixels are counted as the standard counts them: the
// centre of the first is at one.
func (read *Sky) Place(x, y float64) (lon, lat float64) {
	dx, dy := x-read.px0, y-read.py0
	east := read.m11*dx + read.m12*dy
	north := read.m21*dx + read.m22*dy
	if read.projection == cylindrical {
		return onceRound(read.lon0 + east), north
	}
	xi, eta := radians(east), radians(north)
	sinLat0, cosLat0 := math.Sincos(radians(read.lat0))
	across := cosLat0 - eta*sinLat0
	turned := math.Atan2(xi, across)
	raised := math.Atan2(sinLat0+eta*cosLat0, math.Hypot(xi, across))
	return onceRound(read.lon0 + degrees(turned)), degrees(raised)
}

// Pixel is the pixel at a longitude and latitude, in degrees in the
// image's frame. seen is false for a place a tangent projection cannot
// show: on the far half of the sky from its centre.
func (read *Sky) Pixel(lon, lat float64) (x, y float64, seen bool) {
	east, north := apart(read.lon0, lon), lat
	if read.projection == tangent {
		var ahead bool
		east, north, ahead = read.standard(lon, lat)
		if !ahead {
			return math.NaN(), math.NaN(), false
		}
	}
	x = read.px0 + read.in11*east + read.in12*north
	y = read.py0 + read.in21*east + read.in22*north
	return x, y, true
}

// standard is a place's standard coordinates about the tangent point in
// degrees. It says whether the place lies on the near half of the sky.
func (read *Sky) standard(lon, lat float64) (float64, float64, bool) {
	sinLat, cosLat := math.Sincos(radians(lat))
	sinLat0, cosLat0 := math.Sincos(radians(read.lat0))
	sinApart, cosApart := math.Sincos(radians(lon - read.lon0))
	toward := sinLat*sinLat0 + cosLat*cosLat0*cosApart
	if toward <= equator {
		return math.NaN(), math.NaN(), false
	}
	xi := cosLat * sinApart / toward
	eta := (sinLat*cosLat0 - cosLat*sinLat0*cosApart) / toward
	return degrees(xi), degrees(eta), true
}

// radians is an angle in degrees, in radians.
func radians(angle float64) float64 { return angle * math.Pi / halfTurn }

// degrees is an angle in radians, in degrees.
func degrees(angle float64) float64 { return angle * halfTurn / math.Pi }

// onceRound is a longitude brought from 0 to under 360.
func onceRound(lon float64) float64 {
	within := math.Mod(lon, fullTurn) + fullTurn
	return math.Mod(within, fullTurn)
}

// apart is the difference of two longitudes taken the short way round,
// from -180 to under 180.
func apart(from, to float64) float64 {
	return onceRound(to-from+halfTurn) - halfTurn
}
