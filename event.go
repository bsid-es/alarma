package alarma

import (
	"errors"
	"time"
)

type Event struct {
	Name  string         `json:"name"`
	At    time.Time      `json:"at"`
	Every time.Duration  `json:"every"`
	Until time.Time      `json:"until"`
	Count uint           `json:"count"`
	Data  map[string]any `json:"data"`

	last time.Time
}

func (e *Event) Validate() (err error) {
	switch {
	case e.Every < 0:
		return errors.New("every must be non-negative")
	case e.Every > 0 && !e.Until.IsZero() && e.Count != 0:
		return errors.New("until and count are mutually exclusive")
	case e.Every > 0 && !e.Until.IsZero() && !e.Until.After(e.At):
		return errors.New("until must happen after at")
	}

	// Compute when the last instance runs.
	switch {
	case e.Every == 0:
		e.last = e.At

	case e.Every > 0 && !e.Until.IsZero():
		num := e.instanceNumber(e.Until)
		e.last = e.instance(num)

	case e.Every > 0 && e.Count != 0:
		num := time.Duration(e.Count) - 1
		e.last = e.instance(num)
	}

	return nil
}

func (e *Event) Current(from time.Time) time.Time {
	switch {
	case from.Before(e.At):
		// Event didn't start yet.
		return time.Time{}
	case !e.last.IsZero() && !from.Before(e.last):
		// Event is finished.
		return e.last.In(from.Location())
	}
	num := e.instanceNumber(from)
	return e.instance(num).In(from.Location())
}

func (e *Event) Next(from time.Time) time.Time {
	switch {
	case from.Before(e.At):
		// Event didn't start yet.
		return e.At.In(from.Location())
	case !e.last.IsZero() && !from.Before(e.last):
		// Event is finished.
		return time.Time{}
	}
	num := e.instanceNumber(from) + 1
	return e.instance(num).In(from.Location())
}

func (e *Event) instanceNumber(from time.Time) time.Duration {
	return from.Sub(e.At) / e.Every
}

func (e *Event) instance(num time.Duration) time.Time {
	return e.At.Add(num * e.Every)
}
