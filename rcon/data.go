package rcon

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
