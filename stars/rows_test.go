//ninja:vow dojo each row is held to being found at its own mass, and the main sequence between rows to its logarithms, through newDwarf, newStar and lifetime, which stars does not export
package stars

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

// rows is the main sequence, lightest first, or the test stops.
func rows(test *testing.T) []dwarf {
	test.Helper()
	table, err := mainSequence()
	if err != nil {
		test.Fatal(err)
	}
	return table
}

// At a row's own mass the main sequence is that row, to the last bit:
// its light and its radius, for every row, the lightest and the heaviest
// among them, and nothing worked out between rows.
func TestARowsOwnMassIsThatRow(test *testing.T) {
	table := rows(test)
	for _, row := range table {
		found, err := newDwarf(table, row.mass)
		if err != nil {
			test.Errorf("%s at its own mass: %v", row.kind, err)
			continue
		}
		if found.light != row.light || found.radius != row.radius {
			test.Errorf("%s at its own mass is %v and %v, not %v and %v",
				row.kind, found.light, found.radius, row.light, row.radius)
		}
	}
}

// Between two rows the main sequence runs straight in the logarithms: a
// star at the geometric mean of their masses has the geometric mean of
// their light and of their radius, to a few roundings, for every two
// rows side by side.
func TestBetweenRowsTheLogarithmsRunStraight(test *testing.T) {
	table := rows(test)
	share := readTableNumbers(test).BetweenTol
	for place, upper := range table[oneRow:] {
		lower := table[place]
		if !(upper.mass.Over(lower.mass) > whole) {
			continue
		}
		mass := measure.SolarMasses(math.Sqrt(lower.mass.InSolarMasses() *
			upper.mass.InSolarMasses()))
		found, err := newDwarf(table, mass)
		if err != nil {
			test.Fatalf("between %s and %s: %v", lower.kind, upper.kind, err)
		}
		brighter := math.Sqrt(upper.light.Over(lower.light))
		wider := math.Sqrt(upper.radius.Over(lower.radius))
		if got := found.light.Over(lower.light); !(math.Abs(got-brighter) <=
			share*brighter) {
			test.Errorf("between %s and %s the light is %v of the lower, "+
				"not %v", lower.kind, upper.kind, got, brighter)
		}
		if got := found.radius.Over(lower.radius); !(math.Abs(got-wider) <=
			share*wider) {
			test.Errorf("between %s and %s the radius is %v of the lower, "+
				"not %v", lower.kind, upper.kind, got, wider)
		}
	}
}

// A star exactly as old as its burning lasts has ended: the end of a part
// of its life is the start of the next.
func TestAStarAsOldAsItsBurningHasEnded(test *testing.T) {
	for _, born := range rows(test) {
		if star := newStar(born, lifetime(born)); star.Life() != Ended() {
			test.Errorf("%s at the end of its burning is %v", born.kind,
				star.Life())
		}
	}
}

// The anti-test: a moment younger than its burning lasts, a star still
// burns.
func TestAStarAMomentYoungerStillBurns(test *testing.T) {
	younger := whole - readTableNumbers(test).Younger
	for _, born := range rows(test) {
		age := measure.Seconds(lifetime(born).InSeconds() * younger)
		if star := newStar(born, age); star.Life() != Burning() {
			test.Errorf("%s a moment before the end of its burning is %v",
				born.kind, star.Life())
		}
	}
}
