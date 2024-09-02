package log_loop

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	tR = regexp.MustCompile("\\((\\d+)\\)")
	pC = regexp.MustCompile("(CONNECTED|DISCONNECTED) (.+) \\((\\d+)\\)")
	kR = regexp.MustCompile("KILL: (.+)\\((Axis|Allies)/(\\d+)\\) -> (.+)\\((Axis|Allies)/(\\d+)\\) with (.+)")
	cR = regexp.MustCompile("CHAT\\[(Team|Unit)]\\[(.*)\\((Allies|Axis)/(.*)\\)]: (.*)")
)

func ParseLogLine(line string) (StructuredLogLine, error) {
	res := StructuredLogLine{
		Raw: line,
	}
	p := strings.SplitN(line, "] ", 2)
	r := p[1]
	tS := tR.FindStringSubmatch(p[0])
	if len(tS) != 2 {
		return res, fmt.Errorf("could not parse timestamp, expected 1 match, got: %d", len(tS)-1)
	}
	tI, err := strconv.ParseInt(tS[1], 10, 64)
	if err != nil {
		return res, err
	}
	t := time.Unix(tI, 0)
	res.Timestamp = t

	if strings.HasPrefix(r, ActionDisconnected) || strings.HasPrefix(r, ActionConnected) {
		p = pC.FindStringSubmatch(r)
		res.Action = p[1]
		res.Actor.Name = p[2]
		res.Actor.SteamId64 = p[3]
	} else if strings.HasPrefix(r, fmt.Sprintf("%s: ", ActionKill)) {
		p = kR.FindStringSubmatch(r)
		res.Action = ActionKill
		res.Actor.Name = p[1]
		res.Actor.Team = strings.ToLower(p[2])
		res.Actor.SteamId64 = p[3]
		res.Subject.Name = p[4]
		res.Subject.Team = strings.ToLower(p[5])
		res.Subject.SteamId64 = p[6]
		res.Weapon = p[7]
	} else if strings.HasPrefix(r, fmt.Sprintf("%s[", ActionChat)) {
		p = cR.FindStringSubmatch(r)
		println(p)
		res.Action = ActionChat
		res.Actor.Name = p[2]
		res.Actor.Team = strings.ToLower(p[3])
		res.Actor.SteamId64 = p[4]
		res.Message = p[5]
		res.Rest = p[1]
	}

	return res, nil
}
