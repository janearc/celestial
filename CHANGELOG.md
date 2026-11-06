## v0.0.5 - v0.0.6 05 november

added:

- gpu: Apple's Metal through cgo, as slayer's metal package has it:
  kernels compiled from source when a session opens, buffers the cpu
  and the gpu share, batches of dispatches run as one command. a
  session runs on a Device, the gpu or a mock; darwin alone has the gpu
- orbit: stars stepped back through a potential by kick, drift, kick,
  in kiloparsecs and kilometres per second, to each one's nearest
  sampled pass by the galactic centre and how long ago; Kepler, a point
  mass's pull. the reference for gpu/orbit, in 64-bit floats
- orbit/orbittest: what orbit's tests and gpu/orbit's share
- gpu/orbit: the same stepping as a kernel, every star on its own
  thread in 32-bit floats, held to orbit's reference by its tests to a
  thousandth of a kiloparsec, and a thousandth more for each kiloparsec
  of the pass, where the steps resolve the orbit
- gpu/orbit is held on passes from a tenth of a kiloparsec out and on
  a circle a parsec from the centre. a plunge from kiloparsecs to
  parsecs is not tested yet; it wants a galactic potential and steps
  planned for the pass. the steps a command takes are in the ledger
- orbit, gpu/orbit: a pass carries the step's shares of it, taken at
  the step it was found at: how far the star moved in one step and how
  far the pull moved it over one step squared, each over the pass's
  distance (Pass.Steps; NewPass takes four)
- both shares small means the steps resolved the pass; a plunge the
  steps missed has large ones. the kernel is held to the reference on
  them too, at the kernel's own step. orbittest.Shares is their closed
  form for a point mass; orbittest.Pericentre gives its speed as well
- gpu/orbit: a batch in which stars ran away, through the centre or
  past what a float holds, is refused as a Runaway that names every
  one of them by its place in the batch, so a caller can drop them and
  run the rest once
- RanAway reads the stars from a refusal, and NewRunaway makes one for
  a solver that stands in for the kernel. any other refusal of Back is
  still an error with no stars named
- galactocentric: Seen turns the frame round, from a star's place and
  motion about the centre to where the Sun sees it, at what parallax,
  and moving how across the sky and along the line of sight
- Seen is held by the hand on Sgr A* at rest, seen where Reid and
  Brunthaler see it, and by round trips through Of either way. a star
  at the Sun's own place has no direction and is refused
- orbit: the Milky Way as MWPotential2014 has it (Bovy 2015, table 1):
  a bulge with a power-law density cut off exponentially, a
  Miyamoto-Nagai disc and a Navarro-Frenk-White halo, each normalised
  by its share of the force at the model's sun
- Sgr A* is added to it as a point mass, which the paper's model
  leaves out; its pull at the sun is a part in twenty thousand
- the model is a Model, built from the sun's circle with each part
  added; MWPotential2014 reads it from the research ledger. each part
  has its Pull and its Well, and so have the galaxy and a Kepler
- the tests hold the galaxy to the paper's table 1 (the masses, the
  densities and the vertical force at the sun, the rotation curve's
  slope, the escape speed) and each part's density to its definition
- gpu/orbit: the galaxy's pull in Metal, gpu/orbit.MilkyWay, with the
  gamma function summed in 32 bits; Solver.Pulls, the field's pull at
  each place by the kernel, for holding the kernel's pull to the
  reference's and for asking the field
- the agreement property runs over the galaxy's field too, gated by
  the reference pass's own step shares (orbittest.Room), so it serves
  any field; and a parity property holds the kernel's pull to the
  reference's at any place in both fields
- galactocentric: a star as Gaia sees it (its place on the sky,
  parallax, proper motion and radial velocity) to its place and motion
  about Sgr A*, as astropy's Galactocentric frame has it
- the frame: the centre at Sgr A*, the z axis the galactic pole's
  tilted to put the Sun at its measured height, the y axis along the
  rotation at the Sun; Of gives measure's Coord and Velocity
- the published frame reads the ledger: Sgr A*'s place (Reid and
  Brunthaler, two new entries) and distance (GRAVITY), the Sun's height
  (Bennett and Bovy) and its motion toward the centre (Schönrich,
  Binney and Dehnen)
- the Sun's motion along the rotation and toward the pole is Sgr A*'s
  apparent motion reflected (Reid and Brunthaler), 251.5 and 8.6 km/s
- the seam between the frame and orbit's Milky Way is open and said in
  both docs and the ledger: the frame's Sun goes 251.5 km/s along the
  rotation, the model's at 8.277 kpc about 231
- every star the frame places carries the 20 km/s until the model is
  placed at the frame's Sun, or the frame's Sun given the model's speed
- galactocentric: a ProperMotion gives back its Ascension and
  Declination, for a reader to hand on and a test to see
- galactocentric: Frame.MovingSun, the frame with the Sun moving
  otherwise, its axes and the Sun's place kept: how a caller gives
  the frame's Sun the model's speed, the seam's other closing
- sky: NorthGalacticPole, the pole's direction on the equatorial frame
- fits: binary tables. OpenTable is the first BINTABLE in a file, past
  the HDUs that hold none; its Columns by name with their units, its
  Rows read as asked for, a few thousand at a time
- a cell is read as a Float (scaled by TSCAL and TZERO; a null, NaN or
  TNULL, is not a number), an Int, a Logical or Text. columns of one
  value, of the kinds L, B, I, J, K, E, D and A; the rest is refused
- a TSCAL or TZERO that is not a number is refused; a TNULL is read
  whole (Header.Integer, new), and one on a float column is ignored, as
  archives write them; a row the file cannot give is refused, and a
  file that says it ends with the last row's bytes has given them
- a TSCAL of none is read as no scaling: astrogo/fitsio writes TSCALn
  and TZEROn of none on every column, meaning neither was set, and no
  writer means a scale that zeroes every value

changed:

- radiation: ExtremeUltraviolet ends at 121 nm, as iso 21348:2007
  publishes it (6.5.3). it ended at 120, the draft's number
- radiation's named ranges, and fermi's, are read from the research
  ledger, each end cited to the page of the standard it was read on:
  the published standard's first pages to clause 6.5, its proof past
  that, and heasarc for fermi. ranges.json is gone
- fits: a table with columns of many values a row opens. they are
  stepped over, the columns around them read, and a read of one is
  refused, saying how many values it holds (Column.Count). such a
  table was refused whole: APOGEE's allStar has them

- added celestial-data member

## v0.0.4 1 nov

- whole bunch of doc changes for clarity

breaking:

- atmosphere: Depth, Through and DepthAbove return (*float64, error),
  nil beside an error, where they returned nothing beside one
- gone: radiation's ErrRange and ErrWavelength, atmosphere's
  ErrHeight, ErrWavelength and ErrWay. nothing outside their own tests
  read them; each refusal is a plain error naming what was refused
- gone: hdf5's ErrNotHDF5, ErrUnsupported, ErrCorrupt, ErrNotFound and
  ErrWrongType. nothing outside hdf5 read them. each refusal says its
  cause in words, with the address and the section of the
  specification, as before

fixed:

- measure: NewDirection, and atmosphere's NewWay, refused a direction
  of parts too large or too small to square, such as (1e200, 0, 0);
  they scale by the largest part first
- measure: KilometresPerSecond took a velocity of exactly light's
  speed, which as a share of light's rounded to just under one; it is
  held to light's speed in km/s, as given
- atmosphere: Along from the sea pointing down went on through the
  earth to the sky beyond, about 140 times the air straight up; it is
  on the ground at once
- hdf5: a dataspace of more than 32 dimensions was refused as damaged,
  by a limit said to be the specification's. the specification sets
  none; 32 is the most the hdf5 library writes. it is refused now as
  not read, ErrUnsupported, saying so

## v0.0.3 31 oct

refusals check the ranges the physics needs, and say so in plain errors.

breaking:

- hole: NewHole and NewGPUBlackHole return an error beside a nil hole.
  the heaviest hole is about 5.8e76, the most whose own horizon radius
  can place; past it, positions near the horizon overflow
- stars: a star's Stage is its Life
- sky: GPSSeconds returns (*float64, error)
- gone: ErrAbsent, ErrBand, ErrBehind, ErrBright, ErrDate, ErrEllipse,
  ErrEvents, ErrMagnitude, ErrPlace, ErrScale and ErrUnit. nothing read
  them; each refusal is a plain error naming what was refused

new:

- radiation: KeVToNm, MeVToNm, GeVToNm, TeVToNm, MHzToNm, GHzToNm and
  MicronsToNm
- research data: highest-place-m, the top of the 1976 standard
  atmosphere, 1000 km

fixed:

- photometry refuses a magnitude too faint for a float, which gave a
  flux of nothing
- sky refuses a place below the earth's centre or above 1000 km, an
  ascension or position angle past a turn, an ellipse wider than the
  sky, and a julian fraction that cannot be counted

## v0.0.2 31 oct

begin flag day for libtheme-css

## v0.0.1 31 oct

initial release. folds in stuff from:

- libholemath
- libtheme-css
- slayer (unreleased)
- nilgai
