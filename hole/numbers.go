package hole

// The shape of spacetime.
const (
	// dimensions is the four of spacetime: one of time and three of space.
	// It is defined by relativity.
	dimensions = 4
	// timeAxis is the place of time in the rows and columns of a Mat4.
	// It is defined by the order (t, x, y, z).
	timeAxis = 0
	// timeSign is the metric's sign along time. It is defined by the
	// signature, which is minus, plus, plus, plus.
	timeSign = -1
	// spaceSign is the metric's sign along each axis of space. It is
	// defined by the signature.
	spaceSign = 1
	// timelikeNorm is the squared length of a unit timelike vector. It is
	// defined by the signature: minus one.
	timelikeNorm = -1
	// cameraPlaces is the three of the camera's position in the shader's
	// struct. It is defined by space.
	cameraPlaces = 3
)

// The arithmetic.
const (
	// nothing is the zero of a length, a rate or a sum. It is defined by
	// the arithmetic.
	nothing = 0
	// whole is one of what is being counted: a coordinate unit vector's
	// length, a coordinate time, the one in one over a radius. It is
	// defined by the arithmetic.
	whole = 1
	// half is the middle of two ends. It is defined by the arithmetic.
	half = 0.5
	// eitherInfinity is the sign math.IsInf takes to mean an infinity of
	// either sign. It is defined by package math.
	eitherInfinity = 0
)

// The Kerr-Schild form.
const (
	// kernelTime is the null covector's time part: one, always. It is
	// defined by the Kerr-Schild form.
	kernelTime = whole
	// scalarFactor is the 2 of the scalar 2 M r^3 / (r^4 + A^2 z^2). It is
	// defined by the Kerr-Schild form.
	scalarFactor = 2
	// radiusHalf is the one half of the radius's solution,
	// r^2 = (d + sqrt(d^2 + 4 A^2 z^2)) / 2, with d = x^2 + y^2 + z^2
	// less A^2. It is defined by that solution.
	radiusHalf = half
	// radiusFour is the 4 of that same solution, from the quadratic
	// r^4 - d r^2 - A^2 z^2 = 0. It is defined by that solution.
	radiusFour = 4
)

// The radii of a Kerr hole, by Bardeen, Press and Teukolsky (1972), "Rotating
// black holes: locally nonrotating frames, energy extraction, and scalar
// synchrotron radiation", Astrophysical Journal 178, 347.
//
// Read on 2026-10-31 by eye from the ADS scan: the horizon is its equation 2.6,
// the photon orbit 2.17 and 2.18, the innermost stable orbit 2.21.
const (
	// photonSphereFactor is the 3 of 3 M, the photon sphere of a hole with
	// no spin. It is defined by that formula.
	photonSphereFactor = 3
	// shadowSquared is the 27 of b^2 = 27 M^2, the impact parameter at
	// which a ray is just captured by a hole with no spin. It is defined
	// by that formula.
	shadowSquared = 27
	// orbitScale is the 2 of r = 2 M (1 + cos(2 arccos(-+A/M) / 3)). It is
	// defined by that formula.
	orbitScale = 2
	// orbitOne is the 1 of that formula. It is defined by it.
	orbitOne = whole
	// orbitTurn is the 2 in the cosine's angle in that formula. It is
	// defined by it.
	orbitTurn = 2
	// orbitThird is the 3 in the cosine's angle in that formula. It is
	// defined by it.
	orbitThird = 3
	// iscoOne is the 1 of Z1 = 1 + (1 - a^2)^(1/3) ((1 + a)^(1/3) +
	// (1 - a)^(1/3)), and of the cube roots' arguments. It is defined by
	// that formula.
	iscoOne = whole
	// iscoThree is the 3 of Z2 = sqrt(3 a^2 + Z1^2) and of the 3 in
	// r = M (3 + Z2 -+ sqrt((3 - Z1)(3 + Z1 + 2 Z2))). It is defined by
	// that formula.
	iscoThree = 3
	// iscoTwo is the 2 of 2 Z2 in that formula. It is defined by it.
	iscoTwo = 2
)

// The radius a ray cannot come back from, noReturn: the cubic
// y^3 + (A^2 + A Lz/E - 3 M^2) y - 2 M (M^2 - A^2) = 0, from Bardeen's
// photon-orbit ratio.
const (
	// cubicThree is the 3 of the -3 M^2 in the cubic's linear term. It is
	// defined by the cubic.
	cubicThree = 3
	// cubicTwo is the 2 of the -2 M (M^2 - A^2) in its constant term. It is
	// defined by the cubic.
	cubicTwo = 2
	// cubicDegree is the cube's 3: the derivative of y^3 is 3 y^2. It is
	// defined by calculus.
	cubicDegree = 3
)

// The runge-kutta step of order four.
const (
	// midpoint is how far along a step the second and third rates are
	// taken: half of it. It is defined by the method.
	midpoint = half
	// rungeKuttaWeights is what the four rates' weights, 1, 2, 2 and 1, sum
	// to, which the step divides by. It is defined by the method.
	rungeKuttaWeights = 6
	// rungeKuttaDouble is the weight of the second and third rates. It is
	// defined by the method.
	rungeKuttaDouble = 2
)

// The mechanics.
const (
	// hamiltonianHalf is the half of H = (1/2) g^{ab} p_a p_b. It is
	// defined by the hamiltonian.
	hamiltonianHalf = half
	// massSquaredPerH is what mu^2 is of H: minus two. It is defined by
	// H = -mu^2 / 2.
	massSquaredPerH = -2
	// gradientFactor is the 2 in the radius gradient's terms, and in the
	// derivative of the scalar's numerator. It is defined by their
	// derivation from the quartic that defines the radius.
	gradientFactor = 2
	// scalarGradientThree is the 3 of the scalar's gradient,
	// (3 A^2 z^2 - r^4) / (r (r^4 + A^2 z^2)). It is defined by the
	// derivative of r^3 / (r^4 + A^2 z^2).
	scalarGradientThree = 3
	// gradientHalf is the half of reach/2 in the momentum's change. It is
	// defined by the hamiltonian's half.
	gradientHalf = half
)
