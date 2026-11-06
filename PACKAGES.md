# packages

celestial combines the math and physics packages from software i've been using
into one place.

- [slayer](https://github.com/janearc/slayer) - my ray tracing and rendering
  engine, which was the rewrite of [miami](https://github.com/janearc/miami).
- [nilgai](https://github.com/janearc/nilgai) - my first virtual observatory
- [libtheme](https://github.com/janearc/libtheme-css) - a very normal color
  library that uses css as a transit layer and managed full-spectrum radiation
  in addition to, you know, visible light.
- [holemath](https://github.com/janearc/libholemath) - a self-contained
  library i've been using for other software that needs to have realistic
  maths and physics for very large gravitational bodies. black holes, mostly,
  but also neutron stars and geodesics and so on.

all of these had a little bit of overlap, all of them were struggling at the
seams, and so it made the most sense to really just combine the packages into
one place with one api.

i also noticed when i was looking around that for about ten years, golang
hasn't had this functionality. it seems to be because everyone went and moved
to astropy. i am not ashamed to say that i just don't like python. i actually
dislike python so much i have an agent that i have *solely for porting
anything i find in python to golang,* and she role-plays as a priestess of
rene, *he of buck't tooth, short of tail, and small of ears,* to *rid this
machine of the belly-crawling slithering degeneracy.* so it's fair to say that
i wanted this functionality in golang. plus, i am not an astronomer, i don't
think, so i'm not obliged to use astropy if i don't want to.

slayer is interesting; i built slayer because i realized i was doing ray
tracing, and this let me build synthetic telescopes. i started by simultaing
hubble and keyhole, and then i realized i could actually build any kind of
simulated observatory i wanted. it kind of bums me out that these ideas i have
for observatories (you know, i just want to park a telescope out at 550 AU and
see what i can see!) and i know that i won't live long enough to see any of
them come to be. *however,* i know that if my math is good enough, i can dream
up observatories, and see what is possible. i guess some people have other
hobbies like road biking and schutzhund and... nevermind, i guess i skipped
ahead from "i can render images of jupiter and quasars" to "well if the math
is that good why can't i design observatories" and skipped all the video game
stuff. sometimes it's like that.

this lil document has the breakdown of what each of these packages does, and
`go doc` should tell you the rest.

## if you know astropy

- `astropy.units` is measure: a typed quantity per kind, in si at the
  surface
- `astropy.constants` is si, length and research: every number with
  its source
- `astropy.time` is sky's Instant and its scales
- `astropy.coordinates` is sky's Equatorial, Galactic, Place,
  Horizontal and Tangent
- the black body model is radiation's Planck
- `astropy.io.fits` and `astropy.wcs` are fits, for two-dimensional
  images and their TAN and CAR grids, and for binary tables of one
  value a column, read a few thousand rows at a time
- galpy's orbit integration, which astropy leaves to it, is orbit:
  stars run back through a potential to their nearest pass by the
  galactic centre, each pass carrying the step's shares of it, so a
  caller can tell a resolved pass from a plunge the steps missed.
  gpu/orbit does every star at once on the gpu. the potentials are a
  point mass and galpy's MWPotential2014 (Bovy 2015) with Sgr A* added,
  each part with its pull and its well
- astropy's Galactocentric frame is galactocentric: a star as Gaia
  sees it, by its place on the sky, parallax, proper motion and radial
  velocity, to its place and motion about Sgr A*, with the Sun's place
  and motion from the research ledger

planned, not built:
- the ecliptic frame
- precession and nutation
- ut1 from iers' earth orientation

## underneath

### si

- `si.Planck`, `si.Speed`, `si.Charge`, `si.Boltzmann`, `si.Avogadro`
- `si.Gravitation()`: G, which is measured, so it is read from research
- `si.Kilo`, `si.Mega`, `si.Giga`, `si.Tera`, `si.Centi`, `si.Milli`,
  `si.Micro`, `si.Nano`: a prefix is a function of the unit

### length

- `length.Metre`, `length.Angstrom`, `length.AU`, `length.Parsec`,
  `length.LightYear`
- `length.PlanckLength()`
- multiples from si: `si.Kilo(length.Parsec)` is a kiloparsec etc

### research

every number in celestial is cited. these citations are in 
`research/research-derived-data.yml`, each naming its source in
`research/sources.yaml`. there is `game` target to pull this data fresh and do
the extract, please have a look at the `.game` file for instructions on how to
do that. it's a little complicated and i'm deliberately not explaining that
here.

- `research.Value(name)`: value for a given name
- `research.Read()`: all named, cited values

## quantities

### measure

every kind of quantity has its own type, so a mass can't go where an age
goes. you make one from the unit it arrives in and read it in the unit you
want. inside it works in natural units, where c and the reduced planck
and boltzmann constants are one. outside it's all si.

- make one: `measure.Kilograms`, `measure.SolarMasses`,
  `measure.Metres`, `measure.Seconds`, `measure.Years`,
  `measure.Kelvin`, `measure.Watts`, `measure.Degrees` and the rest
- read one: `Mass.InKilograms`, `Length.InMetres`, `Angle.InRadians`
  and the rest
- measured: `measure.Sun()`, `measure.Electron()`, `measure.Proton()`,
  `measure.Hydrogen1()`, `measure.Helium4()`
- laws: `measure.GM`, `measure.Eddington`, `measure.BlackBody`,
  `measure.SurfaceTemperature`, `measure.Photon`
- space: `measure.At` (a point), `measure.Offset` (a displacement),
  `measure.NewDirection`, `measure.KilometresPerSecond` (a velocity)

## light

### radiation

light by its wavelength in nanometres, from gamma rays to radio. what it
looks like is a colour, and colour is libtheme's.

- `radiation.Planck(nm, kelvin)`: a black body's spectral radiance, in
  W m^-2 sr^-1 nm^-1
- to and from a wavelength: `radiation.Energy` and
  `radiation.FromEnergy` (electronvolts), `radiation.NmToHertz` and
  `radiation.HertzToNm`, `radiation.NmToMetres` and
  `radiation.MetresToNm`
- named units: `radiation.KeVToNm`, `MeVToNm`, `GeVToNm`, `TeVToNm`,
  `MHzToNm`, `GHzToNm`, `MicronsToNm`
- `radiation.JanskyToSI`: a flux density in si
- ranges: `radiation.NewRange(short, long)`, and the named ones,
  `radiation.Gamma()`, `XRay()`, `ExtremeUltraviolet()`,
  `Ultraviolet()`, `Infrared()`, `Microwave()`, `Radio()` and
  `Fermi()`
- false colour: `radiation.NewProfile(name, source, render)` maps one
  range onto another, with `Profile.Onto` and `Profile.Back`

### bands

light as amounts in slices of the spectrum, for when it's added,
filtered, redshifted or bent on its way to an eye.

- make one: `bands.Lines(spectrum)` from wavelengths and powers,
  `bands.Split(amounts)` from amounts by band
- `bands.Add` adds two lights; `bands.Lit(light, share)` passes a share
  of one, made with `bands.Even` or `bands.Shares`; `Light.Shift(steps)`
  redshifts it
- read it: `Light.Power(band)`
- `bands.Of(nm)` is the band a wavelength falls in, and `Band.Edges`
  its ends

### photometry

magnitudes and the fluxes they stand for, from a survey's own zero
points on pogson's scale.

- `photometry.WISE()`: the survey and its four bands
- `Survey.Flux(band, magnitude, error)`: the flux density and its
  error, with `Flux.InSI` and `Flux.InJanskys`
- `Photometer`: an interface, so a catalogue reader's tests can use the
  generated mock

## where and when

### sky

- an instant: `sky.Calendar(...)`, `sky.FromJulian`, `sky.FromGPS`
- its scale: `sky.UTC()`, `TAI()`, `TT()`, `TDB()`, `UT1()`, `GPS()`;
  `Instant.On(scale)` moves it to another, with utc's leap seconds from
  the iers table
- read it: `Instant.Calendar`, `Instant.Julian`, `Instant.GPSSeconds`,
  and on ut1, `Instant.MeanSidereal`; `Instant.UT1(dut1)` puts utc on
  ut1
- a place and a direction: `sky.OnEarth(latitude, longitude, height)`
  and `sky.NewEquatorial(ascension, declination)`; `Place.Local` is
  where a direction stands in that place's sky
- the galaxy's frame: `sky.NewGalactic(longitude, latitude)`, with
  `Equatorial.Galactic` and `Galactic.Equatorial`, by the hipparcos
  definition
- a small field: `sky.NewTangent(centre)`, with `Tangent.Onto` and
  `Tangent.Back`; `Projector` is the interface, so tests can use the
  generated mock
- a position's error: `sky.NewEllipse(major, minor, angle)`, and
  `Ellipse.Outline` draws it on the plane

### atmosphere

- `atmosphere.Standard(height)`: the 1976 standard air there, with its
  `Temperature`, `Pressure` and `Density`
- `atmosphere.Along(height, rise)`: the column of air along a look, its
  `Molecules`, and whether it's `Grounded`
- `atmosphere.DryAir(co2)`, co2 in parts a million, and rayleigh on it:
  `CrossSection`, `Depth` and `Through` along a column, `DepthAbove`
  from a pressure, and `Skylight`, the sky's radiance down a look with
  the sun another way
- a way to look, or the sun's: `atmosphere.NewWay(x, y, z)`

## bodies

### stars

- `stars.NewStar(mass, age)`: a main-sequence star from pecaut and
  mamajek's table, with its `Temperature`, `Luminosity`, `Radius` and
  `Life`, burning or ended

### grav

- `grav.Point(mass)`: a body known only by its mass, which is all an
  orbit needs, with its `GM`. what's planned is in `doc/grav.md`

### hole

a kerr hole for a renderer, in kerr-schild form. G and c are one, and
lengths are in the hole's mass. it imports only the standard library.

note: some of hole came over from slayer and it inhabits a kind of in-between
place where it's a little bit ray tracing and a little bit gravity. this
interface is going to change a bit in the near future. 

- `hole.NewHole(mass, spin)`
- the hole: `Hole.Horizon`, `PhotonSphere`, `PhotonOrbit`, `Shadow`,
  `ISCO`, `Radius`, `Metric`, `Inverse`, `Lower`, `Raise`
- vectors: `hole.NewVec3`, `hole.NewVec4` and `hole.Dot`
- a ray: `hole.NewRay(position, momentum)`, or from an observer:
  `Hole.Null` leaving a point, `Hole.Arriving` coming in, `Hole.Dropped`
  and `Hole.Falling` for a body let go from rest
- `Hole.Step(ray, size)` steps a ray, and `Hole.Seen` is where its light
  looks to come from
- what a geodesic keeps: `Energy`, `Lz`, `Carter`, `Hamiltonian`
- observers: `Hole.Static`, `Hole.Circular`, `Hole.Normal`,
  `Hole.FrameOf`; and `hole.Shift`, the redshift between two of them
- moving to slayer, so don't build on these: `March`, `PaceIn`,
  `Hovering`, `ClosestApproach`, `DeflectionAngle`, `Advance`,
  `ClassifyRay`, `Fate`, `choices.json` and all of `gpu.go`

## what detectors publish

### gw

gravitational wave events as the open science centre publishes them, in
si.

- `gw.ReadEvents(json)`: every event in the centre's event json
- `Event.Estimate(name)` by the centre's name for the parameter, with
  `Estimate.Value`, `Unit` and `Interval`; `Event.At` is its instant and
  `Event.Catalog` its catalogue
- `Estimator`: an interface, so tests can use the generated mock

### hdf5

the parts of hdf5 the centre's strain files use, and nothing else.

- `hdf5.NewFile(reader, size)`, then `File.Members(path)`,
  `File.Read(path)` and `File.Attribute(path, name)`
- `Values.Floats`, `Integers`, `Texts` and `Shape`

### fits

two-dimensional fits images, and where their pixels lie on the sky.

- `fits.Open(file)`: the first image in a file, with its `Width`,
  `Height`, `Header` and `Row(y)`, scaled and with `BLANK` as not a
  number
- `fits.ReadHeader(reader)`, then `Header.Has`, `Text` and `Number`
- `Image.Sky()` or `fits.SkyOf(header)`: its grid, TAN or CAR, in icrs
  or galactic; `Sky.Place(x, y)` and `Sky.Pixel(lon, lat)`

### mocks

generated by `game run mocks`: `MockPhotometer`, `MockProjector` and
`MockEstimator`.

## grissom

the sky as layers on a mesh of triangles, shown together.

- `mesh`: an icosahedron's faces split in four as deep as asked, to
  `mesh.Deepest`. `mesh.Locate(place, depth)` is the cell a place is
  in; a `Cell` has `Corners`, `Centre`, `Area`, `Children`, `Parent`
  and a `Key`, which `mesh.FromKey` reads back
- `layer`: `layer.Lay(name, image, depth, source)` lays a fits image
  onto the cells at a depth, each the mean of the pixels in it, in icrs.
  `Layer.At(cell)`, and `Layer.Source`, where the image came from
- `composite`: a `Stack` of layers, each stretched, `Shape`d linear or
  logarithmic, turned on or off and weighed by a `Strength`. `Relate`
  sets one layer `Beside`, `Occludes` or `Masks` another; `Stack.At` is
  each lit layer's level at a cell
