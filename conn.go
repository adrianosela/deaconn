package deaconn

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const readBufferSize = 5242880 // 5 MB

type conn struct {
	inner net.Conn

	ctx       context.Context
	ctxCancel context.CancelFunc

	readTimer         *time.Timer
	readTimerChange   chan struct{}
	readDataAvailable chan []byte
}

// WithDeadlines adds support for deadlines to a given net.Conn.
func WithDeadlines(inner net.Conn) net.Conn {
	ctx, ctxCancel := context.WithCancel(context.Background())

	// initialize readTimer to a Stop()-ed timer
	// (a timer that is non-nil but will never fire).
	readTimer := time.NewTimer(time.Hour)
	readTimer.Stop()

	c := &conn{
		inner: inner,

		ctx:       ctx,
		ctxCancel: ctxCancel,

		readTimer:         readTimer,
		readTimerChange:   make(chan struct{}),
		readDataAvailable: make(chan []byte, 1000),
	}

	go c.continouslyReadIntoBuffer()

	return c
}

func (c *conn) continouslyReadIntoBuffer() {
	defer c.Close()

	buf := make([]byte, readBufferSize)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			n, err := c.inner.Read(buf)
			if n > 0 {
				copied := make([]byte, n)
				copy(copied, buf[:n])
				c.readDataAvailable <- copied[:n]
			} else {
				c.readDataAvailable <- []byte{}
			}
			if err != nil {
				return
			}

		}
	}
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *conn) Read(b []byte) (int, error) {
	for {
		select {
		case data := <-c.readDataAvailable:
			// TODO: ensure b is big enough for data, e.g. write leftovers to buffer
			return copy(b, data), nil
		case <-c.readTimerChange:
			continue
		case <-c.readTimer.C:
			return 0, os.ErrDeadlineExceeded
		case <-c.ctx.Done():
			return 0, io.EOF
		}
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *conn) Write(b []byte) (n int, err error) {
	return c.inner.Write(b)
}

// Close closes the connection.
func (c *conn) Close() error {
	// close inner connection first
	if err := c.inner.Close(); err != nil {
		return err
	}

	// cancel all outstanding reads from the buffer
	c.ctxCancel()

	// drain the timer if needed
	if c.readTimer != nil {
		if !c.readTimer.Stop() {
			<-c.readTimer.C
		}
	}

	// close channels
	close(c.readTimerChange)
	close(c.readDataAvailable)

	return nil
}

// LocalAddr returns the local network address, if known.
func (c *conn) LocalAddr() net.Addr {
	return c.inner.LocalAddr()
}

// RemoteAddr returns the remote network address, if known.
func (c *conn) RemoteAddr() net.Addr {
	return c.inner.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *conn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *conn) SetReadDeadline(t time.Time) error {
	// stop and drain the current timer if not nil
	if c.readTimer != nil {
		if !c.readTimer.Stop() {
			select {
			case <-c.readTimer.C:
				// drain channel if needed
			default:
				// avoid blocking if channel is empty
			}
		}
	}
	// if the deadline is non zero, start a new timer
	if !t.IsZero() {
		c.readTimer = time.NewTimer(time.Until(t))
		c.readTimerChange <- struct{}{}
	}
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *conn) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("write deadlines not supported")
}
