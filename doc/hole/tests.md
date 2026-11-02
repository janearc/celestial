# Checking the black-hole maths

The tests are the working specification. They check known radii and
orbits, conserved quantities, observer frames, redshift, and the numerical
behaviour of the march.

From the repository root, run the package with two processors:

```sh
GOWORK=off GOMAXPROCS=2 go test ./hole
```

`GOWORK=off` keeps an unrelated parent workspace from selecting the wrong
module. For the whole repository, use `game check`.

## What the tests cover

- Horizons, photon orbits, the Schwarzschild shadow at √27 M, and ISCOs.
- Weak-field deflection, including approximately 1.75 arcseconds at the Sun.
- Hamiltonian, energy, axial angular momentum, and Carter constant.
- Step refinement, geodesic paths, and energy-independent photon trajectories.
- Static, circular, normal, and falling observers, including frames where
  a static observer cannot exist.
- Incoming-ray round trips, horizon crossing, shadow capture, and viewing
  from inside the horizon.
- Orbital motion, fall timing, redshift, and the march’s stopping rules.
- The shader parameter layout and CPU reference stepping.

Expected values and their provenance live in `hole/testdata/hole.json`.
Chosen integration tolerances live separately in `hole/choices.json`;
do not present those choices as measured physical constants.

## Finding a check

`metric_test.go`, `geodesic_test.go`, and `rates` implementation comments
cover the geometric foundation. `frame_test.go`, `observers_test.go`, and
`frame_normal` cover local observers. The march, fate, approach, deflection,
fall, shift, GPU, and shader test files cover their named behaviours.

## Property and mutation tests

The property tests use Rapid, with a hundred cases by default. To draw
more cases:

```sh
GOWORK=off go test ./hole/ -rapid.checks 1000
```

Game runs mutation testing. It changes one operator, number, or guard at
a time and runs the tests against each change. A surviving mutation
shows behaviour the tests have not distinguished.

```sh
game mutants --workers 2 --only hole
```
