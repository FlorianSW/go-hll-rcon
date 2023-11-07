package log_loop

import (
	"fmt"
	"time"
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
	Rest      string
}

func (l *StructuredLogLine) String() string {
	return fmt.Sprintf("%#v", l)
}
