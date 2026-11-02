# The hole API

A map of the current public API. Constructors and accessors replace the
public fields and short method names found in some older examples.
Use `go doc -all ./hole` from the repository root for signatures and
full contracts.

## Geometry and vectors

| Name | Purpose |
|---|---|
| `Vec3`, `NewVec3` | Spatial position or direction; components through `AlongX`, `AlongY`, `AlongZ` |
| `Vec4`, `NewVec4` | Spacetime position or vector; `Time`, spatial component accessors, and `Space` |
| `Mat4` | Metric or inverse, indexed in `(t, x, y, z)` order |
| `Dot` | Contract a covector with a contravariant vector |
| `Hole`, `NewHole` | Validated mass M and spin A, exposed through `Mass` and `Spin` |

`Hole.Spin()` is A = J/M in geometric length units. The dimensionless spin
is A/M. Coordinates use a z spin axis and signature `(-, +, +, +)`.

## Metric and characteristic radii

| Hole method | Result |
|---|---|
| `Radius` | Kerr–Schild radius at a position |
| `Metric`, `Inverse` | Covariant and contravariant metric |
| `Lower`, `Raise` | Convert vector components with the local metric |
| `Horizon` | Outer horizon radius |
| `PhotonSphere` | Schwarzschild reference radius, 3M |
| `PhotonOrbit` | Prograde or retrograde equatorial photon orbit |
| `Shadow` | Schwarzschild shadow impact parameter, √27 M |
| `ISCO` | Prograde or retrograde innermost stable circular orbit |

`PhotonSphere` and `Shadow` retain their nonrotating reference meanings
when called on a spinning hole. They do not describe a complete Kerr
shadow boundary.

## Geodesics and conserved quantities

`Ray` stores a spacetime position and covariant momentum. `NewRay` constructs
it; `Position`, `Momentum`, and `Space` expose the state.

| Hole method | Purpose |
|---|---|
| `Hamiltonian` | Half the inverse metric contracted with momentum twice |
| `Energy` | Conserved energy, minus time momentum |
| `Lz` | Conserved angular momentum about z |
| `Carter` | Carter constant |
| `Velocity` | Spacetime derivative along the affine parameter |
| `Step` | One fourth-order Runge–Kutta step at a caller-supplied size |
| `PaceIn` | Ray-dependent step choice used by `March` |
| `Hovering` | Test the incoming-ray stopping condition near the horizon |
| `March` | Integrate to escape, capture, or exhausted steps, returning `Fate` |
| `ImpactParameter` | Total angular momentum divided by energy in the distant limit |
| `ClosestApproach` | Smallest sampled radius during integration |
| `DeflectionAngle` | Angle between initial and escaped spatial directions |
| `ClassifyRay` | Classify by a march as escaping, plunging, or circulating |

`Fate` records how integration stopped; `RayClassification` is the broader
classification. Read the method contracts before interpreting a step-budget
failure as a physical orbit.

## Observers and frames

`Frame` is an orthonormal tetrad. `Observer`, `XAxis`, `YAxis`, and `ZAxis`
return its four vectors in that order.

| Hole method | Purpose |
|---|---|
| `Static` | Four-velocity of an observer held at fixed coordinates |
| `Normal` | Four-velocity normal to constant-time slices, with zero angular momentum |
| `Circular` | Prograde circular motion at the supplied position |
| `FrameOf` | Construct an observer’s tetrad |
| `Null` | Outgoing photon from a local frame direction |
| `Arriving` | Incoming light prepared for backwards camera tracing |
| `Seen` | Recover the apparent local sky direction |
| `Falling` | Drop a unit-mass body from rest on the x axis |
| `Dropped` | Drop a body from rest at an arbitrary position |
| `Advance` | Integrate a body by proper time within the supported chart |

The standalone `Shift` function returns photon energy at the eye divided
by photon energy at the emitter. It takes endpoint momenta and observer
four-velocities, not just one momentum evaluated twice.

## APIs moving to Slayer

The march and its rendering choices are moving to Slayer. Avoid building
new dependencies on `PaceIn`, `Hovering`, `March`, `ClosestApproach`,
`DeflectionAngle`, `Advance`, `ClassifyRay`, `Fate`, or `RayClassification`.

The shader adapter and its supporting types are part of that move too.
These APIs are still implemented here; see the [roadmap](todo.md).

## Shader adapter

`NewGPUBlackHole` constructs a `GPUBlackHole`; `SetCamera` installs a
stationary observer. `ErrNothingStill()` identifies locations where that
camera cannot exist. `Hole` returns the underlying geometry.

`StepRay` updates one ray on the CPU and returns a `RayStep` diagnostic.
`StepRaysBatch` does the same sequentially for a slice, reusing report
storage when capacity permits. There is no GPU dispatch in this adapter.

`RayStep` exposes `Position`, `Momentum`, `Redshift`, `Energy`,
`AngularMomentum`, `CarterConstant`, `Hamiltonian`, and `Radius` for
comparison with shader calculations.

`ShaderParams` exposes float32 geometry and camera constants: `Mass`,
`Spin`, `Horizon`, `PhotonSphere`, `Shadow`, `CameraPosition`, and
`CameraFrame`. `Bytes` serializes the 96-byte little-endian layout.

## Implementation map

`hole.go` contains the metric and radii; `geodesic.go` holds invariants and
capture geometry; `rates.go` evaluates Hamilton’s equations and stepping.
`frame.go` and `frame_normal.go` handle observers; `march.go` implements
integration policy.

`numbers.go` records formula-derived constants,
`choices.go` loads chosen tolerances, and `vec.go` supplies vector helpers.

This map intentionally avoids listing every private helper as a client API.
The [roadmap](todo.md) describes the intended split between physical
primitives and rendering-specific marching.
