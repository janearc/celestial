// Package galactocentric places a star as Gaia sees it in the galaxy:
// from where it is on the sky, how far it is by its parallax and how it
// moves across the sky and along the line of sight, to its place and
// motion about the galactic centre, where orbit's solver runs it back.
//
// Seen turns that round: from a star's place and motion about the centre
// to where the Sun sees it and how, so that a star made in the frame can
// be observed as Gaia would observe it.
//
// The frame is the one astropy's Galactocentric frame describes: the
// centre at Sgr A*, the z axis toward the north galactic pole tilted so
// that the Sun sits at its measured height above the mid-plane.
//
// The x axis runs from the Sun's side toward the centre, and the y
// axis along the galaxy's rotation at the Sun.
//
// The seam with orbit's Milky Way is open: the published frame's Sun
// goes 251.5 km/s along the rotation, Sgr A*'s apparent motion
// reflected, while MWPotential2014 at the Sun's measured distance gives
// it about 231, its circular speed there plus the Sun's own motion.
//
// Every star this frame places carries the frame's Sun; run back in the
// model as it stands, it carries 20 km/s along the rotation the model
// did not give the Sun. Placing the model at the frame's Sun, or the
// frame's Sun at the model's speed, is a choice not made here.
//
// Lengths are kiloparsecs and speeds kilometres per second inside; the
// surface is measure's types.
package galactocentric

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
	"github.com/janearc/celestial/si"
	"github.com/janearc/celestial/sky"
)

// The package's own numbers, each defined.
const (
	whole          = 1.0  // defined: the whole of a share
	nothing        = 0.0  // defined: no length, no speed
	masInArcsecond = 1000 // defined: a thousand milliarcseconds in one
	axes           = 3    // defined: three axes
)

// turn is a whole turn in radians.
const turn = 2 * math.Pi // defined: a circle is two pi radians round

// kiloparsec is the metres in one.
var kiloparsec = si.Kilo(length.Parsec)

// masYearKpc is the speed in kilometres per second of a star at a
// kiloparsec whose proper motion is a milliarcsecond a year: an
// astronomical unit a Julian year, since a milliarcsecond at a
// kiloparsec is an astronomical unit by the parsec's definition.
var masYearKpc = length.AU / si.Kilo(length.Metre) /
	(length.LightYear / si.Speed)

// triple is three numbers along the axes.
type triple [axes]float64

// plus is the sum of two triples.
func (one triple) plus(other triple) triple {
	return triple{one[axisX] + other[axisX], one[axisY] + other[axisY],
		one[axisZ] + other[axisZ]}
}

// times is the triple scaled.
func (one triple) times(by float64) triple {
	return triple{one[axisX] * by, one[axisY] * by, one[axisZ] * by}
}

// dot is the scalar product of two triples.
func (one triple) dot(other triple) float64 {
	return one[axisX]*other[axisX] + one[axisY]*other[axisY] +
		one[axisZ]*other[axisZ]
}

// cross is the vector product of two triples.
func (one triple) cross(other triple) triple {
	return triple{
		one[axisY]*other[axisZ] - one[axisZ]*other[axisY],
		one[axisZ]*other[axisX] - one[axisX]*other[axisZ],
		one[axisX]*other[axisY] - one[axisY]*other[axisX]}
}

// unit is the triple scaled to a length of one.
func (one triple) unit() triple {
	size := math.Sqrt(one.dot(one))
	return one.times(whole / size)
}

// The axes' places in a triple, each defined.
const (
	axisX = iota
	axisY
	axisZ
)

// toward is the unit triple toward a direction on the sky, in the
// equatorial frame's axes: x toward the equinox, z toward the north
// celestial pole.
func toward(seen sky.Equatorial) triple {
	ascension := seen.Ascension().InRadians()
	declination := seen.Declination().InRadians()
	return triple{math.Cos(declination) * math.Cos(ascension),
		math.Cos(declination) * math.Sin(ascension), math.Sin(declination)}
}

// east is the unit triple along which a star's proper motion in
// ascension moves it: along increasing right ascension.
func east(seen sky.Equatorial) triple {
	ascension := seen.Ascension().InRadians()
	return triple{-math.Sin(ascension), math.Cos(ascension), nothing}
}

// north is the unit triple along which a star's proper motion in
// declination moves it: along increasing declination.
func north(seen sky.Equatorial) triple {
	ascension := seen.Ascension().InRadians()
	declination := seen.Declination().InRadians()
	return triple{-math.Sin(declination) * math.Cos(ascension),
		-math.Sin(declination) * math.Sin(ascension), math.Cos(declination)}
}

// Frame is the galactocentric frame: its axes on the equatorial frame,
// and the Sun's place and motion in it.
type Frame struct {
	axes      [axes]triple
	sun       triple
	moving    triple
	sunPlace  measure.Coord
	sunMoving measure.Velocity
}

// NewFrame is the frame about a centre on the sky so far from the Sun,
// with the Sun so high above the mid-plane and moving so, in kilometres
// per second: toward the centre, along the rotation and toward the
// pole.
//
// The z axis is the north galactic pole's, tilted about the Sun's line
// of sight to the centre by what puts the Sun at its height.
//
// It refuses a centre not on the sky, a height that reaches the
// distance, which no positive distance under it allows, and a motion
// past light.
func NewFrame(ascension, declination measure.Angle, distance,
	height measure.Length, toCentre, alongRotation, toPole float64) (*Frame,
	error) {
	centre, err := sky.NewEquatorial(ascension, declination)
	if err != nil {
		return nil, fmt.Errorf("galactocentric: the centre: %w", err)
	}
	far, up := distance.InMetres()/kiloparsec, height.InMetres()/kiloparsec
	if !(math.Abs(up) < far) {
		return nil, fmt.Errorf("galactocentric: the sun %g kpc from the"+
			" centre and %g kpc above the plane", far, up)
	}
	sunMoving, err := measure.KilometresPerSecond(toCentre, alongRotation,
		toPole)
	if err != nil {
		return nil, fmt.Errorf("galactocentric: the sun's motion: %w", err)
	}
	// the centre's direction, and the pole's with the centre's part
	// taken out, span the plane the x and z axes turn in; the y axis is
	// their normal, along the rotation at the sun
	along := toward(*centre)
	polar := toward(sky.NorthGalacticPole())
	shadow := polar.dot(along)
	risen := polar.plus(along.times(-shadow)).unit()
	tilt := math.Asin(up / far)
	sine, cosine := math.Sin(tilt), math.Cos(tilt)
	axisToCentre := along.times(cosine).plus(risen.times(sine))
	axisToPole := along.times(-sine).plus(risen.times(cosine))
	frame := &Frame{axes: [axes]triple{axisToCentre,
		axisToPole.cross(axisToCentre), axisToPole}}
	frame.sun = frame.turned(along.times(-far))
	frame.moving = triple{toCentre, alongRotation, toPole}
	frame.sunPlace, frame.sunMoving = coord(frame.sun), *sunMoving
	return frame, nil
}

// coord is a place in kiloparsecs as a coordinate from the centre.
func coord(place triple) measure.Coord {
	return measure.At(measure.Metres(place[axisX]*kiloparsec),
		measure.Metres(place[axisY]*kiloparsec),
		measure.Metres(place[axisZ]*kiloparsec))
}

// turned is a triple on the equatorial axes as the frame's axes have
// it.
func (frame Frame) turned(one triple) triple {
	return triple{frame.axes[axisX].dot(one), frame.axes[axisY].dot(one),
		frame.axes[axisZ].dot(one)}
}

// Published is the frame from the research ledger: Sgr A* where Reid
// and Brunthaler place it on the sky, as far as the GRAVITY
// collaboration measured it; the Sun as high above the mid-plane as
// Bennett and Bovy found it.
//
// The Sun moves toward the centre as Schönrich, Binney and Dehnen
// measured against the local standard of rest, and along the rotation
// and toward the pole as Sgr A*'s apparent motion has it, which is the
// Sun's own motion reflected.
func Published() (*Frame, error) {
	parsec := length.Parsec
	ascension := measure.Degrees(research.Value("sgr-a-ascension"))
	declination := measure.Degrees(research.Value("sgr-a-declination"))
	distance := research.Value("sgr-a-distance") * parsec
	height := research.Value("sun-height") * parsec
	reflected := masYearKpc * distance / kiloparsec
	alongRotation := -reflected * research.Value("sgr-a-motion-longitude")
	toPole := -reflected * research.Value("sgr-a-motion-latitude")
	return NewFrame(ascension, declination, measure.Metres(distance),
		measure.Metres(height), research.Value("sun-motion-u"),
		alongRotation, toPole)
}

// Sun is the Sun's place in the frame and its motion.
func (frame Frame) Sun() (measure.Coord, measure.Velocity) {
	return frame.sunPlace, frame.sunMoving
}

// ProperMotion is a star's motion across the sky in milliarcseconds a
// year: along increasing right ascension, times the cosine of the
// declination as Gaia gives it, and along increasing declination.
type ProperMotion struct{ ascension, declination float64 }

// NewProperMotion is a proper motion of so many milliarcseconds a year
// along ascension, times the cosine of the declination, and so many
// along declination.
func NewProperMotion(ascension, declination float64) ProperMotion {
	return ProperMotion{ascension, declination}
}

// Ascension is the motion along increasing right ascension, times the
// cosine of the declination, in milliarcseconds a year.
func (motion ProperMotion) Ascension() float64 { return motion.ascension }

// Declination is the motion along increasing declination, in
// milliarcseconds a year.
func (motion ProperMotion) Declination() float64 {
	return motion.declination
}

// Of is a star's place and motion in the frame, from where it is on the
// sky, its parallax, its proper motion and its radial velocity in
// kilometres per second, away from the Sun.
//
// It refuses a parallax that is not positive, which gives no distance,
// and a star it would have outrun light, with nil beside the error.
func (frame Frame) Of(seen sky.Equatorial, parallax measure.Angle,
	motion ProperMotion, radial float64) (*measure.Coord, *measure.Velocity,
	error) {
	mas := parallax.InArcseconds() * masInArcsecond
	if !(mas > nothing) {
		return nil, nil, fmt.Errorf("galactocentric: a parallax of %g mas"+
			" gives no distance", mas)
	}
	distance := whole / mas
	along := toward(seen)
	place := frame.turned(along.times(distance)).plus(frame.sun)
	across := masYearKpc * distance
	moving := along.times(radial).plus(east(seen).times(across *
		motion.ascension)).plus(north(seen).times(across * motion.declination))
	moving = frame.turned(moving).plus(frame.moving)
	velocity, err := measure.KilometresPerSecond(moving[axisX],
		moving[axisY], moving[axisZ])
	if err != nil {
		return nil, nil, fmt.Errorf("galactocentric: the star's motion:"+
			" %w", err)
	}
	at := coord(place)
	return &at, velocity, nil
}

// Sighting is a star as the Sun sees it: where it is on the sky, its
// parallax, its proper motion, and its radial velocity in kilometres
// per second, away from the Sun.
type Sighting struct {
	seen     sky.Equatorial
	parallax measure.Angle
	motion   ProperMotion
	radial   float64
}

// Place is where the star is on the sky.
func (sighting Sighting) Place() sky.Equatorial { return sighting.seen }

// Parallax is the star's parallax.
func (sighting Sighting) Parallax() measure.Angle {
	return sighting.parallax
}

// Motion is the star's proper motion.
func (sighting Sighting) Motion() ProperMotion { return sighting.motion }

// Radial is the star's radial velocity in kilometres per second, away
// from the Sun.
func (sighting Sighting) Radial() float64 { return sighting.radial }

// Seen is a star at a place in the frame, moving so, as the Sun sees
// it: Of turned round, so that a star made in the frame can be observed
// as Gaia would observe it.
//
// It refuses a star at the Sun's own place, with nil beside the error.
// Such a star has no direction: its direction is not a number, and the
// sky refuses that, which is the one check. A place nearer the Sun than
// a coordinate resolves, tens of kilometres there, is the Sun's place.
func (frame Frame) Seen(at measure.Coord,
	moving measure.Velocity) (*Sighting, error) {
	fromCentre, fromSun := kiloparsecsOf(at), frame.sun.times(-whole)
	place := frame.back(fromCentre.plus(fromSun))
	distance := math.Sqrt(place.dot(place))
	along := place.times(whole / distance)
	round := math.Atan2(along[axisY], along[axisX])
	ascension := math.Mod(round+turn, turn)
	declination := math.Atan2(along[axisZ],
		math.Hypot(along[axisX], along[axisY]))
	seen, err := sky.NewEquatorial(measure.Radians(ascension),
		measure.Radians(declination))
	if err != nil {
		return nil, errors.New("galactocentric: a star at the sun's place " +
			"has no direction")
	}
	againstCentre, againstSun := speedOf(moving), frame.moving.times(-whole)
	relative := frame.back(againstCentre.plus(againstSun))
	across := masYearKpc * distance
	motion := ProperMotion{ascension: relative.dot(east(*seen)) / across,
		declination: relative.dot(north(*seen)) / across}
	return &Sighting{seen: *seen,
		parallax: measure.Arcseconds(whole / distance / masInArcsecond),
		motion:   motion, radial: relative.dot(along)}, nil
}

// back is a triple on the frame's axes as the equatorial axes have it:
// turned undone.
func (frame Frame) back(one triple) triple {
	var equatorial triple
	for axis, direction := range frame.axes {
		equatorial = equatorial.plus(direction.times(one[axis]))
	}
	return equatorial
}

// kiloparsecsOf is a coordinate from the centre in kiloparsecs.
func kiloparsecsOf(place measure.Coord) triple {
	by := place.Minus(measure.Origin())
	return triple{by.AlongX().InMetres() / kiloparsec,
		by.AlongY().InMetres() / kiloparsec,
		by.AlongZ().InMetres() / kiloparsec}
}

// speedOf is a velocity in kilometres per second along each axis.
func speedOf(moving measure.Velocity) triple {
	return triple{moving.AlongX().InKilometresPerSecond(),
		moving.AlongY().InKilometresPerSecond(),
		moving.AlongZ().InKilometresPerSecond()}
}

// MovingSun is the frame with the Sun moving otherwise, in kilometres
// per second toward the centre, along the rotation and toward the pole,
// its axes and the Sun's place kept. It refuses a motion past light.
func (frame Frame) MovingSun(toCentre, alongRotation,
	toPole float64) (*Frame, error) {
	sunMoving, err := measure.KilometresPerSecond(toCentre, alongRotation,
		toPole)
	if err != nil {
		return nil, fmt.Errorf("galactocentric: the sun's motion: %w", err)
	}
	moved := frame
	moved.moving = triple{toCentre, alongRotation, toPole}
	moved.sunMoving = *sunMoving
	return &moved, nil
}
