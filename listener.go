package deaconn

import "net"

// listener is a net.Listener implementation which adds
// deadlines support to every accepted net.Conn.
type listener struct {
	inner net.Listener
}

// NewListenerWithDeadlines returns a net.Listener implementation
// which adds deadlines support to every accepted net.Conn.
func NewListenerWithDeadlines(inner net.Listener) net.Listener {
	return &listener{inner: inner}
}

// Accept waits for and returns the next connection to the listener.
func (l *listener) Accept() (net.Conn, error) {
	innerConn, err := l.inner.Accept()
	if err != nil {
		return nil, err
	}
	return WithDeadlines(innerConn), nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *listener) Close() error {
	return l.inner.Close()
}

// Addr returns the listener's network address.
func (l *listener) Addr() net.Addr {
	return l.inner.Addr()
}
