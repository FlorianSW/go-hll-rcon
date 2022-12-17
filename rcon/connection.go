package rcon

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Connection struct {
	id     string
	socket *socket
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
	if deadline, ok := ctx.Deadline(); ok {
		return c.socket.con.SetDeadline(deadline)
	}
	return nil
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
	for {
		if r[len(r)-1:] == "\n" {
			return strings.Split(r, "\n"), nil
		}
		next, err := c.socket.read()
		if err != nil {
			return nil, err
		}
		r += string(next)
	}
}
