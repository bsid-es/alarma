package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	b, _ := json.Marshal(&e)
	fmt.Println(string(b))

	clock := mem.NewClock()
	clock.Now = now

	logger := mem.NewClockLogger(clock)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	clock.Run(ctx)
	defer clock.Interrupt()

	logger.Run(ctx)
	defer logger.Interrupt()

	clock.Reload(&e)

	<-ctx.Done()
}
