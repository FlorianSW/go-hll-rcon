package rconv2

import (
	"context"
	"github.com/floriansw/go-hll-rcon/rconv2/api"
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

func (c *Connection) Players(ctx context.Context) (*api.GetPlayersResponse, error) {
	return execCommand[api.ServerInformation, api.GetPlayersResponse](ctx, c.socket, api.ServerInformation{
		Name:  "players",
		Value: "",
	})
}

func (c *Connection) Player(ctx context.Context, playerId string) (*api.GetPlayerResponse, error) {
	return execCommand[api.ServerInformation, api.GetPlayerResponse](ctx, c.socket, api.ServerInformation{
		Name:  "player",
		Value: playerId,
	})
}

func (c *Connection) AddAdmin(ctx context.Context, playerId, adminGroup, comment string) error {
	_, err := execCommand[api.AddAdmin, any](ctx, c.socket, api.AddAdmin{
		PlayerId:   playerId,
		AdminGroup: adminGroup,
		Comment:    comment,
	})
	return err
}

func execCommand[T Command, U any](ctx context.Context, so *socket, req T) (result *U, err error) {
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
	body := res.Body()
	return &body, nil
}
