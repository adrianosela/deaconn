# deaconn

Band-aid for `net.Conn` implementations that don't support deadlines.

[![Go Report Card](https://goreportcard.com/badge/github.com/adrianosela/deaconn)](https://goreportcard.com/report/github.com/adrianosela/deaconn)
[![Documentation](https://godoc.org/github.com/adrianosela/deaconn?status.svg)](https://godoc.org/github.com/adrianosela/deaconn)
[![GitHub issues](https://img.shields.io/github/issues/adrianosela/deaconn.svg)](https://github.com/adrianosela/deaconn/issues)
[![license](https://img.shields.io/github/license/adrianosela/deaconn.svg)](https://github.com/adrianosela/deaconn/blob/master/LICENSE)

<table border="0" cellpadding="0" cellspacing="0">
  <tr>
    <td>
      <img src="./assets/img/deacon_gopher.png" width="250">
    </td>
    <td>
        <p><b>Deaconn</b> stands for "deadline connection".</p>
        <p>For usage, see below...</p>
    </td>
  </tr>
</table>


## Motivation

#### Short Version:

If an `http.Server`'s underlying `net.Conn`s do not support deadline methods, WebSockets and any other feature which relies on "hijacking" connections will not work.

#### Long Version:

This is a small project that was born out of need... I needed to support HTTP traffic over a `net.Conn` implementation which did not support the `SetDeadline`, `SetReadDeadline`, and `SetWriteDeadline` methods.

This is a problem because the `net/http` package relies on the deadline methods of `net.Conn` to support certain features.

Specifically, the `net/http` package uses `SetReadDeadline()` to set a connection's deadline to a time in the past in order to abort pending/blocked `Read()`s on it.

An example where this is done is when type-asserting an `http.ResponseWriter` as an `http.Hijacker` and then invoking [`Hijack()`](https://github.com/golang/go/blob/3959d54c0bd5c92fe0a5e33fedb0595723efc23b/src/net/http/server.go#L2153-L2176) on it (as is done to handle a WebSocket upgrade for instance).

The `net.Conn` implementation that I specifically was using (which lacked deadlines support) was an `(golang.org/x/crypto/ssh).Channel`.

More Details:

- There's an [open GitHub issue in the Golang GitHub repo (#65930)](https://github.com/golang/go/issues/65930) that tracks the lack of deadline methods support in the `ssh.Channel`
- There's an [open GitHub issue in the Golang GitHub repo (#67152)](https://github.com/golang/go/issues/67152) that tracks the specific issue I faced (i.e. `Hijack()` hanging for WebSocket upgrades when HTTP is served over an `ssh.Channel`)
- There's a an [open Gerrit patch set in the Golang Gerrit repo](https://go-review.googlesource.com/c/crypto/+/562756) that I have personally tested to solve my specific use case effectively

## Usage

To wrap a `net.Listener` in a listener that adds deadlines support to `net.Conn`s:

```
wrappedListener := deaconn.NewListenerWithDeadlines(listener)
defer wrappedListener()

// use the wrappedListener as you would use your original listener
```

To wrap a single `net.Conn` to add deadline support to it:

```
wrappedConn := deaconn.NewConnWithDeadlines(conn)

// use the wrappedConn as you would use your original conn
```
