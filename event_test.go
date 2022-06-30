package alarma_test

import (
	"testing"
	"time"

	"bsid.es/alarma"
)

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name  string
		event alarma.Event
	}{{
		name: "until and count set simultaneously",
		event: alarma.Event{
			Every: 1 * time.Minute,
			Until: time.Now(),
			Count: 1,
		},
	}, {
		name: "until before start",
		event: alarma.Event{
			At:    time.Date(2021, 12, 21, 0, 0, 0, 0, time.UTC),
			Every: 1 * time.Minute,
			Until: time.Date(2021, 12, 20, 0, 0, 0, 0, time.UTC),
		},
	}, {
		name: "same start and until",
		event: alarma.Event{
			At:    time.Date(2021, 12, 21, 0, 0, 0, 0, time.UTC),
			Every: 1 * time.Minute,
			Until: time.Date(2021, 12, 21, 0, 0, 0, 0, time.UTC),
		},
	}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.event.Validate(); err == nil {
				t.Error("expected error")
			} else if got, want := alarma.ErrorCode(err), alarma.ErrInvalid; got != want {
				t.Errorf("wrong error code\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

var (
	refAt          = time.Date(2012, 12, 21, 0, 0, 0, 0, time.UTC)
	refEvery       = 10 * time.Minute
	refFirstStart  = refAt
	refSecondStart = refAt.Add(refEvery)
	refThirdStart  = refSecondStart.Add(refEvery)
)

func TestEventInstances(t *testing.T) {
	type params struct {
		name             string
		from, curr, next time.Time
	}
	tests := []struct {
		name   string
		event  alarma.Event
		params []params
	}{{
		name: "event with single instance",
		event: alarma.Event{
			At: refAt,
		},
		params: []params{{
			name: "before single instance",
			from: refFirstStart.Add(-1),
			curr: time.Time{},
			next: refFirstStart,
		}, {
			name: "start of single instance",
			from: refFirstStart,
			curr: refFirstStart,
			next: time.Time{},
		}},
	}, {
		name: "infinitely recurring event",
		event: alarma.Event{
			At:    refAt,
			Every: refEvery,
		},
		params: []params{{
			name: "before first instance",
			from: refFirstStart.Add(-1),
			curr: time.Time{},
			next: refFirstStart,
		}, {
			name: "start of first instance",
			from: refFirstStart,
			curr: refFirstStart,
			next: refSecondStart,
		}, {
			name: "start of second instance",
			from: refSecondStart,
			curr: refSecondStart,
			next: refThirdStart,
		}},
	}, {
		name: "recurring event with limit date",
		event: alarma.Event{
			At:    refAt,
			Every: refEvery,
			Until: refSecondStart.Add(1),
		},
		params: []params{{
			name: "before first instance",
			from: refFirstStart.Add(-1),
			curr: time.Time{},
			next: refFirstStart,
		}, {
			name: "start of first instance",
			from: refFirstStart,
			curr: refFirstStart,
			next: refSecondStart,
		}, {
			name: "start of second instance",
			from: refSecondStart,
			curr: refSecondStart,
			next: time.Time{},
		}},
	}, {
		name: "recurring event with limit count",
		event: alarma.Event{
			At:    refAt,
			Every: refEvery,
			Count: 2,
		},
		params: []params{{
			name: "before first instance",
			from: refFirstStart.Add(-1),
			curr: time.Time{},
			next: refFirstStart,
		}, {
			name: "start of first instance",
			from: refFirstStart,
			curr: refFirstStart,
			next: refSecondStart,
		}, {
			name: "start of second instance",
			from: refSecondStart,
			curr: refSecondStart,
			next: time.Time{},
		}},
	}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.event.Validate(); err != nil {
				t.Fatal(err)
			}
			for _, params := range tt.params {
				params := params
				t.Run(params.name, func(t *testing.T) {
					gotCurr := tt.event.Current(params.from)
					gotNext := tt.event.Next(params.from)
					if wantCurr := params.curr; gotCurr != wantCurr {
						t.Errorf("wrong result for current instance\ngot:  %v\nwant: %v", gotCurr, wantCurr)
					}
					if wantNext := params.next; gotNext != wantNext {
						t.Errorf("wrong result for next instance\ngot:  %v\nwant: %v", gotNext, wantNext)
					}
				})
			}
		})
	}
}

func BenchmarkEventCurrent(b *testing.B) {
	benchmarkEventInstance(b, (*alarma.Event).Current)
}

func BenchmarkEventNext(b *testing.B) {
	benchmarkEventInstance(b, (*alarma.Event).Next)
}

func benchmarkEventInstance(b *testing.B, f func(*alarma.Event, time.Time) time.Time) {
	b.Helper()
	event := alarma.Event{
		At:    refAt,
		Every: refEvery,
		Count: 2,
	}
	if err := event.Validate(); err != nil {
		b.Fatal(err)
	}
	tests := []struct {
		name string
		from time.Time
	}{{
		name: "before first instance",
		from: event.At.Add(-1),
	}, {
		name: "start of first instance",
		from: event.At,
	}, {
		name: "start of second instance",
		from: event.At.Add(event.Every),
	}}
	for _, tt := range tests {
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f(&event, tt.from)
			}
		})
	}
}
