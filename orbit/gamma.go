package orbit

import "math"

// The incomplete gamma functions' numbers, each defined.
const (
	whole     = 1.0    // defined: the whole of a share
	mostTerms = 500    // defined: past any convergence here; ends a loop a slip could make endless
	tolerance = 1e-15  // defined: a term this share of its sum is rounding
	tiny      = 1e-300 // defined: a floor under the fraction's first numerator
	twice     = 2      // defined by the fraction: its denominators grow by two a term
)

// lowerGamma is the regularised lower incomplete gamma function P(order,
// x): the share of the gamma function's integral that lies below x. It
// is summed as a series below order + 1 and as a continued fraction
// above (Numerical Recipes' gammp and gammq), each to rounding.
//
// Either converges for any x that is a number, in far fewer terms than
// the most allowed; for one that is not, the value is not a number
// either.
func lowerGamma(order, x float64) float64 {
	if x < order+whole {
		return gammaSeries(order, x)
	}
	return whole - gammaFraction(order, x)
}

// upperGamma is the regularised upper incomplete gamma function Q(order,
// x), the share above x, which is one less the lower.
func upperGamma(order, x float64) float64 {
	if x < order+whole {
		return whole - gammaSeries(order, x)
	}
	return gammaFraction(order, x)
}

// gammaScale is the factor both forms share: x to the order, times e to
// the minus x, over the gamma function of the order.
func gammaScale(order, x float64) float64 {
	lnGamma, _ := math.Lgamma(order)
	return math.Exp(-x + order*math.Log(x) - lnGamma)
}

// gammaSeries is P(order, x) by its series, which converges fast for x
// below order + 1.
func gammaSeries(order, x float64) float64 {
	term := whole / order
	sum := term
	for index := range mostTerms {
		term *= x / (order + float64(index+firstStep))
		sum += term
		if math.Abs(term) < math.Abs(sum)*tolerance {
			break
		}
	}
	return sum * gammaScale(order, x)
}

// gammaFraction is Q(order, x) by its continued fraction, evaluated by
// Lentz's method, which converges fast for x above order + 1.
//
// The fraction is built term by term as the ratio of its numerators'
// and its denominators' partial values, and it is done when a term
// changes the value by rounding alone.
//
// Lentz's guards against a partial value of nothing are left out: for
// x above order + 1 the partial values stay away from it, which
// Numerical Recipes guards for arguments this field never asks.
func gammaFraction(order, x float64) float64 {
	offset := x + whole - order
	numerators := whole / tiny
	denominators := whole / offset
	value := denominators
	for index := range mostTerms {
		term := float64(index + firstStep)
		coefficient := -term * (term - order)
		offset += twice
		denominators = whole / (coefficient*denominators + offset)
		numerators = offset + coefficient/numerators
		change := denominators * numerators
		value *= change
		if math.Abs(change-whole) < tolerance {
			break
		}
	}
	return gammaScale(order, x) * value
}
