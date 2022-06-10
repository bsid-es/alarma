package mem

import (
	"context"
	"log"

	"bsid.es/alarma"
)

type ClockLogger struct {
	clock alarma.Clock

	sub    alarma.ClockSubscription
	cancel context.CancelFunc
}

func NewClockLogger(clock alarma.Clock) *ClockLogger {
	return &ClockLogger{
		clock: clock,
	}
}

func (l *ClockLogger) Run(ctx context.Context) error {
	l.sub = l.clock.Subscribe(ctx)
	ctx, l.cancel = context.WithCancel(ctx)
	go l.run(ctx)
	return nil
}

func (l *ClockLogger) Interrupt() error {
	l.cancel()
	return nil
}

func (l *ClockLogger) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			l.sub.Close()
			return

		case alarm, ok := <-l.sub.C():
			if !ok {
				l.sub = l.clock.Subscribe(ctx)
				continue
			}
			log.Printf("Run %q at %v\n", alarm.Event, alarm.At)

		}
	}
}
