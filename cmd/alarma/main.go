package main

import (
	"context"
	"log"
	"time"

	"bsid.es/alarma"
	"bsid.es/alarma/mem"
)

func now() time.Time {
	return time.Now().In(time.UTC)
}

func main() {
	e := alarma.Event{
		Name:  "event",
		At:    now().Add(3 * time.Second).Truncate(time.Second),
		Every: 200 * time.Millisecond,
		Count: 10,
	}
	if err := e.Validate(); err != nil {
		panic(err)
	}

	clock := mem.NewClock()
	clock.Now = now

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	clock.Run(ctx)
	defer clock.Interrupt()

	clock.RegisterFunc(ctx, func(ctx context.Context, alarm alarma.Alarm) {
		log.Printf("Run %q at %v\n", alarm.Event, alarm.At)
	})

	clock.Reload(ctx, &e)

	<-ctx.Done()
}
