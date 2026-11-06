//go:build darwin

package orbit

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/orbit"
)

// The galaxy's numbers as the kernel reads them, each defined: where
// each is among them, and how many there are.
const (
	bulgeGMAt     = iota // the bulge's gm
	orderAt              // the bulge's gamma order
	cutoffAt             // the bulge's cut-off, kpc
	lnGammaAt            // ln of the gamma function at the order
	discGMAt             // the disc's gm
	discLengthAt         // the disc's scale length, kpc
	discHeightAt         // the disc's scale height, kpc
	haloGMAt             // the halo's gm, of the mass of a scale
	haloScaleAt          // the halo's scale radius, kpc
	holeGMAt             // the hole's gm
	galaxyNumbers        // how many numbers the galaxy has
)

// The kernel's gamma functions' numbers, each defined for 32-bit floats
// as orbit's are for 64.
const (
	gammaTerms     = 200   // defined: ends a loop a slip made endless
	gammaTolerance = 1e-7  // defined: a term this small is 32-bit rounding
	gammaTiny      = 1e-30 // defined: a floor under the first numerator
	haloSeriesUpTo = 0.1   // defined: the halo share is a series below
	haloTerms      = 9     // defined: the series last power, to rounding
	threeHalves    = 1.5   // defined: a cube over a square, of the square
)

// MilkyWay is the galaxy's pull as orbit has it, in Metal: the bulge by
// the regularised lower incomplete gamma function summed in 32 bits,
// the disc, the halo with its small-distance series, and the hole, each
// as orbit's reference defines them, with the galaxy as the reference.
func MilkyWay(galaxy orbit.MilkyWay) Field {
	numbers := make([]float32, galaxyNumbers)
	_, cutoff := galaxy.Bulge().Shape()
	order := galaxy.Bulge().Order()
	lnGamma, _ := math.Lgamma(order)
	numbers[bulgeGMAt] = float32(galaxy.Bulge().GM())
	numbers[orderAt] = float32(order)
	numbers[cutoffAt] = float32(cutoff)
	numbers[lnGammaAt] = float32(lnGamma)
	scaleLength, height := galaxy.Disc().Shape()
	numbers[discGMAt] = float32(galaxy.Disc().GM())
	numbers[discLengthAt] = float32(scaleLength)
	numbers[discHeightAt] = float32(height)
	numbers[haloGMAt] = float32(galaxy.Halo().GM())
	numbers[haloScaleAt] = float32(galaxy.Halo().Scale())
	numbers[holeGMAt] = float32(galaxy.Hole().GM())
	return NewField(galaxyPull(), numbers, galaxy)
}

// galaxyPull is the galaxy's pull in Metal, with its numbers' places and
// the gamma functions' numbers as Go holds them.
func galaxyPull() string {
	return fmt.Sprintf(galaxySource, bulgeGMAt, orderAt, cutoffAt,
		lnGammaAt, discGMAt, discLengthAt, discHeightAt, haloGMAt,
		haloScaleAt, holeGMAt, gammaTerms, gammaTolerance, gammaTiny,
		twice, haloSeriesUpTo, haloTerms, threeHalves, whole)
}

// twice and whole are the kernel's, as orbit's gamma functions have
// them. Defined.
const (
	twice = 2   // defined by the fraction: denominators grow by two a term
	whole = 1.0 // defined: the whole of a share
)

// galaxySource is the galaxy's pull in Metal, before its numbers are
// filled in. The gamma functions are orbit's in 32 bits; the halo's
// mass share is summed as a series near the centre, where its two terms
// cancel.
const galaxySource = `
constant uint bulgeGM = %d, order = %d, cutoff = %d, lnGamma = %d;
constant uint discGM = %d, discLength = %d, discHeight = %d;
constant uint haloGM = %d, haloScale = %d, holeGM = %d;
constant uint gammaTerms = %d;
constant float gammaTolerance = %g, gammaTiny = %g, twice = %d;
constant float haloSeriesUpTo = %g;
constant uint haloTerms = %d;
constant float threeHalves = %g, whole = %g;

static inline float gammaScale(float order, float x, float lnGamma) {
	return exp(-x + order * log(x) - lnGamma);
}

static inline float gammaSeries(float order, float x, float lnGamma) {
	float term = whole / order;
	float sum = term;
	for (uint n = 1; n <= gammaTerms; n++) {
		term *= x / (order + float(n));
		sum += term;
		if (fabs(term) < fabs(sum) * gammaTolerance) {
			break;
		}
	}
	return sum * gammaScale(order, x, lnGamma);
}

static inline float gammaFraction(float order, float x, float lnGamma) {
	float offset = x + whole - order;
	float numerators = whole / gammaTiny;
	float denominators = whole / offset;
	float value = denominators;
	for (uint n = 1; n <= gammaTerms; n++) {
		float term = float(n);
		float coefficient = -term * (term - order);
		offset += twice;
		denominators = whole / (coefficient * denominators + offset);
		numerators = offset + coefficient / numerators;
		float change = denominators * numerators;
		value *= change;
		if (fabs(change - whole) < gammaTolerance) {
			break;
		}
	}
	return gammaScale(order, x, lnGamma) * value;
}

static inline float lowerGamma(float order, float x, float lnGamma) {
	if (x < order + whole) {
		return gammaSeries(order, x, lnGamma);
	}
	return whole - gammaFraction(order, x, lnGamma);
}

static inline float haloWithin(float x) {
	if (x < haloSeriesUpTo) {
		float sum = 0.0, sign = whole, power = x * x;
		for (uint n = 2; n <= haloTerms; n++) {
			sum += sign * float(n - 1) / float(n) * power;
			sign = -sign;
			power *= x;
		}
		return sum;
	}
	return log(whole + x) - x / (whole + x);
}

static inline float3 pull(float3 at, constant float *field) {
	float r = length(at);
	float cubed = r * r * r;
	float cut = r / field[cutoff];
	float bulge = field[bulgeGM] * lowerGamma(field[order], cut * cut,
	                                          field[lnGamma]) / cubed;
	float halo = field[haloGM] * haloWithin(r / field[haloScale]) / cubed;
	float hole = field[holeGM] / cubed;
	float lift = sqrt(at.z * at.z + field[discHeight] * field[discHeight]);
	float lifted = field[discLength] + lift;
	float squared = at.x * at.x + at.y * at.y + lifted * lifted;
	float3 disc = at * (-field[discGM] / pow(squared, threeHalves));
	disc.z *= lifted / lift;
	return at * (-(bulge + halo + hole)) + disc;
}
`
