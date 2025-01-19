package log_loop

import (
	"fmt"
	"time"
)

const (
	ActionKill         = "KILL"
	ActionConnected    = "CONNECTED"
	ActionDisconnected = "DISCONNECTED"
	ActionChat         = "CHAT"
	ActionMatchStart   = "MATCH START"
	ActionMatchEnded   = "MATCH ENDED"
)

type Player struct {
	Name      string
	SteamId64 string
	Team      string
}

type StructuredLogLine struct {
	Raw       string
	Timestamp time.Time
	Action    string
	Actor     Player
	Subject   Player
	Weapon    string
	Message   string
	Result    *MatchResult
	Rest      string
}

type MatchResult struct {
	Axis   int
	Allied int
}

func (l *StructuredLogLine) String() string {
	return fmt.Sprintf("%#v", l)
}
