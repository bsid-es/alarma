package mem

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"bsid.es/alarma"
)

type Clock struct {
	Now func() time.Time

	newEvents chan []*alarma.Event

	mu   sync.Mutex
	subs map[chan<- alarma.Alarm]struct{}

	cancel context.CancelFunc
}

func NewClock() *Clock {
	return &Clock{
		Now:       time.Now,
		newEvents: make(chan []*alarma.Event, 1),
		subs:      make(map[chan<- alarma.Alarm]struct{}),
		cancel:    func() {},
	}
}

func (c *Clock) Run(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	go c.run(ctx)
	return nil
}

func (c *Clock) Interrupt() error {
	c.cancel()
	return nil
}

var _ alarma.Clock = (*Clock)(nil)

func (c *Clock) Reload(ctx context.Context, events ...*alarma.Event) {
	select {
	case <-c.newEvents:
	default:
	}
	select {
	case <-ctx.Done():
	case c.newEvents <- events:
	}
}

const subBufferSize = 16

func (c *Clock) Register(ctx context.Context, h alarma.Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub := make(chan alarma.Alarm, subBufferSize)
	c.subs[sub] = struct{}{}
	go handleAlarm(ctx, h, sub)
}

func handleAlarm(ctx context.Context, handler alarma.Handler, sub <-chan alarma.Alarm) {
	for {
		select {
		case <-ctx.Done():
			return

		case alarm, ok := <-sub:
			if !ok {
				return
			}
			handler.HandleAlarm(ctx, alarm)
		}
	}
}

func (c *Clock) RegisterFunc(ctx context.Context, f alarma.HandlerFunc) {
	c.Register(ctx, alarma.Handler(f))
}

func (c *Clock) run(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		for sub := range c.subs {
			delete(c.subs, sub)
			close(sub)
		}
	}()

	var q schedQueue

	timer := time.NewTimer(1<<63 - 1)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done(): // Operation was canceled.
			return

		case events := <-c.newEvents:
			// There are new events. Rebuild queue and schedule next event.
			now := c.Now()

			// Rebuild scheduling queue.
			q = make(schedQueue, 0, len(events))
			for _, event := range events {
				next := event.Next(now)
				if next.IsZero() {
					continue
				}
				q = append(q, schedQueueEntry{
					at:    next,
					event: event,
				})
			}
			heap.Init(&q)

			// Schedule next event.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if len(q) > 0 {
				at := q[0].at
				now := c.Now()
				timer.Reset(at.Sub(now))
			}

		case <-timer.C:
			fire := &q[0]

			now := c.Now()
			at := fire.at
			if now.Before(at) {
				// Time drift. Sleep again.
				timer.Reset(at.Sub(now))
				continue
			}

			event := fire.event
			c.publish(ctx, event, at)

			if next := event.Next(at); !next.IsZero() {
				// There's another instance to run. Schedule it.
				fire.at = next
				heap.Fix(&q, 0)
			} else {
				// Event is finished. Drop it.
				heap.Pop(&q)
			}

			if len(q) > 0 {
				at = fire.at
				now = c.Now()
				timer.Reset(at.Sub(now))
			}
		}
	}
}

func (c *Clock) publish(ctx context.Context, event *alarma.Event, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	alarm := alarma.Alarm{
		Event: event.Name,
		At:    at,
		Data:  event.Data,
	}
	for sub := range c.subs {
		select {
		case <-ctx.Done():
			return
		case sub <- alarm:
		}
	}
}

type schedQueueEntry struct {
	at    time.Time
	event *alarma.Event
}

type schedQueue []schedQueueEntry

var _ heap.Interface = (*schedQueue)(nil)

func (q schedQueue) Len() int {
	return len(q)
}

func (q schedQueue) Less(i, j int) bool {
	ti, tj := q[i].at, q[j].at
	return ti.Before(tj)
}

func (q schedQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *schedQueue) Push(x any) {
	*q = append(*q, x.(schedQueueEntry))
}

func (q *schedQueue) Pop() any {
	old := *q
	n := len(old)
	it := old[n-1]
	old[n-1] = schedQueueEntry{}
	*q = old[:n-1]
	return it
}
