package alarma

import (
	"context"
	"time"
)

type Clock interface {
	Reload(ctx context.Context, events ...*Event)
	Register(ctx context.Context, h Handler)
	RegisterFunc(ctx context.Context, f HandlerFunc)
}

type Handler interface {
	HandleAlarm(ctx context.Context, alarm Alarm)
}

type HandlerFunc func(context.Context, Alarm)

var _ Handler = (HandlerFunc)(nil)

func (f HandlerFunc) HandleAlarm(ctx context.Context, alarm Alarm) {
	f(ctx, alarm)
}

type Alarm struct {
	Event string         `json:"event"`
	At    time.Time      `json:"at"`
	Data  map[string]any `json:"data"`
}
