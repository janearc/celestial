package measure

// degreesPerTurn is how many degrees make a whole turn. Defined by the
// degree.
const degreesPerTurn = 360

// arcsecondsPerDegree is sixty arcminutes of sixty arcseconds. Defined
// by the arcsecond.
const arcsecondsPerDegree = 3600

// Angle is an angle, held in radians, which have no unit.
type Angle struct{ natural float64 }

// Radians is an angle in radians.
func Radians(amount float64) Angle { return Angle{amount} }

// Degrees is an angle in degrees.
func Degrees(amount float64) Angle {
	return Angle{amount * fullTurn / degreesPerTurn}
}

// InRadians is the angle in radians.
func (angle Angle) InRadians() float64 { return angle.natural }

// InDegrees is the angle in degrees.
func (angle Angle) InDegrees() float64 {
	return angle.natural * degreesPerTurn / fullTurn
}

// Arcseconds is an angle in arcseconds, the unit positions and their
// errors on the sky are given in.
func Arcseconds(amount float64) Angle {
	return Degrees(amount / arcsecondsPerDegree)
}

// InArcseconds is the angle in arcseconds.
func (angle Angle) InArcseconds() float64 {
	return angle.InDegrees() * arcsecondsPerDegree
}
