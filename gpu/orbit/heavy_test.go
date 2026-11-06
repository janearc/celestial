//go:build darwin

package orbit_test

import (
	"math/rand/v2"
	"testing"

	"github.com/janearc/celestial/gpu"
	gpuorbit "github.com/janearc/celestial/gpu/orbit"
	"github.com/janearc/celestial/orbit"
)

// load is a benchmark's batch: so many stars over so many steps.
type load struct {
	Stars int
	Steps int
	Seed  int
}

// heavy is the benchmark's batch: so many stars drawn from the picked
// ranges by a seed, so a run is the same run each time.
type heavy struct {
	numbers picked
	draw    *rand.Rand
}

// star is one star drawn from the ranges.
func (batch heavy) star() orbit.State {
	drawn := batch.numbers.Drawn
	x, y, z := batch.triple(drawn.Farthest)
	vx, vy, vz := batch.triple(drawn.Fastest)
	return orbit.StateOf(x, y, z, vx, vy, vz)
}

// triple is three numbers within a size of zero either way.
func (batch heavy) triple(size float64) (float64, float64, float64) {
	return batch.spread(size), batch.spread(size), batch.spread(size)
}

// spread is a number within a size of zero either way.
func (batch heavy) spread(size float64) float64 {
	return (batch.draw.Float64() - half) * size * halves
}

// The spread's numbers, each defined.
const (
	half   = 0.5 // defined: the midpoint of the unit interval
	halves = 2   // defined: the two halves of it, either side of zero
)

// run runs a load on the gpu as many times as the benchmark asks, and
// reports the gpu's own time for one run.
func run(bench *testing.B, numbers picked, given load) {
	bench.Helper()
	made, err := gpuorbit.Open(gpu.Metal(), centre(numbers), given.Stars)
	if gpu.NoDevice(err) {
		bench.Skip(err)
	}
	if err != nil {
		bench.Fatal(err)
	}
	defer made.Close()
	seed := uint64(given.Seed)
	batch := heavy{numbers, rand.New(rand.NewPCG(seed, seed))}
	stars := make([]orbit.State, given.Stars)
	for index := range stars {
		stars[index] = batch.star()
	}
	span := orbit.Units(numbers.Drawn.LongestSpan)
	bench.ResetTimer()
	for range bench.N {
		if _, _, err := made.Back(stars, span, given.Steps); err != nil {
			bench.Fatal(err)
		}
	}
	bench.StopTimer()
	bench.ReportMetric(made.GPUTime().Seconds()/float64(bench.N),
		"gpu-s/run")
}

// a million stars over a thousand steps on the gpu, and sixteen times
// that: how long the gpu spends, from which the whole catalogue's run
// is reckoned; testdata's about says how the two compared when they
// were written. run with
// go test -run none -bench Heavy -benchtime 1x ./gpu/orbit/
func BenchmarkAHeavyBatch(bench *testing.B) {
	numbers := pickedNumbers(bench)
	run(bench, numbers, numbers.Heavy)
}

func BenchmarkAHeavierBatch(bench *testing.B) {
	numbers := pickedNumbers(bench)
	run(bench, numbers, numbers.Heavier)
}
