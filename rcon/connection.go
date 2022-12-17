package rcon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Connection struct {
	id     string
	socket *socket
	parent *context.Context
}

// WithContext inherits applicable values from the given context.Context and applies them to the underlying
// RCon connection. There is generally no need to call this method explicitly, the ConnectionPool (where you usually
// get this Connection from) takes care of propagating the outer context.
//
// However, n cases where you want to have a different context.Context for retrieving a Connection from the ConnectionPool
// and when executing commands, using this method can be useful. One use case might be to have a different timeout while
// waiting for a Connection from the ConnectionPool, as when executing a command on the Connection.
//
// Returns an error if context.Context values could not be applied to the underlying Connection.
func (c *Connection) WithContext(ctx context.Context) error {
	c.parent = &ctx
	if deadline, ok := ctx.Deadline(); ok {
		return c.socket.con.SetDeadline(deadline)
	}
	return nil
}

func (c *Connection) Context() context.Context {
	if c.parent != nil {
		return *c.parent
	}
	return context.Background()
}

// ListCommand executes the raw command provided and returns the result as a list of strings. A list with regard to
// the RCon protocol is delimited by tab characters. The result is split by tab characters to produce the resulting
// list response.
func (c *Connection) ListCommand(cmd string) ([]string, error) {
	return c.socket.listCommand(cmd)
}

// Command executes the raw command provided and returns the result as a plain string.
func (c *Connection) Command(cmd string) (string, error) {
	return c.socket.command(cmd)
}

// ShowLog is a higher-level method to read logs using RCon using the `showlog` raw command. While it would be possible
// to execute `showlog` with Command, it is not recommended to do so. Showlog has a different response size depending
// on the duration from when logs should be returned. As RCon does not provide a way to communicate the length of the
// response data, this method will try to guess if the returned data is complete and reads from the underlying stream
// of data until it has all. This is not the case with Command.
func (c *Connection) ShowLog(d time.Duration) ([]string, error) {
	r, err := c.socket.command(fmt.Sprintf("showlog %0f", d.Minutes()))
	if err != nil {
		return nil, err
	}
	// there is no need to read more data, the server has no logs for the specified timeframe
	if r == "EMPTY" {
		return nil, nil
	}
	for {
		// HLL RCon does not indicate the length of data returned for the command, instead we need to read as long as
		// we do not get any data anymore. For that we loop through read() until there is no data to be received anymore.
		// Unfortunately when the server does not have data anymore, it simply does not return anything (other than
		// EOF e.g.).
		next, err := c.continueRead(c.Context())

		if errors.Is(err, os.ErrDeadlineExceeded) {
			return strings.Split(r, "\n"), nil
		} else if err != nil {
			return nil, err
		}
		r += string(next)
	}
}

func (c *Connection) continueRead(pCtx context.Context) ([]byte, error) {
	// Considering that multiple reads on the same data stream should not have much latency, we assume a pretty low
	// timeout for subsequent reads to reduce the latency for ShowLog.
	ctx, cancel := context.WithDeadline(pCtx, time.Now().Add(50*time.Millisecond))
	defer cancel()
	defer func() { _ = c.WithContext(pCtx) }()
	_ = c.WithContext(ctx)
	next, err := c.socket.read()
	return next, err
}
