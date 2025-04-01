package rconv2

import (
	"context"
)

// Connection represents a persistent connection to a HLL server using RCon. It can be used to issue commands against
// the HLL server and query data. The connection can either be utilised using the higher-level API methods, or by sending
// raw commands using ListCommand or Command.
//
// A Connection is not thread-safe by default. Do not attempt to run multiple commands (either of the higher-level or
// low-level API). Doing so may either run into non-expected indefinitely blocking execution (until the context.Context
// deadline exceeds) or to mixed up data (sending a command and getting back the response for another command).
// Instead, in goroutines use a ConnectionPool and request a new connection for each goroutine. The ConnectionPool will
// ensure that one Connection is only used once at the same time. It also speeds up processing by opening a number of
// Connections until the pool size is reached.
type Connection struct {
	id     string
	socket *socket
}

func (c *Connection) Players(ctx context.Context) (*GetPlayersResponse, error) {
	err := c.socket.SetContext(ctx)
	if err != nil {
		return nil, err
	}
	req := Request[serverInformationCommand, GetPlayersResponse]{
		Body: serverInformationCommand{
			Name:  "players",
			Value: "",
		},
		Command: "ServerInformation",
	}
	res, err := req.do(c.socket)
	if err != nil {
		return nil, err
	}
	return &res.Content, nil
}
