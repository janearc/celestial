package gw

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/celestial/sky"
)

// Event is one merger in one of the centre's catalogues:
//   - its name and version
//   - the catalogue it is in
//   - its gps time
//   - every estimate the centre gives for it, each under the centre's
//     own name for the parameter
type Event struct {
	name, catalog string
	version       int
	at            sky.Instant
	estimates     map[string]*Estimate
}

// String is the event's name and version, joined as the centre keys
// them: GW150914-v3.
func (event *Event) String() string {
	return fmt.Sprintf("%s-v%d", event.name, event.version)
}

// Catalog is the short name of the event's catalogue, as the centre
// writes it.
func (event *Event) Catalog() string { return event.catalog }

// At is the event's gps time.
func (event *Event) At() sky.Instant { return event.at }

// Parameters is the name of every parameter the event has an estimate
// for, sorted.
func (event *Event) Parameters() []string {
	return slices.Sorted(maps.Keys(event.estimates))
}

// Estimator gives estimates by the centre's name for each parameter.
// Event is one. A caller that reads estimates takes an Estimator, so its
// tests can stand the generated mock in for an event.
type Estimator interface {
	Estimate(parameter string) (*Estimate, error)
}

// Event must stay an Estimator: the build fails here if it stops being
// one.
var _ Estimator = &Event{}

// Estimate is the event's estimate of a parameter, by the centre's name
// for it. returns nil and error for a parameter the event has none of,
// whether the centre left it blank or never gave it.
func (event *Event) Estimate(parameter string) (*Estimate, error) {
	if estimate, held := event.estimates[parameter]; held {
		return estimate, nil
	}
	return nil, fmt.Errorf("gw: %s has no estimate of %q", event,
		parameter)
}

// ReadEvents is every event in the centre's event json, as its event api
// serves it: an object of events, each keyed by name and version, read
// in the order of their keys. returns nil and error for:
//   - a file that is not that json, or holds no events
//   - an event without its name or gps time, or a gps time sky refuses
//   - a value without a unit, or in a unit gw has no conversion for
//   - a bound on the wrong side of its value
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
