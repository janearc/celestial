//ninja:vow dojo the cross product is held to one worked by hand, and the angle between two directions to the arccosine, through cross and angleBetween, which hole does not export
package hole

import (
	"math"
	"testing"
)

// The cross product of two directions is the one worked by hand, part by
// part, and the angle between them is the arccosine of their cosine.
func TestTheCrossProductIsWorkedByHand(t *testing.T) {
	given := innerNumbers(t).Cross
	first, second := given.First.space(), given.Second.space()
	if got, want := first.cross(second), given.Product.space(); got != want {
		t.Errorf("%v crossed with %v is %v, not %v", first, second, got,
			want)
	}
	want := math.Acos(given.Dot / math.Sqrt(given.LengthsSquared))
	if got := angleBetween(first, second); !(math.Abs(got-want) <=
		given.Tolerance) {
		t.Errorf("the angle between %v and %v is %v, not %v", first,
			second, got, want)
	}
}

// The anti-test: the cross product turns about when the two are taken the
// other way round, and a direction crossed with itself is nothing.
func TestTheCrossProductTurnsAboutAndIsNothingAlongItself(t *testing.T) {
	given := innerNumbers(t).Cross
	first, second := given.First.space(), given.Second.space()
	product := given.Product.space()
	turned := Vec3{x: -product.x, y: -product.y, z: -product.z}
	if got := second.cross(first); got != turned {
		t.Errorf("%v crossed with %v is %v, not %v", second, first, got,
			turned)
	}
	for _, alone := range []Vec3{first, second} {
		if got := alone.cross(alone); got != (Vec3{}) {
			t.Errorf("%v crossed with itself is %v", alone, got)
		}
	}
}
