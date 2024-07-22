package deadline

import (
	"sync"
	"time"
)

// Deadline represents an entity that has a settable
// deadline and respects the handling of deadlines
// as per the net.Conn interface documentation.
type Deadline interface {
	Done() chan struct{}
	Set(t time.Time)
}

// deadline is the default Deadline implementation.
type deadline struct {
	mutex sync.Mutex
	timer *time.Timer
	done  chan struct{}
}

// New returns a new instance of the default Deadline implementation.
func New() Deadline {
	return &deadline{
		mutex: sync.Mutex{},
		timer: nil,
		done:  make(chan struct{}),
	}
}

// Done returns a channel which fires whenever a deadline is exceeded.
func (d *deadline) Done() chan struct{} {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.done
}

// Set sets a new time for the deadline.
func (d *deadline) Set(t time.Time) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.timer != nil {
		// stop the timer and
		// drain the channel
		if !d.timer.Stop() {
			<-d.done
		}
		d.timer = nil
	}

	doneIsClosed := false
	select {
	case <-d.done:
		doneIsClosed = true
	default:
		// default clause to ensure we don't block
	}

	// handle the zero-time case as per the net.Conn spec:
	// a zero value for means I/O operations will not time out
	if t.IsZero() {
		if doneIsClosed {
			d.done = make(chan struct{})
		}
		return
	}

	duration := time.Until(t)

	// handle t in the past (fire immediately)
	if duration <= 0 {
		if !doneIsClosed {
			close(d.done)
		}
		return
	}

	// handle t in the future (fire after ttl)
	if doneIsClosed {
		d.done = make(chan struct{})
	}
	d.timer = time.AfterFunc(duration, func() { close(d.done) })
}
