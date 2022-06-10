package alarma

import (
	"context"
	"time"
)

type Clock interface {
	Reload(...*Event)
	Subscribe(context.Context) ClockSubscription
}

type ClockSubscription interface {
	// C returns a channel. (wat?)
	//
	// If the subscriber can't keep up with the events coming from this channel,
	// Clock unsubscribes it and closes its channel; in this case, the
	// subscription holder will need to subscribe again.
	C() <-chan ClockAlarm

	// Close closes the subscription.
	Close() error
}

type ClockAlarm struct {
	Event string         `json:"event"`
	At    time.Time      `json:"at"`
	Data  map[string]any `json:"data"`
}
