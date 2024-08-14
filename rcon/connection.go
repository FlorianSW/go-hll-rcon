package rcon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	} else {
		return c.socket.con.SetDeadline(time.Now().Add(20 * time.Second))
	}
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
	re := regexp.MustCompile("(?m)^(\\[.+? \\(\\d+\\)])")
	for {
		// HLL RCon does not indicate the length of data returned for the command, instead we need to read as long as
		// we do not get any data anymore. For that we loop through read() until there is no data to be received anymore.
		// Unfortunately when the server does not have data anymore, it simply does not return anything (other than
		// EOF e.g.).
		next, err := c.continueRead(c.Context())
		if errors.Is(err, os.ErrDeadlineExceeded) {
			lines := split(re, r, -1)
			var result []string
			for _, line := range lines {
				if line == "" {
					continue
				}
				result = append(result, strings.ReplaceAll(line, "\n", " "))
			}
			return result, nil
		} else if err != nil {
			return nil, err
		}
		r += string(next)
	}
}

func split(re *regexp.Regexp, s string, n int) []string {
	if n == 0 {
		return nil
	}

	matches := re.FindAllStringIndex(s, n)
	str := make([]string, 0, len(matches))

	beg := 0
	end := 0
	p := ""
	for _, match := range matches {
		if n > 0 && len(str) >= n-1 {
			break
		}

		end = match[0]
		if match[1] != 0 {
			str = append(str, p+s[beg:end])
		}
		beg = match[1]
		p = s[match[0]:match[1]]
	}

	if end != len(s) {
		str = append(str, p+s[beg:])
	}

	return str
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

// AdminIds returns a list of admins on the server. An admin is identified by their Steam ID, and an optional name/comment.
// The role is a potentially custom-defined role that can have a set of permissions. Use AdminGroups to get a list of
// available roles for this server.
func (c *Connection) AdminIds() ([]AdminId, error) {
	v, err := c.ListCommand("get adminids")
	if err != nil {
		return nil, err
	}
	var result []AdminId
	for _, s := range v {
		parts := strings.SplitN(s, " ", 3)
		result = append(result, AdminId{
			Name:      strings.ReplaceAll(parts[2], "\"", ""),
			SteamId64: parts[0],
			Role:      parts[1],
		})
	}
	return result, nil
}

// AdminGroups returns a list of potentially customized roles that are available for this server. A role can have a
// specific set of permissions and can be used in AddAdmin command.
func (c *Connection) AdminGroups() ([]string, error) {
	return c.ListCommand("get admingroups")
}

// AddAdmin adds a player to the list of admins with the specified role. The role needs to be a role available on the
// server. Use the AdminGroups command to get available roles.
func (c *Connection) AddAdmin(id AdminId) error {
	_, err := c.Command(fmt.Sprintf("adminadd %s %s %s", id.SteamId64, id.Role, id.Name))
	return err
}

// DeleteAdmin removes a player from the list of admins. Correspond to AddAdmin and AdminIds for more details.
func (c *Connection) DeleteAdmin(steamId string) error {
	_, err := c.Command(fmt.Sprintf("admindel %s", steamId))
	return err
}

// ServerName returns the currently set server name
func (c *Connection) ServerName() (string, error) {
	return c.Command("get name")
}

// Slots returns the current number of players connected to the server as the first return value. The second return value
// is the total number of players allowed to be connected to the server at the same time.
func (c *Connection) Slots() (int, int, error) {
	p, err := c.Command("get slots")
	if err != nil {
		return 0, 0, err
	}
	n := strings.Split(p, "/")
	players, _ := strconv.Atoi(n[0])
	maxPlayers, _ := strconv.Atoi(n[1])
	return players, maxPlayers, nil
}

// GameState returns information about the currently played round on the server.
func (c *Connection) GameState() (GameState, error) {
	res := GameState{}
	r, err := c.Command("get gamestate")
	if err != nil {
		return res, err
	}
	lines := strings.Split(r, "\n")
	for _, line := range lines {
		kv := strings.SplitN(line, ": ", 2)
		switch kv[0] {
		case "Map":
			res.Map = kv[1]
		case "Next Map":
			res.NextMap = kv[1]
		case "Remaining Time":
			var h, m, s int
			_, _ = fmt.Sscanf(kv[1], "%d:%d:%d", &h, &m, &s)
			res.RemainingTime = time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second
		case "Players":
			sides := strings.Split(kv[1], " - ")
			for _, side := range sides {
				skv := strings.SplitN(side, ": ", 2)
				if skv[0] == "Allied" {
					res.Players.Allies, _ = strconv.Atoi(skv[1])
				} else if skv[0] == "Axis" {
					res.Players.Axis, _ = strconv.Atoi(skv[1])
				}
			}
		case "Score":
			sides := strings.Split(kv[1], " - ")
			for _, side := range sides {
				skv := strings.SplitN(side, ": ", 2)
				if skv[0] == "Allied" {
					res.Score.Allies, _ = strconv.Atoi(skv[1])
				} else if skv[0] == "Axis" {
					res.Score.Axis, _ = strconv.Atoi(skv[1])
				}
			}
		}
	}
	return res, nil
}

// MapFilter A filter used in commands that return list of maps, e.g. Maps or MapRotation.
// The filter should return true, when the map should be included in the result set and false
// when the map should be skipped.
type MapFilter func(idx int, name string, result []string) bool

// Maps Returns the available maps on the server. These map names can be used in commands like SwitchMap
// and AddToMapRotation
func (c *Connection) Maps(filters ...MapFilter) ([]string, error) {
	maps, err := c.ListCommand("get mapsforrotation")

	return filter(maps, filters...), err
}

func filter(maps []string, filters ...MapFilter) []string {
	var result []string
	for i, m := range maps {
		add := true
		for _, filter := range filters {
			if !filter(i, m, result) {
				add = false
			}
		}
		if add {
			result = append(result, m)
		}
	}
	return result
}

// SwitchMap Changes the map on the server. The map name must be one that is available on the server.
// You can get the available maps with the Maps function.
// If the map is not in the map rotation, yet, then it will be added to the Map Rotation.
func (c *Connection) SwitchMap(mapName string) error {
	_, err := c.Command(fmt.Sprintf("map %s", mapName))
	if errors.Is(err, CommandFailed) {
		err = c.addToMapRotation(mapName)
		if err != nil {
			return err
		}
		_, err = c.Command(fmt.Sprintf("map %s", mapName))
	}

	return err
}

func (c *Connection) addToMapRotation(mapName string) error {
	maps, err := c.MapRotation()
	if err != nil {
		return err
	}
	return c.AddToMapRotation(mapName, maps[len(maps)-1])
}

// MapRotation Returns a list of map names, which are currently in the map rotation.
// Maps can be duplicated in the list.
func (c *Connection) MapRotation(filters ...MapFilter) ([]string, error) {
	mapsString, err := c.Command("rotlist")
	if err != nil {
		return nil, err
	}
	maps := strings.Split(mapsString, "\n")
	return filter(maps[:len(maps)-1], filters...), err
}

// AddToMapRotation Adds a map to the map rotation after the mentioned map.
func (c *Connection) AddToMapRotation(mapName string, afterMap string) error {
	_, err := c.Command(fmt.Sprintf("rotadd /Game/Maps/%s /Game/Maps/%s", mapName, afterMap))
	return err
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

// MessagePlayer sends a message to the player with the steam ID, if the player is on the server. The message
// can contain newline characters.
func (c *Connection) MessagePlayer(steamId string, message string) error {
	_, err := c.Command(fmt.Sprintf("message %s %s", steamId, message))
	return err
}
