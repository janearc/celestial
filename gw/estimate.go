package gw

import (
	"fmt"
	"strings"
)

// Estimate is a value the centre publishes for an event, in si units, and its
// credible interval, as the centre states it: how far below and above the
// value the interval runs. the centre gives ninety per cent intervals. A bound
// it leaves blank is unknown.
type Estimate struct {
	unit         string
	value        float64
	below, above float64
	bounded      bool
}

// Value is the estimate, in the si unit Unit names.
func (estimate *Estimate) Value() float64 { return estimate.value }

// Unit is the si unit the estimate is in, as written: kg, m, s^-1, J, W,
// sr, or nothing for a pure number.
func (estimate *Estimate) Unit() string { return estimate.unit }

// Interval is the lowest and highest the credible interval reaches, and
// whether the centre gave both its bounds. When it did not, the interval is
// not known and the two are the value.
func (estimate *Estimate) Interval() (lowest, highest float64, known bool) {
	if !estimate.bounded {
		return estimate.value, estimate.value, false
	}
	return estimate.value - estimate.below, estimate.value + estimate.above,
		true
}

// String is the estimate as the centre writes it: the value, the interval's
// reach below and above, and the unit.
func (estimate *Estimate) String() string {
	written := fmt.Sprintf("%g %s", estimate.value, estimate.unit)
	if estimate.bounded {
		written = fmt.Sprintf("%g -%g +%g %s", estimate.value,
			estimate.below, estimate.above, estimate.unit)
	}
	return strings.TrimSpace(written)
}
