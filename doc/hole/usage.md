# Using hole

Construct the hole and vectors through their current APIs. For a stationary
observer outside the ergosurface, trace incoming light backwards from the
camera:

```go
h, err := hole.NewHole(1, 0.6)
if err != nil {
    return err
}
x := hole.NewVec3(10, 0, 0)
n := hole.NewVec3(-1, 0, 0)
eye := h.Static(x)
frame := h.FrameOf(x, eye)
ray := h.Arriving(x, frame, n)
end, fate := h.March(ray, 100, 10000)
// Inspect fate before using the endpoint as an escaped ray.
```

This is an outline inside a function returning an error. Here `farRadius`
is 100 and the step budget is 10,000; they are example choices, not
accuracy guarantees.

`Static` returns nonfinite values where an observer
cannot remain stationary. Choose an admissible observer before building
a frame; `Normal` also works in the ergosphere outside the horizon.

`Arriving` negates the incoming photon momentum so the integration can
advance its parameter while moving backwards in coordinate time. Use
`Null` for outgoing light instead; frame dragging makes the two paths
different around a spinning hole.

For a ray connecting an emitter and an eye, compute the energy ratio with
momenta and four-velocities at the corresponding endpoints:

```go
gain := hole.Shift(ray.Momentum(), eye, end.Momentum(), emitter)
```

`emitter` is the emitting observer’s four-velocity at the endpoint. A ratio
above one means a blueshift; below one means a redshift. It includes
relative motion as well as gravity. In general, the momentum at one end
cannot stand in for the momentum at the other.

`March` and its fates are moving to Slayer, together with the shader
adapter. Celestial will retain geodesic stepping at a caller-chosen size.
See the [roadmap](todo.md) before building new dependencies on the march.

## Conventions to keep close

The package uses geometric units, a z spin axis, covariant momentum,
and the `(-, +, +, +)` signature. `Hole.Spin()` is A = J/M, in the same
length units as M, rather than the dimensionless A/M.

`March` may stop before the horizon once an inward ray cannot return.
Its returned state is where integration stopped, not necessarily a horizon
intersection. Check `Fate` rather than inferring capture from radius alone.

`GPUBlackHole` is a CPU stepping and shader-layout adapter. It does not
execute GPU work. Its 96-byte `ShaderParams.Bytes()` layout is a contract
with the shader; the tests check that contract.
