package deadline

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	new := New()

	asserted, ok := new.(*deadline)
	if !ok {
		t.Error("expected New to return a deadline implementation of the Deadline interface")
	}

	select {
	case <-asserted.done:
		t.Error("expected New to return a deadline with the done channel open but was closed")
	default:
		// fallthrough
	}

	if asserted.timer != nil {
		t.Error("expected New to return a deadline with a nil timer but was non-nil")
	}
}

func TestDone(t *testing.T) {
	d := &deadline{
		mutex: sync.Mutex{},
		done:  make(chan struct{}),
	}
	if d.Done() != d.done {
		t.Error("expected channel returned by Done() to refer to the same channel as d.done")
	}
}

func TestSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		deadline *deadline

		beforeFunc func(*deadline)

		t time.Time

		timerNilAfter   bool
		doneClosedAfter bool
	}{
		// zero-time
		{
			name: "setting zero-time on deadline with timer nil and channel open --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			t:               time.Time{},
			timerNilAfter:   true,
			doneClosedAfter: false,
		},
		{
			name: "setting zero-time on deadline with timer nil and channel closed --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Time{},
			timerNilAfter:   true,
			doneClosedAfter: false,
		},
		{
			name: "setting zero-time on deadline with timer non-nil and channel open --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			t:               time.Time{},
			timerNilAfter:   true,
			doneClosedAfter: false,
		},
		{
			name: "setting zero-time on deadline with timer non-nil and channel closed --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Time{},
			timerNilAfter:   true,
			doneClosedAfter: false,
		},
		// time in the past
		{
			name: "setting negative time on deadline with timer nil and channel open --> timer nil and channel closed",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			t:               time.Now().Add(-time.Hour),
			timerNilAfter:   true,
			doneClosedAfter: true,
		},
		{
			name: "setting negative time on deadline with timer nil and channel closed --> timer nil and channel closed",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Now().Add(-time.Hour),
			timerNilAfter:   true,
			doneClosedAfter: true,
		},
		{
			name: "setting negative time on deadline with timer non-nil and channel open --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			t:               time.Now().Add(-time.Hour),
			timerNilAfter:   true,
			doneClosedAfter: true,
		},
		{
			name: "setting negative time on deadline with timer non-nil and channel closed --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Now().Add(-time.Hour),
			timerNilAfter:   true,
			doneClosedAfter: true,
		},
		// time in the future
		{
			name: "setting positive time on deadline with timer nil and channel open --> timer nil and channel closed",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			t:               time.Now().Add(time.Hour),
			timerNilAfter:   false,
			doneClosedAfter: false,
		},
		{
			name: "setting positive time on deadline with timer nil and channel closed --> timer nil and channel closed",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: nil,
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Now().Add(time.Hour),
			timerNilAfter:   false,
			doneClosedAfter: false,
		},
		{
			name: "setting positive time on deadline with timer non-nil and channel open --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			t:               time.Now().Add(time.Hour),
			timerNilAfter:   false,
			doneClosedAfter: false,
		},
		{
			name: "setting positive time on deadline with timer non-nil and channel closed --> timer nil and channel open",
			deadline: &deadline{
				mutex: sync.Mutex{},
				timer: time.NewTimer(time.Until(time.Now().Add(time.Hour))),
				done:  make(chan struct{}),
			},
			beforeFunc:      func(d *deadline) { close(d.done) },
			t:               time.Now().Add(time.Hour),
			timerNilAfter:   false,
			doneClosedAfter: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.beforeFunc != nil {
				test.beforeFunc(test.deadline)
			}
			test.deadline.Set(test.t)

			select {
			case <-test.deadline.done:
				if !test.doneClosedAfter {
					t.Error("expected done channel to be open after setting time but was closed")
				}
			default:
				if test.doneClosedAfter {
					t.Error("expected done channel to be closed after setting time but was open")
				}
			}

			if test.timerNilAfter {
				if test.deadline.timer != nil {
					t.Error("expected timer to be nil after setting time but was non-nil")
				}
			} else {
				if test.deadline.timer == nil {
					t.Error("expected timer to be non-nil after setting time but was nil")
				}
			}

		})
	}

}
