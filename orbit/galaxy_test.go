package orbit_test

import (
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/orbit"
	"github.com/janearc/celestial/orbit/orbittest"
	"github.com/janearc/celestial/si"
)

// galaxyNumbers is the numbers the galaxy's tests use, from
// testdata/galaxy.json.
type galaxyNumbers struct {
	Sun struct {
		Distance  float64
		Speed     float64
		SpeedRoom float64 `json:"speed room"`
	}
	Shares struct{ Bulge, Disc, Halo, Room float64 }
	Hole   struct {
		Suns      float64
		Share     float64 `json:"share at the sun"`
		Near      float64
		ShareNear float64 `json:"share near"`
		GMRoom    float64 `json:"gm room"`
	}
	Masses struct {
		Bulge, Disc, Virial, Within float64
		VirialRadius                float64 `json:"virial radius"`
		WithinRadius                float64 `json:"within radius"`
		Far                         float64
		PointRoom                   float64 `json:"point room"`
		Cutoffs                     float64 `json:"cut-offs"`
		Room                        struct{ Bulge, Disc, Virial, Within, Arithmetic float64 }
	}
	Densities struct {
		Total, Dark float64
		Room        struct{ Total, Dark float64 }
	}
	Vertical struct {
		Height, Want, Room float64
	} `json:"vertical force"`
	Slope          struct{ Want, Room, Step float64 }
	Escape         struct{ Want, Room float64 }
	Shape          struct{ Exponent, Cutoff, Length, Height, Scale float64 }
	Step           float64                                 `json:"difference step"`
	Places         struct{ Bulge, Disc, Halo [][]float64 } `json:"density places"`
	DefinitionRoom float64                                 `json:"definition room"`
	Drawn          struct{ Farthest, Nearest float64 }
	GradientRoom   float64 `json:"gradient room"`
	Halved         float64
	Refused        struct {
		Sun, Circular, Share, Exponent, Cutoff, Length, Height, Scale float64
	}
	None float64
}

// pickedGalaxy reads the galaxy's test numbers, or the test stops.
func pickedGalaxy(test *testing.T) galaxyNumbers {
	test.Helper()
	var numbers galaxyNumbers
	raw, err := os.ReadFile("testdata/galaxy.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// The galaxy tests' own numbers, each defined.
const (
	cube    = 3            // defined: a volume is a length cubed
	twice   = 2            // defined: a central difference spans two steps, and a square is a power of two
	sphere  = twice * turn // defined: a sphere's solid angle is two turns
	mnLifts = 3            // defined by the Miyamoto-Nagai density: three lifts beside the length
	mnPower = 2.5          // defined by the Miyamoto-Nagai density: the reach to the five halves
)

// The axes' places in a triple, each defined.
const (
	axisX = iota
	axisY
	axisZ
)

// welled is a field with a well: a potential whose Pull is minus the
// gradient of its Well.
type welled interface {
	orbit.Potential
	Well(at orbit.Place) float64
}

// gmSun is a sun's mass times the gravitational constant in the
// package's units, by the route grav and si take.
func gmSun() float64 {
	kilometre := si.Kilo(length.Metre)
	return si.Gravitation() * measure.SolarMasses(whole).InKilograms() /
		(si.Kilo(length.Parsec) * kilometre * kilometre)
}

// galaxy is the published model with Sgr A* at the centre, or the test
// stops.
func galaxy(test *testing.T, numbers galaxyNumbers) *orbit.MilkyWay {
	test.Helper()
	made, err := orbit.NewMilkyWay(orbit.MWPotential2014(),
		orbittest.Centre(numbers.Hole.Suns))
	if err != nil {
		test.Fatal(err)
	}
	return made
}

// placeOf is a place from a triple of kiloparsecs.
func placeOf(at []float64) orbit.Place {
	return orbit.PlaceOf(at[axisX], at[axisY], at[axisZ])
}

// pullAt is a field's pull at a triple, as a triple.
func pullAt(field orbit.Potential, at []float64) []float64 {
	x, y, z := field.Pull(placeOf(at)).Along()
	return []float64{x, y, z}
}

// radial is the pull toward the centre at a radius in the plane, as a
// positive number.
func radial(field orbit.Potential, radius float64) float64 {
	return -pullAt(field, []float64{radius, nothing, nothing})[axisX]
}

// circular is the circular speed at a radius in the plane.
func circular(field orbit.Potential, radius float64) float64 {
	return math.Sqrt(radial(field, radius) * radius)
}

// divergence is the divergence of a field's pull at a triple, by central
// differences of a step: minus four pi times G times the density.
func divergence(field orbit.Potential, at []float64, step float64) float64 {
	sum := nothing
	for axis := range at {
		plus, minus := slices.Clone(at), slices.Clone(at)
		plus[axis] += step
		minus[axis] -= step
		sum += (pullAt(field, plus)[axis] - pullAt(field, minus)[axis]) /
			(twice * step)
	}
	return sum
}

// gDensity is a field's density times G at a triple, from the
// divergence of its pull, in the package's units.
func gDensity(field orbit.Potential, at []float64, step float64) float64 {
	return -divergence(field, at, step) / sphere
}

// density is a field's density at a triple in suns a cubic parsec, from
// the divergence of its pull.
func density(field orbit.Potential, at []float64, step float64) float64 {
	perKiloparsec := si.Kilo(length.Parsec) / length.Parsec
	return gDensity(field, at, step) / (gmSun() * math.Pow(perKiloparsec,
		cube))
}

// gradient is minus the gradient of a well at a triple, by central
// differences of a step: what the pull should be.
func gradient(field welled, at []float64, step float64) []float64 {
	pull := make([]float64, len(at))
	for axis := range at {
		plus, minus := slices.Clone(at), slices.Clone(at)
		plus[axis] += step
		minus[axis] -= step
		pull[axis] = -(field.Well(placeOf(plus)) - field.Well(placeOf(minus))) /
			(twice * step)
	}
	return pull
}

// size is the length of a triple.
func size(triple []float64) float64 {
	return orbittest.Size(triple[axisX], triple[axisY], triple[axisZ])
}

// pullSize is the size of a field's pull at a triple.
func pullSize(field orbit.Potential, at []float64) float64 {
	return size(pullAt(field, at))
}

// weighed is the mass in suns a field pulls as at a triple, were it a
// point mass: its pull times the distance squared, over a sun's gm.
func weighed(field orbit.Potential, at []float64) float64 {
	return pullSize(field, at) * size(at) * size(at) / gmSun()
}

// parts is a model's ten numbers, for the tests to build models from.
type parts struct {
	sun, circular                  float64
	bulgeShare, exponent, cutoff   float64
	discShare, scaleLength, height float64
	haloShare, scale               float64
}

// partsOf is a model's numbers, read back from it.
func partsOf(model orbit.Model) parts {
	var taken parts
	taken.sun, taken.circular = model.Sun()
	taken.bulgeShare, taken.discShare, taken.haloShare = model.Shares()
	taken.exponent, taken.cutoff = model.BulgeShape()
	taken.scaleLength, taken.height = model.DiscShape()
	taken.scale = model.HaloShape()
	return taken
}

// model is the model built from the numbers.
func (taken parts) model() orbit.Model {
	return orbit.NewModel(taken.sun, taken.circular).
		WithBulge(taken.bulgeShare, taken.exponent, taken.cutoff).
		WithDisc(taken.discShare, taken.scaleLength, taken.height).
		WithHalo(taken.haloShare, taken.scale)
}

// the published model is the ledger's, its disc height in kiloparsecs,
// and is what its numbers build; the sun circles at the model's speed,
// lifted by the hole's part; and each part pulls its share of the force
// there.
func TestTheSunCirclesAtTheModelsSpeed(test *testing.T) {
	numbers := pickedGalaxy(test)
	model := orbit.MWPotential2014()
	shape := numbers.Shape
	if taken := partsOf(model); taken.exponent != shape.Exponent ||
		taken.cutoff != shape.Cutoff || taken.scaleLength != shape.Length ||
		taken.height != shape.Height || taken.scale != shape.Scale ||
		taken.model() != model {
		test.Errorf("the model's numbers are %+v, its shapes want %+v",
			taken, shape)
	}
	made := galaxy(test, numbers)
	sun := numbers.Sun
	if got := circular(made, sun.Distance); !orbittest.Within(got,
		sun.Speed, sun.SpeedRoom) {
		test.Errorf("the sun circles at %g km/s, want %g", got, sun.Speed)
	}
	force := sun.Speed * sun.Speed / sun.Distance
	shares := numbers.Shares
	for name, part := range map[string]struct {
		field orbit.Potential
		share float64
	}{"bulge": {made.Bulge(), shares.Bulge}, "disc": {made.Disc(),
		shares.Disc}, "halo": {made.Halo(), shares.Halo}} {
		got := radial(part.field, sun.Distance) / force
		if !orbittest.Within(got, part.share, shares.Room) {
			test.Errorf("the %s pulls %g of the force at the sun, want %g",
				name, got, part.share)
		}
	}
}

// the test and the anti-test: far out every part is a point mass, and
// the masses they weigh there are the paper's: the bulge's and the
// disc's whole, the halo's within the virial radius, and the mass within
// sixty kiloparsecs, with the disc counted whole.
func TestTheMassesAreThePapers(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	masses := numbers.Masses
	far := masses.Far
	bulge := weighed(made.Bulge(), []float64{far, nothing, nothing})
	sun := numbers.Sun
	arithmetic := numbers.Shares.Bulge * sun.Speed * sun.Speed * sun.Distance /
		gmSun()
	if !orbittest.Within(bulge, masses.Bulge, masses.Room.Bulge) ||
		!orbittest.Within(bulge, arithmetic, masses.Room.Arithmetic*arithmetic) {
		test.Errorf("the bulge weighs %g suns, want %g, by the arithmetic %g",
			bulge, masses.Bulge, arithmetic)
	}
	disc := weighed(made.Disc(), []float64{far, nothing, nothing})
	above := weighed(made.Disc(), []float64{nothing, nothing, far})
	if !orbittest.Within(disc, masses.Disc, masses.Room.Disc) ||
		!orbittest.Within(above, disc, masses.PointRoom*disc) {
		test.Errorf("the disc weighs %g suns in its plane and %g above,"+
			" want %g", disc, above, masses.Disc)
	}
	virial := weighed(made.Halo(), []float64{masses.VirialRadius, nothing,
		nothing})
	if !orbittest.Within(virial, masses.Virial, masses.Room.Virial) {
		test.Errorf("the halo weighs %g suns within %g kpc, want %g", virial,
			masses.VirialRadius, masses.Virial)
	}
	edge := []float64{masses.WithinRadius, nothing, nothing}
	within := weighed(made.Bulge(), edge) + weighed(made.Halo(), edge) + disc
	if !orbittest.Within(within, masses.Within, masses.Room.Within) {
		test.Errorf("the galaxy weighs %g suns within %g kpc, want %g",
			within, masses.WithinRadius, masses.Within)
	}
}

// at the sun the mid-plane density, the dark matter's density, the
// vertical force a kiloparsec and a tenth up, the rotation curve's slope
// and the escape speed are the paper's.
func TestTheSunsNeighbourhoodIsThePapers(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	sun := []float64{numbers.Sun.Distance, nothing, nothing}
	densities := numbers.Densities
	if got := density(made, sun, numbers.Step); !orbittest.Within(got,
		densities.Total, densities.Room.Total) {
		test.Errorf("the density at the sun is %g suns a cubic parsec, want"+
			" %g", got, densities.Total)
	}
	if got := density(made.Halo(), sun, numbers.Step); !orbittest.Within(
		got, densities.Dark, densities.Room.Dark) {
		test.Errorf("the dark density at the sun is %g suns a cubic parsec,"+
			" want %g", got, densities.Dark)
	}
	vertical := numbers.Vertical
	up := []float64{numbers.Sun.Distance, nothing, vertical.Height}
	perKiloparsec := si.Kilo(length.Parsec) / length.Parsec
	unit := turn * gmSun() * perKiloparsec * perKiloparsec
	if got := -pullAt(made, up)[axisZ] / unit; !orbittest.Within(got,
		vertical.Want, vertical.Room) {
		test.Errorf("the vertical force %g kpc up is %g, want %g",
			vertical.Height, got, vertical.Want)
	}
	slope := numbers.Slope
	in, out := numbers.Sun.Distance-slope.Step, numbers.Sun.Distance+slope.Step
	got := (math.Log(circular(made, out)) - math.Log(circular(made, in))) /
		(math.Log(out) - math.Log(in))
	if !orbittest.Within(got, slope.Want, slope.Room) {
		test.Errorf("the rotation curve's slope at the sun is %g, want %g",
			got, slope.Want)
	}
	escape := numbers.Escape
	well := made.Well(placeOf(sun))
	if got := math.Sqrt(-twice * well); !orbittest.Within(got, escape.Want,
		escape.Room) {
		test.Errorf("the escape speed at the sun is %g km/s, want %g", got,
			escape.Want)
	}
}

// each part's density, from the divergence of its pull, is its
// definition: the bulge's power law under its cut-off, the Miyamoto-Nagai
// disc's, the Navarro-Frenk-White halo's, at places in and out of the
// plane, near the centre and far out.
func TestEachPartsDensityIsItsDefinition(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	model := partsOf(orbit.MWPotential2014())
	definitions := map[string]struct {
		field  orbit.Potential
		places [][]float64
		want   func(made *orbit.MilkyWay, model parts, at []float64) float64
	}{
		"bulge": {made.Bulge(), numbers.Places.Bulge, bulgeDefined},
		"disc":  {made.Disc(), numbers.Places.Disc, discDefined},
		"halo":  {made.Halo(), numbers.Places.Halo, haloDefined},
	}
	for name, part := range definitions {
		for _, at := range part.places {
			got := gDensity(part.field, at, numbers.Step)
			want := part.want(made, model, at)
			if !orbittest.Within(got, want, numbers.DefinitionRoom*want) {
				test.Errorf("the %s's density at %v is %g, its definition %g",
					name, at, got, want)
			}
		}
	}
}

// bulgeDefined is a galaxy's bulge's density definition times G at a
// triple, from the bulge's own shape, which must be the model's.
func bulgeDefined(made *orbit.MilkyWay, model parts, at []float64) float64 {
	exponent, cutoff := made.Bulge().Shape()
	if exponent != model.exponent || cutoff != model.cutoff {
		return math.NaN()
	}
	return bulgeDensity(made.Bulge().GM(), exponent, cutoff, size(at))
}

// discDefined is a galaxy's disc's density definition times G at a
// triple, from the disc's own shape, which must be the model's.
func discDefined(made *orbit.MilkyWay, model parts, at []float64) float64 {
	scaleLength, height := made.Disc().Shape()
	if scaleLength != model.scaleLength || height != model.height {
		return math.NaN()
	}
	return discDensity(made.Disc().GM(), scaleLength, height, at)
}

// haloDefined is a galaxy's halo's density definition times G at a
// triple, from the halo's own shape, which must be the model's.
func haloDefined(made *orbit.MilkyWay, model parts, at []float64) float64 {
	scale := made.Halo().Scale()
	if scale != model.scale {
		return math.NaN()
	}
	return haloDensity(made.Halo().GM(), scale, size(at))
}

// bulgeDensity is the bulge's density definition times G at a distance:
// the whole gm times the distance to the exponent times the cut-off's
// exponential, over two pi, the cut-off to three plus the exponent and
// the gamma function of half that.
func bulgeDensity(gm, exponent, cutoff, distance float64) float64 {
	order := (cube + exponent) / twice
	cut := distance / cutoff
	return gm * math.Pow(distance, exponent) * math.Exp(-cut*cut) /
		(turn * math.Pow(cutoff, cube+exponent) * math.Gamma(order))
}

// discDensity is the Miyamoto-Nagai density times G at a triple.
func discDensity(gm, scaleLength, height float64, at []float64) float64 {
	lift := math.Hypot(at[axisZ], height)
	radius := math.Hypot(at[axisX], at[axisY])
	reach := scaleLength + lift
	top := scaleLength*radius*radius + (scaleLength+mnLifts*lift)*reach*reach
	bottom := math.Pow(radius*radius+reach*reach, mnPower) * lift * lift * lift
	return gm * height * height * top / (sphere * bottom)
}

// haloDensity is the Navarro-Frenk-White density times G at a distance:
// the mass of a scale over four pi scale radii cubed, over the distance
// in scales times one plus it squared.
func haloDensity(gm, scale, distance float64) float64 {
	cut := distance / scale
	return gm / (sphere * scale * scale * scale * cut * (whole + cut) *
		(whole + cut))
}

// the anti-test: a parsec from the centre the galaxy is mostly the hole,
// and at the sun the hole is nothing; the hole is Sgr A*'s mass.
func TestTheHoleRulesTheInnerParsecAndNothingAtTheSun(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	hole := numbers.Hole
	if want := hole.Suns * gmSun(); !orbittest.Within(made.Hole().GM(), want,
		hole.GMRoom*want) {
		test.Errorf("the hole's gm is %g, want %g", made.Hole().GM(), want)
	}
	near := []float64{hole.Near, nothing, nothing}
	if share := pullSize(made.Hole(), near) / size(pullAt(made,
		near)); !(share >= hole.ShareNear) {
		test.Errorf("a parsec out the hole is %g of the pull, want at least"+
			" %g", share, hole.ShareNear)
	}
	sun := []float64{numbers.Sun.Distance, nothing, nothing}
	if share := pullSize(made.Hole(), sun) / pullSize(made, sun); !(share <=
		hole.Share) {
		test.Errorf("at the sun the hole is %g of the pull, past %g", share,
			hole.Share)
	}
}

// the anti-test: past ten cut-offs the bulge is a point mass, and its
// shells there hold nothing: its well is a point mass's.
func TestPastItsCutoffTheBulgeIsAPointMass(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	_, cutoff := orbit.MWPotential2014().BulgeShape()
	far := numbers.Masses.Cutoffs * cutoff
	at := []float64{far, nothing, nothing}
	gm := made.Bulge().GM()
	room := numbers.Masses.PointRoom
	if weighed := pullSize(made.Bulge(), at) * far * far; !orbittest.Within(
		weighed, gm, room*gm) {
		test.Errorf("past its cut-off the bulge pulls as %g, not its %g",
			weighed, gm)
	}
	if well, want := made.Bulge().Well(placeOf(at)), -gm/far; !orbittest.Within(
		well, want, room*gm/far) {
		test.Errorf("past its cut-off the bulge's well is %g, not a point"+
			" mass's %g", well, want)
	}
}

// the anti-test: the centre is no place for a pull: the bulge's, the
// halo's and the hole's are not numbers there, and the disc's is
// nothing, since the disc has a height.
func TestTheCentreIsNoPlaceForAPull(test *testing.T) {
	numbers := pickedGalaxy(test)
	made := galaxy(test, numbers)
	centre := []float64{numbers.None, numbers.None, numbers.None}
	for name, field := range map[string]orbit.Potential{"bulge": made.Bulge(),
		"halo": made.Halo(), "hole": made.Hole(), "galaxy": made} {
		if pull := pullAt(field, centre); finite(size(pull)) {
			test.Errorf("the %s pulls %v at the centre", name, pull)
		}
	}
	if pull := pullAt(made.Disc(), centre); size(pull) != numbers.None {
		test.Errorf("the disc pulls %v at the centre", pull)
	}
}

// the anti-test: a model with every part's share halved circles the sun
// slower by the root of a half, so each part is scaled by its share.
func TestHalvedSharesSlowTheSun(test *testing.T) {
	numbers := pickedGalaxy(test)
	taken := partsOf(orbit.MWPotential2014())
	taken.bulgeShare *= numbers.Halved
	taken.discShare *= numbers.Halved
	taken.haloShare *= numbers.Halved
	made, err := orbit.NewMilkyWay(taken.model(),
		orbittest.Centre(numbers.Hole.Suns))
	if err != nil {
		test.Fatal(err)
	}
	want := numbers.Sun.Speed * math.Sqrt(numbers.Halved)
	if got := circular(made, numbers.Sun.Distance); !orbittest.Within(got,
		want, numbers.Sun.SpeedRoom) {
		test.Errorf("with halved shares the sun circles at %g km/s, want %g",
			got, want)
	}
}

// a model may leave a part out: a share of nothing is not refused, and
// the part pulls nothing; nor is a disc with no length, which is a
// Plummer sphere, and pulls.
func TestAModelMayLeaveAPartOut(test *testing.T) {
	numbers := pickedGalaxy(test)
	taken := partsOf(orbit.MWPotential2014())
	taken.bulgeShare = numbers.None
	taken.scaleLength = numbers.None
	made, err := orbit.NewMilkyWay(taken.model(),
		orbittest.Centre(numbers.Hole.Suns))
	if err != nil {
		test.Fatal(err)
	}
	sun := []float64{numbers.Sun.Distance, nothing, nothing}
	if pull := pullSize(made.Bulge(), sun); pull != numbers.None {
		test.Errorf("a bulge with no share pulls %g", pull)
	}
	if pull := pullSize(made.Disc(), sun); !(pull > numbers.None) {
		test.Errorf("a disc with no length pulls %g", pull)
	}
}

// the anti-test: a model the physics cannot hold is refused, with
// nothing beside the error: no circle for the sun, a negative share, a
// bulge whose well diverges or whose exponent is past any number or
// with no cut-off, a disc with a negative length or no height, a halo
// with no scale.
func TestAModelThePhysicsCannotHoldIsRefused(test *testing.T) {
	numbers := pickedGalaxy(test)
	refused := numbers.Refused
	published := partsOf(orbit.MWPotential2014())
	broken := map[string]parts{}
	taken := published
	taken.sun = refused.Sun
	broken["the sun at the centre"] = taken
	taken = published
	taken.circular = refused.Circular
	broken["no circular speed"] = taken
	taken = published
	taken.discShare = refused.Share
	broken["a negative share"] = taken
	taken = published
	taken.exponent = refused.Exponent
	broken["a bulge whose well diverges"] = taken
	taken = published
	taken.exponent = math.MaxFloat64
	broken["a bulge of an exponent past any number"] = taken
	taken = published
	taken.cutoff = refused.Cutoff
	broken["a bulge with no cut-off"] = taken
	taken = published
	taken.scaleLength = refused.Length
	broken["a disc of negative length"] = taken
	taken = published
	taken.height = refused.Height
	broken["a disc with no height"] = taken
	taken = published
	taken.scale = refused.Scale
	broken["a halo with no scale"] = taken
	for name, taken := range broken {
		made, err := orbit.NewMilkyWay(taken.model(),
			orbittest.Centre(numbers.Hole.Suns))
		if err == nil || made != nil {
			test.Errorf("%s was not refused: %v, %v", name, made, err)
		}
	}
}

// galaxyProperties is the galaxy's properties, under a test.
type galaxyProperties struct {
	test    *testing.T
	numbers galaxyNumbers
	made    *orbit.MilkyWay
}

// the property: at any place minus the well's gradient is the pull, for
// each part and the whole; the whole is its parts; and every well is
// below nothing.
func TestMinusTheWellsGradientIsThePull(test *testing.T) {
	numbers := pickedGalaxy(test)
	holds := galaxyProperties{test, numbers, galaxy(test, numbers)}
	rapid.Check(test, holds.minusTheGradient)
}

// minusTheGradient is the property at one drawn place.
func (holds galaxyProperties) minusTheGradient(draw *rapid.T) {
	at := holds.anyPlace(draw)
	made := holds.made
	parts := []welled{made.Bulge(), made.Disc(), made.Halo(), made.Hole()}
	sum := make([]float64, len(at))
	well := nothing
	for _, part := range parts {
		pull := pullAt(part, at)
		for axis := range sum {
			sum[axis] += pull[axis]
		}
		well += part.Well(placeOf(at))
	}
	total := pullAt(made, at)
	for axis := range sum {
		if !orbittest.Within(total[axis], sum[axis], holds.numbers.GradientRoom*
			size(sum)) {
			draw.Errorf("at %v the galaxy pulls %v, its parts %v", at, total, sum)
		}
	}
	place := placeOf(at)
	if deep := made.Well(place); !orbittest.Within(deep, well,
		holds.numbers.GradientRoom*math.Abs(well)) {
		draw.Errorf("at %v the galaxy's well is %g, its parts' %g", at, deep,
			well)
	}
	for _, field := range append(parts, made) {
		pull, want := pullAt(field, at), gradient(field, at, holds.numbers.Step)
		room := holds.numbers.GradientRoom * size(pull)
		for axis := range pull {
			if !orbittest.Within(want[axis], pull[axis], room) {
				draw.Errorf("at %v a pull is %v, minus its well's gradient %v",
					at, pull, want)
			}
		}
		if deep := field.Well(place); deep >= nothing {
			draw.Errorf("at %v a well is %g, not below nothing", at, deep)
		}
	}
}

// anyPlace is a place drawn from the picked ranges, no nearer the
// centre than the nearest.
func (holds galaxyProperties) anyPlace(draw *rapid.T) []float64 {
	drawn := holds.numbers.Drawn
	at := make([]float64, cube)
	for axis := range at {
		at[axis] = rapid.Float64Range(-drawn.Farthest, drawn.Farthest).Draw(
			draw, "at")
	}
	if size(at) < drawn.Nearest {
		at[axisX] += math.Copysign(drawn.Nearest, at[axisX])
	}
	return at
}
