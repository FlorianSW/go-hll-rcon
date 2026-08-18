package rconv2

import (
	"context"
)

// Connection represents a persistent connection to a HLL(V) server using RCon. It can be used to issue commands against
// the server and query data. Connection is a common type that supports commands that are available in both HLL:V and HLL WW2.
// It is recommended to use a game-specific connection type for issuing commands that are game-server specific, such as HLLConnection
// or HLLVConnection. Game-specific connection tyspes inherit from Connection, including the below notices.
//
// A Connection is not thread-safe by default. Do not attempt to run multiple commands in different threads or go-routines.
// Doing so may either run into non-expected indefinitely blocking execution (until the context.Context
// deadline exceeds) or to mixed up data (sending a command and getting back the response for another command).
// Instead, in goroutines, use a ConnectionPool and request a new connection for each goroutine. The ConnectionPool will
// ensure that one Connection is only used once at the same time. It also speeds up processing by opening a number of
// Connections until the pool size is reached.
type Connection struct {
	id     string
	socket *socket
}

func (c *Connection) getId() string {
	return c.id
}

func (c *Connection) getSocket() *socket {
	return c.socket
}

func newConnection(id string, socket *socket) *Connection {
	return &Connection{
		id:     id,
		socket: socket,
	}
}

type connectionFactory[T GameConnection] func(id string, socket *socket) T

func execCommand[T, U any](ctx context.Context, so *socket, req T) (result *U, err error) {
	err = so.SetContext(ctx)
	if err != nil {
		return nil, err
	}
	r := Request[T, U]{
		Body: req,
	}
	res, err := r.do(so)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, NewUnexpectedStatus(res.StatusCode, res.StatusMessage)
	}
	return new(res.Body()), nil
}
