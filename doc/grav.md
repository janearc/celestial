# grav: bodies, and the gravity between them

the gravBody of the engineering guide, where saturn is the bad example
and gravBody the good one. a planet, a brown dwarf, a star, a white
dwarf, a neutron star and a hole are all one kind of thing here.

what each is follows from its mass, size, makeup and age, and grav
says only the part gravity decides. written 2026-10-05; nothing built
yet.

## what it stands on

- measure: every quantity, in MeV with c, the reduced planck constant
  and boltzmann's constant one, and newton's G as measured. places are
  measure.Coord.
- materials: what a body is made of, which grav asks only one thing:
  its mass.

## matter

```go
type Matter interface {
	Mass() measure.Mass
}
```
for stars and planets: you need a materials. grav doesn't really care about
solid or gas, that's materials' job.

for holes you need a bare `mesaure.Mass`.

## spin

```go
func Turning(axis measure.Direction, day measure.Duration) (Spin, error)
func Momentum(axis measure.Direction,
	amount measure.AngularMomentum) (Spin, error)
```

## bodies and holes

```go
func NewBody(matter Matter, age measure.Duration, spin Spin) (Body, error)
func NewHole(mass measure.Mass, spin Spin) (Body, error)
```

error checking happens during construction. holes have nonzero spin. bodies
have nonzero mass (and non-infinite). 

you don't specify a radius because gravity figures that out for you: if you
have a material, and a certain amount of mass, and a certain amount of angular
momentum, quelle surprise, the math did it for you.

see also:
- baraffe and colleagues' tracks run from half a jupiter to beyond the
  sun's mass.
- mist, choi and colleagues 2016, runs from a tenth of a sun
  to three hundred suns, through giants and white dwarfs.

## the constructor

```go
func NewBody(matter Matter, age measure.Duration, spin Spin) (Body, error) {
	if matter == nil || !spin.Made() {
		return nil, refuse("a body needs matter, and a spin " +
			"made by Turning or Momentum")
	}
	mass := matter.Mass()
	if !mass.Positive() {
		return nil, refuse("a body of mass %v", mass)
	}
	if !age.Within(measure.Cosmos()) {
		return nil, refuse("a body %v old", age)
	}
	radius, err := radiusOf(matter, age)
	if err != nil {
		return nil, err
	}
	if radius.Over(horizon(mass)) <= 1 {
		return nil, refuse("inside its own horizon, this body " +
			"would be a hole already; holes come from NewHole")
	}
	return newBody(matter, age, spin, radius), nil
}
```

## body methods

```go
func (body Body) Mass() measure.Mass
func (body Body) Radius() measure.Length
func (body Body) Age() measure.Duration
func (body Body) Spin() Spin
func (body Body) Density() measure.Density
func (body Body) GM() measure.Length
func (body Body) Horizon() measure.Length
func (body Body) Compactness() Compactness
func (body Body) EscapeSpeed() measure.LightFraction
func (body Body) SurfaceGravity() measure.Acceleration
```

GM is a length in natural units: half the horizon radius of a hole of
that mass. Horizon is twice it for a still hole; a spinning hole's is
smaller, and comes from holes, not here.

Compactness is GM over the radius: a length over a length, with no
unit. the sun's is two in a million, a neutron star's about a sixth,
and a hole's a half.

EscapeSpeed is a share of light's speed, which in natural units is
what a speed is. its type says so: measure.LightFraction.

## bodies interacting

    func Pull(one, other Body, apart measure.Length) measure.Force
    func Hill(small, large Body, orbit Orbit) measure.Length
    func Roche(small, large Body) measure.Length
    func Tidal(star, hole Body) measure.Length

Hill is how far from the small body its own moons can stay its own.
Roche is how close the small body can come before tides tear it
apart.

Tidal is the same question for a star near a hole. If that radius is
outside the hole's horizon, the star is torn apart where it can be
seen. If not, the hole swallows it whole, and that answer comes from
the masses alone.

## orbits

an orbit is two bodies and a shape. a shape is made of parts, and each
part refuses what it can't be where it is made:

```go
func NewEccentricity(value float64) (Eccentricity, error)
func NewTilt(angle measure.Angle) (Tilt, error)
func NewNode(angle measure.Angle) (Node, error)
func NewPeriapsis(angle measure.Angle) (Periapsis, error)

func NewShape(apart measure.Length, eccentricity Eccentricity,
	tilt Tilt, node Node, periapsis Periapsis,
	since measure.Duration) (Shape, error)

func NewOrbit(primary, secondary Body, shape Shape) (Orbit, error)
func (orbit Orbit) Period() measure.Duration
func (orbit Orbit) At(when measure.Duration) Places
```

the three angles are three types, so they can't be passed in the
wrong order. what each part refuses:

- apart, the semi-major axis: anything not above nought.
- eccentricity: below nought, or one or more. open orbits, parabolas
  and hyperbolas, wait for a picture with a passing comet.
- tilt: outside nought to half a turn. a tilt of 200 degrees is the
  160 degree orbit with its node swung round, and one orbit gets one
  spelling.
- node and periapsis: any finite angle, folded into one turn.
- since, how long after periapsis the orbit stands at time nought: any
  finite time, negative too. NewOrbit folds it by the period, which
  only it knows.

every part refuses a number that is not a number, or is endless.

the period is kepler's third law, from both masses. At turns a time
into a place by kepler's equation, solved by newton's method to the
precision of a float, with a bound on its steps. it says so if it does
not converge, rather than handing back a guess.

Places answers, through its methods, where each body is round the
barycentre and how it moves. the orbit's plane is the kit's: with no
tilt, its north is y.

## three bodies, sketched

how a caller makes a star, a planet of one rock, and a giant of
nothing but hydrogen, all through one door. a sketch: none of it is
built yet. each err is checked; the checks are left out to keep the
sketch short.

```go
// a ball of hydrogen and helium at the sun's mass. nothing here says
// it is a star: its mass does.
solar, err := materials.SolarAbundances()
sunMatter, err := materials.NewAmount(solar, measure.SolarMasses(1))
sunSpin, err := grav.Turning(measure.Up(), measure.Days(25.4))
sun, err := grav.NewBody(sunMatter, measure.Years(4.6e9), sunSpin)

// the same door, a thousandth of the mass, and only hydrogen
hydrogen, err := materials.NewElements(map[string]float64{"H": 1})
gas, err := materials.NewAmount(hydrogen, measure.Kilograms(1.898e27))
giantSpin, err := grav.Turning(measure.Up(), measure.Hours(9.93))
giant, err := grav.NewBody(gas, measure.Years(4.6e9), giantSpin)

// a planet of one rock, all of it olivine at its bulk density
formula, err := materials.NewFormula(map[string]float64{
	"Mg": 2, "Si": 1, "O": 4})
olivine, err := materials.NewMaterial("forsterite", formula,
	measure.GramsPerCC(3.27))
rock, err := materials.NewAmount(materials.Only(olivine),
	measure.Kilograms(5.97e24))
rockSpin, err := grav.Turning(measure.Up(), measure.Days(1))
planet, err := grav.NewBody(rock, measure.Years(4.6e9), rockSpin)
```

- the rock planet comes out about 7,600 km in radius, a fifth wider
  than the earth's 6,371. that is right for what was asked: the
  earth has an iron core, and its rock is squeezed by its own weight.
- one density, uncompressed, is the first cut, and the gap is honest.

## the giants

no door of their own. a giant is a body whose matter is mostly gas,
and the door works out the rest. what tells saturn from uranus is
their matter, and that comes from published interior models, which
still disagree, so each names the model it takes.

```go
saturn, err := grav.NewBody(materials.Planet("saturn"),
	measure.Years(4.6e9), saturnSpin)
uranus, err := grav.NewBody(materials.Planet("uranus"),
	measure.Years(4.6e9), uranusSpin)
```

roughly what those tables hold, to know what to expect:

- saturn: hydrogen and helium nearly all through, with a diffuse core
  of rock and ice, perhaps a tenth to a fifth of its mass. the
  diffuse core is from ring seismology, mankovich and fuller 2021.
- uranus and neptune: mostly ices, water, ammonia and methane, about
  three quarters of the mass; a rocky core; and an envelope of
  hydrogen and helium of only a tenth or so. helled and colleagues
  2011 is one model of several.

so a jupiter, a saturn and an ice giant differ only in their shares,
and the same door makes all three.

## bounds

every bound the door applies, and where it comes from. none is a
number written by hand:

| bound | refused | from |
|---|---|---|
| mass | not above nought, or not finite | the number |
| age | below nought, or past the universe's | planck's age |
| spin | a day shorter than breakup | the density |
| horizon | a radius inside the body's own horizon | 2GM; to NewHole |
| gas floor | a gas world that lost its gas over its age | jeans escape |
| tracks | a gas body outside every published track | the tracks |
| shares | matter whose shares don't sum to one | materials |
| layers | given layers not ordered by density | the densities |

breakup is the shortest day a body held only by its own gravity can
have: the square root of three pi over G times its density. it depends
on density alone, not on size.

a body past every track is refused rather than guessed at. half a
jupiter is the lightest gas ball baraffe's tracks reach, and three
hundred suns the heaviest mist does; between them the door answers,
outside them it says there is no published answer.

## fusion is not written down

nothing in the code says "past this mass, it burns". in the tracks, a
ball past about 0.075 suns settles into burning hydrogen steadily, and
its light stops falling. a ball below it never does, and cools for
ever.

between about thirteen and eighty jupiters it burns deuterium for a
while first: a brown dwarf. so planet, brown dwarf and star are what a
body is called, found from the tracks, never a door to choose.

what a ball of gas does in its core is a matter of its mass alone, in
a vacuum. the small things that change it come from outside or from
turning:

- heating from a nearby star puffs a giant up. that is two bodies,
  so it waits for systems.
- spin flattens a body and lifts its equator a little. it waits for a
  density profile.

## the smallest gas world

a ball of gas has a floor as well as a ceiling. too little mass and it
can't hold its gas: the fastest molecules leave, and over the body's
age the envelope is gone and what is left is rock.

this one is grav's, mostly: how fast a molecule must go to leave is
the body's escape speed. how fast molecules go is their temperature
over their mass, from measure and materials. so the floor depends on
the gas and on how warm it is, not on one size:

- hydrogen, being light, leaves first. the earth keeps nitrogen and
  oxygen but lost its hydrogen.
- titan is small and keeps a thick nitrogen sky, because it is cold
  and nitrogen is heavy.
- in a vacuum a small ball of hydrogen would freeze solid rather than
  boil away, so heating from outside decides a great deal.

the door works it out from jeans' escape: how much leaves in a year,
from the escape speed and the thermal speeds, against the body's age.
no rule of thumb for how many times the escape speed is enough; the
rate and the age say.

weird ones to test it against: the super-puffs, like kepler 51's
planets, far too light for their size. young and still shrinking,
hazy, or ringed are the readings on offer.

and the hycean worlds, oceans under hydrogen, argued over around
k2-18b and not settled either way. those need layers, so they wait
for density profiles.

## what the tracks settle

- the mist tracks give a burning star's lifetime by mass. stars'
  burnedShare, the one modelled number left, goes, and with it the
  heavy stars that die four times too young.
- they carry giants and white dwarfs, so ended stops being a dead end.
- a hole is the one body the door can't make. a hole comes from a
  history, not a structure, so it keeps NewHole(mass, spin).

## not in grav

- light, and burning: emit and stars.
- the strong field near a hole: holes, the contract in front of
  libholemath.
- orbits that shrink as gravitational waves carry energy away:
  peters' formula, a layer on top of Orbit, for the two holes.
- three bodies or more: systems, a later package.
- tides that change a body's shape: later, when a picture asks.

## measure gains

each a type like the others, with private insides:

- Acceleration, Force, Angle and Density.
- Direction, a unit vector.
- Coord and Displacement, below.
- and measure.Sun's measures become methods, Sun().Mass() and the
  rest, by the rule above.

## what has no unit

with c, the reduced planck constant and boltzmann's constant one, more
things have no unit than anyone expects, so each type says on its face
what it is a ratio of:

| type | a ratio of |
|---|---|
| Eccentricity | a length over a length: focus from centre, over apart |
| Compactness | a length over a length: GM over the radius |
| LightFraction | a speed over light's speed |
| AngularMomentum | a count of the reduced planck constant |
| Direction | a unit vector: lengths over their own length |

each is still typed, so no two of them can be swapped.

## places

measure.Coord with x, y and z, each a position.
a place is not a length, so a position is its own type, and the
arithmetic says what places are:

  - a Coord less a Coord is a Displacement, three Lengths.
  - a Coord plus a Displacement is a Coord.
  - a Coord plus a Coord does not compile.

what a Coord can't say alone is which frame it is in: the sun's, the
hole's, the camera's. a location package on top of measure keeps
frames, when there are frames to keep.

## open

- roche's limit for a fluid body comes from roche's own numerical
  solution, not a formula. for a rigid body it is exact. the fluid
  number is cited where it is used.
