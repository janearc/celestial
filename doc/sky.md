# Sky: where things are, seen from places

The question is simple enough: where does an object appear from here,
at this instant, in this frame? The Sun over a hillside, a star seen
through the Sun’s gravitational lens, and a merger timed at two detectors
all need some version of that answer.

This note separates the current foundations from the broader design.
`sky` is built on `measure`; ephemerides and detector waveforms are not
implemented merely because the plan below describes them.

## What is implemented

- Two-part instants, calendar conversion, and named time scales.
- UTC leap-second tables, GPS conversion, and TT/TDB conversion.
- UT1 from a supplied DUT1, and Greenwich mean sidereal time.
- Earth sites and equatorial-to-horizontal coordinates.
- Gnomonic tangent planes, east/north offsets, and error ellipses.

Calendar conversion uses the Julian calendar before 1582-10-15 and the
Gregorian calendar afterward, rejecting the skipped reform dates. Years
use astronomical numbering, including year zero. A UTC leap second is a
real extra second in the day, not a duplicate timestamp.

## Instants and scales

An `Instant` stores a whole modified Julian day and a fraction of that day.
This avoids the roughly 40-microsecond resolution limit of one contemporary
float64 Julian date. It preserves fine timing without claiming that the
underlying clock or astronomical model is equally accurate.

| Scale | Meaning |
|---|---|
| UTC | Civil time with leap seconds |
| TAI | Continuous atomic time |
| TT | TAI + 32.184 seconds |
| TDB | Barycentric dynamical time, with a small periodic difference from TT |
| GPS | TAI − 19 seconds; also used by the open science centre event times |
| UT1 | Earth rotation time, related to UTC by measured DUT1 |

The embedded leap table records TAI−UTC as 37 seconds from 2017 and limits
UTC conversion to the dates covered by its bulletin. The offset is dated
data, not a promise about all future time.

`On` converts supported scales. UT1 requires an external DUT1 measurement
through `UT1`; the library does not silently assume Earth’s rotation
matches UTC. TT/TDB uses the seven-term USNO Circular 179 approximation,
with its documented validity and accuracy.

## Sites, directions, and small fields

An Earth `Place` carries latitude, longitude, and height. Equatorial
coordinates use right ascension east along the equator and declination
north of it. `Local` converts to azimuth measured from north toward east
and altitude above the horizon, using the supplied sidereal angle.

A `Tangent` projects a small field onto the plane touching the sky sphere
at its centre. East and north are gnomonic standard coordinates, stored
as `measure.Angle` values. They approximate angular offsets only near
the centre; the projection itself uses the full geometry.

`Onto` rejects directions at or beyond a quarter-turn from the centre.
`Back` reverses the projection. `Ellipse` describes positional uncertainty
with major and minor semi-axes and a position angle east of north; its
outline is returned as offsets on that plane.

## The larger design

An object would provide barycentric ICRS position and velocity at an
instant. Observing one object from another would return a sighting:
direction, distance, and the time the light left.

Corrections would be explicit choices: light travel time, aberration,
solar deflection, and refraction at terrestrial observing sites. Frames
would then express the result in equatorial, ecliptic, galactic, or
local horizontal coordinates.

Accuracy should travel with the result as angular and distance estimates.
Initial Meeus-based models could carry his published estimates; later
JPL ephemerides could improve them without changing what a sighting means.

## Derived locations and waves

Lagrange points derive from two orbiting bodies. The Sun’s gravitational
lens derives from its mass and radius: grazing light reaches the axis at
about 548 AU, with focal distance `R²c² / (4GM)`. A target’s focal line
extends away from the Sun opposite the target direction.

For gravitational waves, the plan adds a merger’s two polarizations and
orientation, an interferometer’s site and arm directions, and the antenna
pattern. Arrival delays follow the detector separation projected along
the propagation direction, divided by c.

Rendering those signals belongs
to the application; geometry and timing belong here.

## Original implementation sequence

Each stage uses externally published answers with citations:

1. Calendar and Julian dates: Meeus examples 7.a and 7.b; time scales
   and leap seconds from IERS.
2. Sidereal time and local coordinates: Meeus 12.a, 12.b, and 13.b.
3. A low-accuracy Sun, around 0.01 degrees: Meeus 25.a.
4. The solar lens: IAU constants and a target’s focal line.

The first two stages have current foundations in `sky`. Solar and lunar
positions, planets, proper motion, Keplerian and circumbinary orbits,
Lagrange points, detectors, and JPL ephemerides remain part of the plan.

The proposed lunar reference is Meeus 47.a; Kepler-16b motivates the
circumbinary work.

Tests should compare published or independently derived expectations,
not ask the implementation to supply its own answer. Every empirical
value and chosen parameter should identify its origin.
