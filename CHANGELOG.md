## v0.0.5 - v0.0.6 05 november

changed:

- radiation: ExtremeUltraviolet ends at 121 nm, as iso 21348:2007
  publishes it (6.5.3). it ended at 120, the draft's number
- radiation's named ranges, and fermi's, are read from the research
  ledger, each end cited to the page of the standard it was read on:
  the published standard's first pages to clause 6.5, its proof past
  that, and heasarc for fermi. ranges.json is gone

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
