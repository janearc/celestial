# sky: where things are, seen from places

every use of this package asks one question. where is a thing, seen
from somewhere, at some instant, in some frame?

the sun over a hillside at dusk asks it. so does pointing at kepler-16b
through the sun's gravitational lens from beyond pluto. so does a
neutron star merger whose waves reach two detectors milliseconds apart.

these are one question with different arguments. the package is built
around it.

it is built on measure.

## the parts

**an instant** is a moment, held as a julian date in two parts: a whole
day and a fraction of it. one float64 date resolves about forty
microseconds today. two parts resolve far below a nanosecond, and
timing a wave between detectors needs that.

**a scale** says whose clock an instant is on:

| scale | what it is |
|---|---|
| utc | civil time. it gains a leap second when the earth falls behind |
| tai | atomic time. it never leaps; tai less utc is 37 s since 2017 |
| tt | the clock the ephemerides run on. it is tai plus 32.184 s exactly |
| tdb | tt as the solar system's centre keeps it. it differs by under 2 ms |
| gps | the satellites' clock, and LIGO's and Virgo's. it is tai less 19 s exactly |
| ut1 | the earth's own turning, which iers measures |

**a thing** is anything that can say where it is at an instant. the
sun, the moon, planets and stars are things. so are a body on an orbit,
a fixed point, a place on earth and a spacecraft.

a thing answers with its place and velocity from the solar system's
barycentre, on the icrs axes.

**seeing** is one thing from another at an instant. the answer is a
sighting: a direction, a distance and when the light left.

each correction is named, and the caller says which are on: light
time, aberration, the sun's bending of light and refraction. only a
place on earth has refraction.

**a frame** turns a sighting into two angles and a distance. the frames
are the sky's (right ascension and declination), the ecliptic, the
galaxy's, and the local sky at a place (azimuth from north and
altitude).

**accuracy** rides with every answer. every place, velocity and
sighting says how far off it can be, as an angle and a length. the
first answers carry meeus's figures. later ones are better.

## places made from things

some places are worked out from things, not looked up:

- a lagrange point of any two things on their orbit.
- the sun's gravitational lens. light grazing the sun bends toward the
  line through its middle. it meets that line at least 548 AU out. the lens
  for a target is the line away from the sun opposite it. that distance
  is the sun's radius squared times c squared, over four times its GM.

## gravitational waves

waves and light share the sky. a merger's light and its waves come from
one thing, and sky gives both one place and one distance.

- **a merger** is a thing with a waveform: two polarisations over time
  and the angle they are turned by.
- **a detector** is an interferometer at a place with two arms along
  the ground. its answer to a wave from a direction is the antenna
  pattern, worked out in closed form.
- **arrival** differs from detector to detector. the delay is how far
  apart they stand along the wave's way, over c. that delay is how
  detectors find a merger on the sky.

this is what the package must hold. it is built after the sun and the
moon. rendering the waves is the tracer's work. this package tells it
where and when.

## built first, in order

each step has tests from published answers, and each test names its
source:

1. instants: calendar dates to julian dates and back, held to meeus's
   examples 7.a and 7.b. the scales follow, with the leap seconds as
   iers publishes them.
2. sidereal time (meeus 12.a and 12.b), places on earth, and the local
   sky from the sky's frame (meeus 13.b).
3. the sun at low accuracy, to about 0.01 degrees (meeus 25.a).
4. the sun's lens: its focal distance from the iau's constants, and its
   focal line for a target.

after those, behind the same doors: the moon (meeus 47.a), the planets,
stars with their own motion, kepler orbits, kepler-16b round its two
stars, lagrange points, detectors, and jpl's ephemerides for when meeus
is not enough.

## rules

- tests hold published answers, never this package's own.
- every number that is not nature's says where it came from.
