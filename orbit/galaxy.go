package orbit

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/research"
)

// Model is a galaxy's numbers as Bovy 2015 gives MWPotential2014's: the
// Sun's distance from the centre and the circular speed there, which
// the model is normalised to; each part's share of the radial force
// there; and each part's shape.
//
// Lengths are kiloparsecs, the speed is kilometres per second, shares
// and the exponent are plain numbers. A model is built from the Sun's
// circle with each part added: NewModel(8, 220).WithBulge(0.05, -1.8,
// 1.9) and so on, as MWPotential2014 does from the research ledger.
type Model struct {
	sun, circular                     float64
	bulgeShare, bulgeExponent, cutoff float64
	discShare, scaleLength, height    float64
	haloShare, scale                  float64
}

// NewModel is a model of a galaxy where the Sun is so many kiloparsecs
// from the centre, circling at so many kilometres per second, with no
// parts yet.
func NewModel(sun, circular float64) Model {
	return Model{sun: sun, circular: circular}
}

// WithBulge is the model with its bulge: the bulge's share of the
// radial force at the Sun, its density's exponent and its cut-off in
// kiloparsecs.
func (model Model) WithBulge(share, exponent, cutoff float64) Model {
	model.bulgeShare, model.bulgeExponent, model.cutoff = share, exponent,
		cutoff
	return model
}

// WithDisc is the model with its disc: the disc's share of the radial
// force at the Sun, its scale length and its scale height, in
// kiloparsecs.
func (model Model) WithDisc(share, scaleLength, height float64) Model {
	model.discShare, model.scaleLength, model.height = share, scaleLength,
		height
	return model
}

// WithHalo is the model with its halo: the halo's share of the radial
// force at the Sun and its scale radius in kiloparsecs.
func (model Model) WithHalo(share, scale float64) Model {
	model.haloShare, model.scale = share, scale
	return model
}

// Sun is the Sun's distance from the centre in kiloparsecs and the
// circular speed there in kilometres per second.
func (model Model) Sun() (distance, circular float64) {
	return model.sun, model.circular
}

// Shares is each part's share of the radial force at the Sun.
func (model Model) Shares() (bulge, disc, halo float64) {
	return model.bulgeShare, model.discShare, model.haloShare
}

// BulgeShape is the bulge's density exponent and its cut-off in
// kiloparsecs.
func (model Model) BulgeShape() (exponent, cutoff float64) {
	return model.bulgeExponent, model.cutoff
}

// DiscShape is the disc's scale length and scale height in kiloparsecs.
func (model Model) DiscShape() (scaleLength, height float64) {
	return model.scaleLength, model.height
}

// HaloShape is the halo's scale radius in kiloparsecs.
func (model Model) HaloShape() float64 { return model.scale }

// MWPotential2014 is the model as the research ledger has it from the
// paper's table 1: the Sun at 8 kpc circling at 220 km/s, the bulge a
// twentieth of the force there, the disc three fifths, the halo the
// rest.
func MWPotential2014() Model {
	parsec := length.Parsec / kiloparsec
	return NewModel(research.Value("mw2014-sun-distance"),
		research.Value("mw2014-circular-speed")).
		WithBulge(research.Value("mw2014-bulge-share"),
			research.Value("mw2014-bulge-exponent"),
			research.Value("mw2014-bulge-cutoff")).
		WithDisc(research.Value("mw2014-disc-share"),
			research.Value("mw2014-disc-length"),
			research.Value("mw2014-disc-height")*parsec).
		WithHalo(research.Value("mw2014-halo-share"),
			research.Value("mw2014-halo-scale"))
}

// MilkyWay is the galaxy's pull as a model has it: a bulge whose
// density is a power law with an exponential cut-off, a Miyamoto-Nagai
// disc and a Navarro-Frenk-White halo, and the black hole at the centre
// as a point mass.
//
// Each part is normalised by its share of the radial force at the
// Sun's distance in the model, where the circular speed is the model's.
//
// Bovy 2015 says MWPotential2014 does not contain the black hole. It is
// added here, from the measured mass of Sgr A*, since the inner parsecs
// are this search's regime; its pull is a part in twenty thousand of
// the galaxy's at the Sun.
//
// The model lives at its own distance for the Sun and its own circular
// speed there. A caller placing the Sun at its measured distance does
// that in the transform, as galpy does, not by rescaling the model.
//
// That seam is open. At the measured distance, 8.277 kpc, the model's
// circular speed is about 219 km/s (220 at 8 kpc, falling as the
// curve's slope has it) and the Sun's about 231 with its own motion
// added; the galactocentric frame's Sun goes 251.5 along the rotation.
//
// The 20 km/s between them is carried by every star the frame places
// until the model is placed at the frame's Sun, as galpy's ro and vo
// do, or the frame's Sun is given the model's speed. That choice is not
// made here.
type MilkyWay struct {
	bulge Bulge
	disc  Disc
	halo  Halo
	hole  Kepler
}

// NewMilkyWay is the galaxy of a model, with the given black hole at
// the centre: the caller makes it, as a point of the ledger's
// sgr-a-mass for the real one.
//
// It refuses a model the physics cannot hold: no circle for the Sun, a
// negative share, a bulge whose well diverges, or a part with no size.
func NewMilkyWay(model Model, hole grav.Body) (*MilkyWay, error) {
	if !(model.sun > nothing) || !(model.circular > nothing) {
		return nil, fmt.Errorf("orbit: a model with the sun %g kpc out"+
			" circling at %g km/s", model.sun, model.circular)
	}
	for _, share := range []float64{
		model.bulgeShare, model.discShare,
		model.haloShare,
	} {
		if !(share >= nothing) {
			return nil, fmt.Errorf(
				"orbit: a model with a part's share"+
					" of the force %g", share)
		}
	}
	force := model.circular * model.circular / model.sun
	bulge, err := newBulge(model.bulgeShare*force, model.bulgeExponent,
		model.cutoff, model.sun)
	if err != nil {
		return nil, err
	}
	disc, err := newDisc(model.discShare*force, model.scaleLength,
		model.height, model.sun)
	if err != nil {
		return nil, err
	}
	halo, err := newHalo(model.haloShare*force, model.scale, model.sun)
	if err != nil {
		return nil, err
	}
	return &MilkyWay{*bulge, *disc, *halo, NewKepler(hole)}, nil
}

// Bulge is the bulge's pull.
func (galaxy MilkyWay) Bulge() Bulge { return galaxy.bulge }

// Disc is the disc's pull.
func (galaxy MilkyWay) Disc() Disc { return galaxy.disc }

// Halo is the halo's pull.
func (galaxy MilkyWay) Halo() Halo { return galaxy.halo }

// Hole is the black hole's pull.
func (galaxy MilkyWay) Hole() Kepler { return galaxy.hole }

// Pull is the four pulls together.
func (galaxy MilkyWay) Pull(at Place) Pull {
	sum := galaxy.bulge.Pull(at).along.plus(galaxy.disc.Pull(at).along)
	sum = sum.plus(galaxy.halo.Pull(at).along)
	return Pull{sum.plus(galaxy.hole.Pull(at).along)}
}

// Well is the four wells together.
func (galaxy MilkyWay) Well(at Place) float64 {
	return galaxy.bulge.Well(at) + galaxy.disc.Well(at) +
		galaxy.halo.Well(at) + galaxy.hole.Well(at)
}

// Well is a point mass's well at a place, in kilometres squared per
// second squared, below nothing: its gm over the distance.
func (field Kepler) Well(at Place) float64 {
	return -field.gm / at.at.norm()
}

// The galaxy's own numbers, each defined.
const (
	outer       = 3   // defined: three powers of r in a sphere
	halfPower   = 0.5 // defined: the well's order, half under the mass's
	halves      = 2   // defined: a square is a power of two
	threeHalves = 1.5 // defined: a cube over a square, of the square
)

// Bulge is a bulge whose density falls as a power of the distance from
// the centre with an exponential cut-off: proportional to r to the
// exponent, times e to the minus (r over the cut-off) squared, as galpy's
// PowerSphericalPotentialwCutoff. The exponent is negative.
//
// Its pull is that of the mass within the distance, which the
// regularised lower incomplete gamma function gives, of order half of
// three plus the exponent.
type Bulge struct {
	gm       float64
	exponent float64
	cutoff   float64
}

// newBulge is a bulge whose pull at a radius on the model's circle is
// so much, with a density exponent and a cut-off in kiloparsecs: the
// whole bulge's gm follows from the share of its mass within that
// radius.
//
// It refuses an exponent of minus two or past it, where the well's
// closed form fails though the mass converges to minus three, an
// exponent past any number, and a cut-off that is not positive.
func newBulge(pull, exponent, cutoff, radius float64) (*Bulge, error) {
	if !(exponent > -halves) || !(exponent < math.MaxFloat64) {
		return nil, fmt.Errorf(
			"orbit: a bulge of exponent %g, past what"+
				" its well's closed form holds", exponent)
	}
	if !(cutoff > nothing) {
		return nil, fmt.Errorf("orbit: a bulge cut off at %g kpc",
			cutoff)
	}
	bulge := Bulge{gm: whole, exponent: exponent, cutoff: cutoff}
	bulge.gm = pull * radius * radius / bulge.within(radius)
	return &bulge, nil
}

// GM is the whole bulge's mass times the gravitational constant, in
// kiloparsecs times kilometres per second squared.
func (bulge Bulge) GM() float64 { return bulge.gm }

// Shape is the bulge's density exponent and its cut-off in kiloparsecs.
func (bulge Bulge) Shape() (exponent, cutoff float64) {
	return bulge.exponent, bulge.cutoff
}

// Order is the incomplete gamma function's order for the mass within a
// distance: half of three plus the exponent.
func (bulge Bulge) Order() float64 { return (outer + bulge.exponent) / halves }

// within is the share of the bulge's mass within a distance.
func (bulge Bulge) within(distance float64) float64 {
	cut := distance / bulge.cutoff
	return lowerGamma(bulge.Order(), cut*cut)
}

// Pull is the pull of the mass within the distance, toward the centre.
// At the centre itself it is not a number.
func (bulge Bulge) Pull(at Place) Pull {
	distance := at.at.norm()
	cubed := distance * distance * distance
	return Pull{at.at.times(-bulge.gm * bulge.within(distance) / cubed)}
}

// Well is the bulge's potential at a place, in kilometres squared per
// second squared, below nothing: the mass within over the distance, and the
// shells outside, which the regularised upper incomplete gamma function
// of half an order less gives.
func (bulge Bulge) Well(at Place) float64 {
	distance := at.at.norm()
	cut := distance / bulge.cutoff
	inner := bulge.within(distance) / distance
	above := bulge.Order() - halfPower
	shells := math.Gamma(above) * upperGamma(above, cut*cut) /
		(bulge.cutoff * math.Gamma(bulge.Order()))
	return -bulge.gm * (inner + shells)
}

// Disc is a Miyamoto-Nagai disc: a Plummer sphere stretched along the
// plane by a scale length, with a scale height, both in kiloparsecs.
type Disc struct {
	gm     float64
	length float64
	height float64
}

// newDisc is a disc whose pull at a radius in its plane is so much,
// with a scale length and a scale height in kiloparsecs.
//
// It refuses a negative length and a height that is not positive,
// since a disc with none is a sheet whose pull across it is not a
// number.
func newDisc(pull, scaleLength, height, radius float64) (*Disc, error) {
	if !(scaleLength >= nothing) {
		return nil, fmt.Errorf("orbit: a disc of scale length %g"+
			" kpc", scaleLength)
	}
	if !(height > nothing) {
		return nil, fmt.Errorf("orbit: a disc of scale height %g kpc",
			height)
	}
	disc := Disc{gm: whole, length: scaleLength, height: height}
	disc.gm = pull /
		disc.Pull(PlaceOf(radius, nothing, nothing)).along.norm()
	return &disc, nil
}

// GM is the disc's mass times the gravitational constant, in kiloparsecs
// times kilometres per second squared.
func (disc Disc) GM() float64 { return disc.gm }

// Shape is the disc's scale length and scale height in kiloparsecs.
func (disc Disc) Shape() (scaleLength, height float64) {
	return disc.length, disc.height
}

// reach is the disc's reach at a place: the scale length plus the
// height's hypotenuse with the place's height, and the square of the
// distance from the place to the disc's centre along it.
func (disc Disc) reach(at vec) (lifted, squared float64) {
	lifted = disc.length + math.Hypot(at.z, disc.height)
	return lifted, at.x*at.x + at.y*at.y + lifted*lifted
}

// Pull is the disc's pull: in the plane as a point mass at the reach,
// out of it the same scaled by the lift's share of the hypotenuse.
func (disc Disc) Pull(at Place) Pull {
	lifted, squared := disc.reach(at.at)
	cubed := math.Pow(squared, threeHalves)
	along := at.at.times(-disc.gm / cubed)
	along.z *= lifted / math.Hypot(at.at.z, disc.height)
	return Pull{along}
}

// Well is the disc's potential at a place, in kilometres squared per
// second squared, below nothing: a point mass's at the reach.
func (disc Disc) Well(at Place) float64 {
	_, squared := disc.reach(at.at)
	return -disc.gm / math.Sqrt(squared)
}

// Halo is a Navarro-Frenk-White halo: a density that falls as the
// distance near the centre and as its cube far out, turning at a scale
// radius in kiloparsecs.
//
// Its gm is of the mass of a scale: four pi times the density's scale
// times the radius cubed, times the gravitational constant.
type Halo struct {
	gm    float64
	scale float64
}

// newHalo is a halo whose pull at a radius is so much, with a scale
// radius in kiloparsecs. It refuses a scale that is not positive.
func newHalo(pull, scale, radius float64) (*Halo, error) {
	if !(scale > nothing) {
		return nil, fmt.Errorf("orbit: a halo of scale radius %g kpc",
			scale)
	}
	halo := Halo{gm: whole, scale: scale}
	halo.gm = pull * radius * radius / halo.within(radius)
	return &halo, nil
}

// GM is the mass of a scale times the gravitational constant, in
// kiloparsecs times kilometres per second squared.
func (halo Halo) GM() float64 { return halo.gm }

// Scale is the halo's scale radius in kiloparsecs.
func (halo Halo) Scale() float64 { return halo.scale }

// within is the mass within a distance as a share of the mass of a
// scale.
func (halo Halo) within(distance float64) float64 {
	cut := distance / halo.scale
	return math.Log1p(cut) - cut/(whole+cut)
}

// Pull is the pull of the mass within the distance, toward the centre.
// At the centre itself it is not a number.
func (halo Halo) Pull(at Place) Pull {
	distance := at.at.norm()
	cubed := distance * distance * distance
	return Pull{at.at.times(-halo.gm * halo.within(distance) / cubed)}
}

// Well is the halo's potential at a place, in kilometres squared per
// second squared, below nothing.
func (halo Halo) Well(at Place) float64 {
	distance := at.at.norm()
	return -halo.gm * math.Log1p(distance/halo.scale) / distance
}
