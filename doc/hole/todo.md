# Work ahead

## Thin lens planes

Far from an individual hole, a thin plane of mass can model the deflection
of passing light. Galaxies and clusters make arcs and Einstein rings;
a passing star can briefly brighten a background source in microlensing.

Useful first models are a point mass, a singular isothermal sphere, and a
mass sheet for the surrounding field. Each maps a position on the lens
plane to a deflection. A renderer could query them as it queries Kerr
geometry, without implementing the same lens model in each scene.

## Separate the physics from rendering policy

The package currently includes both geodesic stepping and higher-level
marching. `Step` advances by a caller-supplied interval; `March` chooses
steps and decides when a ray escapes or is captured.

The march is ray tracing and is moving to Slayer, together with its
choices and `gpu.go`. Celestial will retain the physics a renderer needs:

- One geodesic step at a size the caller chooses.
- The radius at a position.
- The horizon and the capture boundary beyond which a ray cannot return.
- The vector arithmetic used by a step.

The move includes `March`, `PaceIn`, `Hovering`, `ClosestApproach`,
`DeflectionAngle`, `Fate`, `Advance`, `ClassifyRay`, and the shader adapter.
They are still implemented here, but new callers should plan around the
smaller stepping API.
