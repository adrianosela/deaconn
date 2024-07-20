package deaconn

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/adrianosela/deaconn/deadline"
)

const readBufferSize = 5242880 // 5 MB

type conn struct {
	inner net.Conn

	ctx       context.Context
	ctxCancel context.CancelFunc

	rxMutex    sync.Mutex
	rxData     chan []byte
	rxDeadline deadline.Deadline

	txMutex    sync.Mutex
	txDeadline deadline.Deadline
}

type txResult struct {
	n   int
	err error
}

// WithDeadlines adds support for deadlines to a given net.Conn.
func WithDeadlines(inner net.Conn) net.Conn {
	ctx, ctxCancel := context.WithCancel(context.Background())

	c := &conn{
		inner: inner,

		ctx:       ctx,
		ctxCancel: ctxCancel,

		rxMutex:    sync.Mutex{},
		rxData:     make(chan []byte),
		rxDeadline: deadline.New(),

		txMutex:    sync.Mutex{},
		txDeadline: deadline.New(),
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

			copied := make([]byte, n)
			copy(copied, buf[:n])

			c.rxData <- copied[:n]

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
	c.rxMutex.Lock()
	defer c.rxMutex.Unlock()

	if b == nil {
		return 0, nil
	}

	select {
	// connection closed
	case <-c.ctx.Done():
		return 0, io.EOF
	// deadline exceeded
	case <-c.rxDeadline.Done():
		return 0, os.ErrDeadlineExceeded
	// data available or rxData channel closed
	case data, ok := <-c.rxData:
		if !ok {
			return 0, io.EOF
		}
		// TODO: ensure b is big enough for data and if not
		// write the leftovers to a buffer for the next read
		return copy(b, data), nil
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *conn) Write(b []byte) (n int, err error) {
	c.txMutex.Lock()
	defer c.txMutex.Unlock()

	if b == nil {
		return 0, nil
	}

	copied := make([]byte, len(b))
	copy(copied, b)

	txResultChan := make(chan txResult)
	go func() {
		defer close(txResultChan)

		n, err := io.Copy(c.inner, bytes.NewReader(copied))
		txResultChan <- txResult{int(n), err}
	}()

	select {
	// connection closed
	case <-c.ctx.Done():
		return 0, io.EOF
	// deadline exceeded
	case <-c.txDeadline.Done():
		return 0, os.ErrDeadlineExceeded
	// write completed
	case result := <-txResultChan:
		return result.n, result.err
	}
}

// Close closes the connection.
func (c *conn) Close() error {
	defer close(c.rxData)
	defer c.ctxCancel()
	return c.inner.Close()
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
	select {
	case <-c.ctx.Done():
		return net.ErrClosed
	default:
		c.rxDeadline.Set(t)
		return nil
	}
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *conn) SetWriteDeadline(t time.Time) error {
	select {
	case <-c.ctx.Done():
		return net.ErrClosed
	default:
		c.txDeadline.Set(t)
		return nil
	}
}
