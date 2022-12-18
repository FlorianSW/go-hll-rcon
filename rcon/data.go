package rcon

import "time"

type PlayerId struct {
	Name      string
	SteamId64 string
}

type Unit struct {
	Id   int
	Name string
}

type Score struct {
	CombatEffectiveness int
	Offensive           int
	Defensive           int
	Support             int
}

type PlayerInfo struct {
	Name      string
	SteamId64 string
	Team      string
	Role      string
	Loadout   string
	Unit      Unit
	Kills     int
	Deaths    int
	Score     Score
	Level     int
}

type PlayerCount struct {
	Axis   int
	Allies int
}

type GameScore struct {
	Axis   int
	Allies int
}

type GameState struct {
	Players       PlayerCount
	Score         GameScore
	RemainingTime time.Duration
	Map           string
	NextMap       string
}
