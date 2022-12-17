package rcon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
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

// PlayerIds issues the `get playerids` command to the server and returns a list of parsed PlayerIds. The players returned
// are the ones currently connected to the server.
func (c *Connection) PlayerIds() ([]PlayerId, error) {
	v, err := c.ListCommand("get playerids")
	if err != nil {
		return nil, err
	}
	var result []PlayerId
	for _, s := range v {
		parts := strings.Split(s, " : ")
		result = append(result, PlayerId{
			Name:      parts[0],
			SteamId64: parts[1],
		})
	}
	return result, nil
}

// PlayerInfo returns more information about a specific player by using its name. The player needs to be connected to
// the server for this command to succeed.
func (c *Connection) PlayerInfo(name string) (PlayerInfo, error) {
	// Name: xxxx
	// steamID64: 7656xxxx
	// Team: Allies
	// Role: Assault
	// Unit: 5 - FOX
	// Loadout: Veteran
	// Kills: 0 - Deaths: 7
	// Score: C 0, O 20, D 240, S 0
	// Level: 81
	res := PlayerInfo{}
	v, err := c.Command(fmt.Sprintf("playerinfo %s", name))
	if err != nil {
		return res, err
	}
	lines := strings.Split(v, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		key := parts[0]
		value := parts[1]
		switch key {
		case "Name":
			res.Name = value
		case "steamID64":
			res.SteamId64 = value
		case "Team":
			res.Team = value
		case "Role":
			res.Role = value
		case "Unit":
			up := strings.Split(value, " - ")
			uid, _ := strconv.Atoi(up[0])
			res.Unit = Unit{
				Id:   uid,
				Name: up[1],
			}
		case "Loadout":
			res.Loadout = value
		case "Kills":
			kd := strings.Split(value, " - Deaths: ")
			k, _ := strconv.Atoi(kd[0])
			d, _ := strconv.Atoi(kd[1])
			res.Kills = k
			res.Deaths = d
		case "Score":
			res.Score = Score{}
			score := strings.Split(value, ", ")
			for _, s := range score {
				kv := strings.Split(s, " ")
				sv, _ := strconv.Atoi(kv[1])
				switch kv[0] {
				case "C":
					res.Score.CombatEffectiveness = sv
				case "O":
					res.Score.Offensive = sv
				case "D":
					res.Score.Defensive = sv
				case "S":
					res.Score.Support = sv
				}
			}
		case "Level":
			lvl, _ := strconv.Atoi(value)
			res.Level = lvl
		default:
			continue
		}
	}
	return res, nil
}
