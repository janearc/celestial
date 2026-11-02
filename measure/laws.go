package measure

import (
	"math"

	"github.com/janearc/celestial/research"
)

// What follows from measured.go and exact.go by physics alone. With the speed
// of light, the reduced Planck constant and Boltzmann's constant all one, the
// constants that are left are pure numbers.
const (
	// stefanDenominator is defined by Planck's law: summed over every
	// frequency and direction it gives pi to the fourth over fifteen, over
	// the four pi squared the reduced constant brings, which is pi squared
	// over sixty.
	stefanDenominator = 60
	// stefan is Stefan and Boltzmann's constant with no units at all.
	// Defined by the one above.
	stefan = math.Pi * math.Pi / stefanDenominator
	// stefanPower is the power of the temperature in Stefan's law: a hot
	// surface gives off its temperature to the fourth. Defined by it.
	stefanPower = 4
	// steradiansOverTurns is the solid angle all round a point over a
	// full turn. Defined: a sphere is four pi steradians.
	steradiansOverTurns = 2
	// fullSphere is the solid angle all round a point, in steradians.
	// Defined by the one above.
	fullSphere = steradiansOverTurns * fullTurn
	// dipoleThirds is a dipole's scattering summed over every direction,
	// in thirds of a sphere. Defined by its formula: eight pi over three
	// is two thirds of four pi.
	dipoleThirds = 2.0 / 3
	// dipoleSphere makes Thomson's cross-section. Defined by the two above.
	dipoleSphere = fullSphere * dipoleThirds
)

// Sphere is the area of a sphere of a radius.
func Sphere(radius Length) Area {
	return Area{fullSphere * radius.natural * radius.natural}
}

// BlackBody is the power a black surface of an area gives off at a
// temperature.
func BlackBody(heat Temperature, area Area) Power {
	glow := math.Pow(heat.natural, stefanPower)
	return Power{stefan * glow * area.natural}
}

// SurfaceTemperature is how hot a black surface of an area must be to give off
// a power: the fourth root of Stefan's law, undone.
func SurfaceTemperature(power Power, area Area) Temperature {
	fourth := power.natural / (stefan * area.natural)
	return Temperature{math.Sqrt(math.Sqrt(fourth))}
}

// Photon is the energy of light of a wavelength: a full turn over the
// wavelength.
func Photon(wavelength Length) Energy {
	return Energy{fullTurn / wavelength.natural}
}

// Thomson is how much of a beam one free electron turns aside: from how
// strongly light and charge couple, and the electron's mass.
func Thomson() Area {
	ratio := research.Value(fineStructure) / Electron().natural
	return Area{dipoleSphere * ratio * ratio}
}

// Eddington is the brightest a mass can shine before its own light
// pushes its gas away: gravity pulling on each proton against light
// pushing on each electron, for gas of pure hydrogen.
//
// Past it, a star sheds itself, and a hole's food is blown back out.
func Eddington(mass Mass) Power {
	pull := gravitation() * mass.natural * Proton().natural
	return Power{fullSphere * pull / Thomson().natural}
}

// GM is a mass times newton's G: in natural units a length, half the
// horizon radius of a hole of that mass. Bodies whose GM is measured keep
// it; this is for a mass known as a mass.
func GM(mass Mass) Length {
	return Length{gravitation() * mass.natural}
}
