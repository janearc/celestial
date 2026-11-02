package gw

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/celestial/sky"
)

// Event holds one versioned catalogue event: name, catalogue, GPS time, and
// estimates under the centre’s original parameter names.
type Event struct {
	name, catalog string
	version       int
	at            sky.Instant
	estimates     map[string]*Estimate
}

// String returns the centre’s event key, such as GW150914-v3.
func (event *Event) String() string {
	return fmt.Sprintf("%s-v%d", event.name, event.version)
}

// Catalog returns the centre’s short catalogue name.
func (event *Event) Catalog() string { return event.catalog }

// At returns the event time as a GPS-scale instant.
func (event *Event) At() sky.Instant { return event.at }

// Parameters returns estimated parameter names in sorted order.
func (event *Event) Parameters() []string {
	return slices.Sorted(maps.Keys(event.estimates))
}

// Estimator supplies estimates by the centre’s parameter names. Event
// implements it; callers can use the generated mock to test estimate handling
// separately from catalogue parsing.
type Estimator interface {
	Estimate(parameter string) (*Estimate, error)
}

// Event must stay an Estimator: the build fails here if it stops being
// one.
var _ Estimator = &Event{}

// Estimate looks up a reported parameter. A blank or absent value returns an
// error rather than an invented zero.
func (event *Event) Estimate(parameter string) (*Estimate, error) {
	if estimate, held := event.estimates[parameter]; held {
		return estimate, nil
	}
	return nil, fmt.Errorf("gw: %s has no estimate of %q", event,
		parameter)
}

// ReadEvents parses the centre’s event-API JSON and returns events in key
// order. It rejects malformed or empty event objects, missing names or GPS
// times, unsupported units, and bounds on the wrong side of their estimates.
// Invalid GPS times retain the sky conversion error.
func ReadEvents(raw []byte) ([]*Event, error) {
	var file struct {
		Events map[string]map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("gw: not the centre's events: %w", err)
	}
	if len(file.Events) == nothingRead {
		return nil, errors.New("gw: no events in the centre's json")
	}
	var events []*Event
	for _, key := range slices.Sorted(maps.Keys(file.Events)) {
		event, err := readEvent(key, file.Events[key])
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// nothingRead is a count of nothing. Defined by counting.
const nothingRead = 0
