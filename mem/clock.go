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
	subs map[*ClockSubscription]struct{}

	cancel context.CancelFunc
}

func NewClock() *Clock {
	return &Clock{
		Now:       time.Now,
		newEvents: make(chan []*alarma.Event, 1),
		subs:      make(map[*ClockSubscription]struct{}),
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

func (c *Clock) Reload(events ...*alarma.Event) {
	select {
	case <-c.newEvents:
	default:
	}
	c.newEvents <- events
}

const subBufferSize = 16

func (c *Clock) Subscribe(ctx context.Context) alarma.ClockSubscription {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub := &ClockSubscription{
		clock: c,
		c:     make(chan alarma.ClockAlarm, subBufferSize),
	}
	c.subs[sub] = struct{}{}
	return sub
}

func (c *Clock) run(ctx context.Context) {
	var q schedQueue

	timer := time.NewTimer(1<<63 - 1)
	timer.Stop()

	for {
		select {
		case <-ctx.Done(): // Operation was canceled.
			timer.Stop()
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
				<-timer.C
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
			c.publish(event, at)

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

func (c *Clock) publish(event *alarma.Event, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	alarm := alarma.ClockAlarm{
		Event: event.Name,
		At:    at,
		Data:  event.Data,
	}
	for sub := range c.subs {
		select {
		case sub.c <- alarm:
		default:
			// Drop subscription.
			// FIXME(fmrsn): Explain why.
			sub.close()
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

var _ alarma.ClockSubscription = (*ClockSubscription)(nil)

type ClockSubscription struct {
	clock *Clock
	c     chan alarma.ClockAlarm
	once  sync.Once
}

func (sub *ClockSubscription) C() <-chan alarma.ClockAlarm {
	return sub.c
}

func (sub *ClockSubscription) Close() error {
	sub.clock.mu.Lock()
	defer sub.clock.mu.Unlock()
	sub.close()
	return nil
}

func (sub *ClockSubscription) close() {
	sub.once.Do(func() {
		close(sub.c)
	})
	delete(sub.clock.subs, sub)
}
