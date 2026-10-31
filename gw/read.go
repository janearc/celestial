package gw

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/janearc/celestial/sky"
)

// The keys the centre's event format gives an event, and the suffixes it
// writes beside a value. Each is the format's own.
const (
	nameKey       = "commonName"
	versionKey    = "version"
	catalogKey    = "catalog.shortName"
	gpsKey        = "GPS"
	parametersKey = "parameters"
	pipelineKey   = "pipeline_type"
	preferredKey  = "is_preferred"
	estimation    = "pe"
	lowerSuffix   = "_lower"
	upperSuffix   = "_upper"
	unitSuffix    = "_unit"
)

// fields is an object of the centre's json, each value as it was written.
type fields map[string]json.RawMessage

// read decodes the field of a key into a place, leaving the place as it
// was when the key is absent or null.
func (object fields) read(key string, into any) error {
	written, held := object[key]
	if !held {
		return nil
	}
	if err := json.Unmarshal(written, into); err != nil {
		return fmt.Errorf("%w: %s: %w", errEvents, key, err)
	}
	return nil
}

// readEvent is one event of the centre's json, its estimates first from the
// event's own preferred values and then from its preferred parameter
// estimation, for any the first does not give.
func readEvent(key string, object fields) (*Event, error) {
	event := Event{}
	var seconds *float64
	for name, into := range map[string]any{nameKey: &event.name,
		versionKey: &event.version, catalogKey: &event.catalog,
		gpsKey: &seconds} {
		if err := object.read(name, into); err != nil {
			return nil, err
		}
	}
	if event.name == "" || seconds == nil {
		return nil, fmt.Errorf("%w: %s has no name or gps time",
			errEvents, key)
	}
	at, err := sky.FromGPS(*seconds)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", errEvents, key, err)
	}
	event.at = *at
	if event.estimates, err = readEstimates(object); err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if err := event.estimatedFrom(object); err != nil {
		return nil, err
	}
	return &event, nil
}

// estimatedFrom adds, from the event's preferred parameter estimation,
// every estimate the event's own values do not give.
func (event *Event) estimatedFrom(object fields) error {
	var pipelines map[string]fields
	if err := object.read(parametersKey, &pipelines); err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		var kind string
		var preferred bool
		if err := pipeline.read(pipelineKey, &kind); err != nil {
			return err
		}
		if err := pipeline.read(preferredKey, &preferred); err != nil {
			return err
		}
		if kind != estimation || !preferred {
			continue
		}
		more, err := readEstimates(pipeline)
		if err != nil {
			return fmt.Errorf("%s: %w", event, err)
		}
		for name, estimate := range more {
			if _, held := event.estimates[name]; !held {
				event.estimates[name] = estimate
			}
		}
	}
	return nil
}

// readEstimates is every estimate an object gives: each value that has a
// unit beside it, with the bounds beside it too.
func readEstimates(object fields) (map[string]*Estimate, error) {
	estimates := map[string]*Estimate{}
	for key := range object {
		parameter, valued := strings.CutSuffix(key, unitSuffix)
		if !valued {
			continue
		}
		estimate, err := readEstimate(object, parameter)
		if err != nil {
			return nil, err
		}
		if estimate != nil {
			estimates[parameter] = estimate
		}
	}
	return estimates, nil
}

// nothing is no reach at all. Defined by the scale.
const nothing = 0.0

// readEstimate is a parameter's estimate in si, from its value, its
// bounds and its unit, or none when the centre left the value blank.
//
// The centre writes each bound as how far from the value it lies, the
// lower one below nothing. readEstimate refuses a value with no unit, a
// unit it has no conversion for, and a bound on the wrong side, and then
// gives none.
func readEstimate(object fields, parameter string) (*Estimate, error) {
	var value, lower, upper *float64
	var unit *string
	for key, into := range map[string]any{parameter: &value,
		parameter + lowerSuffix: &lower,
		parameter + upperSuffix: &upper,
		parameter + unitSuffix:  &unit} {
		if err := object.read(key, into); err != nil {
			return nil, err
		}
	}
	if value == nil {
		return nil, nil
	}
	if unit == nil {
		return nil, fmt.Errorf("%w: %s has a value and no unit",
			errEvents, parameter)
	}
	conversion, err := toSI(*unit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", parameter, err)
	}
	estimate := Estimate{unit: conversion.unit,
		value: *value * conversion.factor}
	if lower == nil || upper == nil {
		return &estimate, nil
	}
	if *lower > nothing || *upper < nothing {
		return nil, fmt.Errorf("%w: %s has bounds %v and %v", errEvents,
			parameter, *lower, *upper)
	}
	estimate.below, estimate.above = -*lower*conversion.factor,
		*upper*conversion.factor
	estimate.bounded = true
	return &estimate, nil
}
