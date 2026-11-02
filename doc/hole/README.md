# hole

I have a little ray tracer, and I think gravity is pretty cool, so I keep
ending up here: black holes, bent light, and cameras falling into places
you would not put an actual camera. Mostly, this is an exercise in maths.

The package covers:

- Kerr geometry in Kerr–Schild coordinates.
- Null and timelike geodesics from Hamilton’s equations.
- Conserved quantities, their numerical drift, and redshift between observers.
- A falling camera’s frame—because you *want* to see.

G = c = 1; mass sets the length scale. Coordinates are `(t, x, y, z)`,
momentum is covariant, the signature is `(-, +, +, +)`, and spin is along z.

Start with [usage](usage.md), consult the [API map](definitions.md), and
read the [test guide](tests.md) for the checks behind it. The
[roadmap](todo.md) separates current code from planned changes.

## Author

Jane Michelle Arc · jmich@fastmail.nl

i am not a potato
