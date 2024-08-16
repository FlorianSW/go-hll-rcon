package api

import (
	"context"
	"github.com/floriansw/go-hll-rcon/rcon"
)

var (
	SquadTypeCommander = SquadType("Commander")
	SquadTypeRecon     = SquadType("Recon")
	SquadTypeArmor     = SquadType("Armor")
	SquadTypeInfantry  = SquadType("Infantry")
)

type RConPool interface {
	GetWithContext(ctx context.Context) (*rcon.Connection, error)
	WithConnection(ctx context.Context, f func(c *rcon.Connection) error) error
	Return(c *rcon.Connection, err error)
}

type TeamView map[string]*Team

type Score struct {
	CombatEffectiveness int `json:"combat_effectiveness"`
	Offensive           int `json:"offensive"`
	Defensive           int `json:"defensive"`
	Support             int `json:"support"`
}

func (s *Score) Merge(o Score) {
	s.Defensive += o.Defensive
	s.Support += o.Support
	s.Offensive += o.Offensive
	s.CombatEffectiveness += o.CombatEffectiveness
}

func fromRconScore(s rcon.Score) Score {
	return Score{
		CombatEffectiveness: s.CombatEffectiveness,
		Offensive:           s.Offensive,
		Defensive:           s.Defensive,
		Support:             s.Support,
	}
}

type Unit struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Player struct {
	Name      string `json:"name"`
	SteamId64 string `json:"steam_id_64"`
	Team      string `json:"team"`
	Role      string `json:"role"`
	Loadout   string `json:"loadout"`
	Unit      Unit   `json:"unit"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Score     Score  `json:"score"`
	Level     int    `json:"level"`
}

func fromPlayerInfo(s rcon.PlayerInfo) Player {
	return Player{
		Name:      s.Name,
		SteamId64: s.SteamId64,
		Team:      s.Team,
		Role:      s.Role,
		Loadout:   s.Loadout,
		Unit: Unit{
			Id:   s.Unit.Id,
			Name: s.Unit.Name,
		},
		Kills:  s.Kills,
		Deaths: s.Deaths,
		Score:  fromRconScore(s.Score),
		Level:  s.Level,
	}
}

type Team struct {
	Squads         map[string]*Squad `json:"squads"`
	NoSquadPlayers []Player          `json:"no_squad_players"`
	Commander      *Squad            `json:"commander"`
	Score          Score             `json:"score"`
}

type SquadType string

type Squad struct {
	Type    SquadType `json:"type"`
	Score   Score     `json:"score"`
	Players []Player  `json:"players"`
}

type GameScore struct {
	Axis   int `json:"axis"`
	Allies int `json:"allies"`
}

type PlayerCount struct {
	Axis   int `json:"axis"`
	Allies int `json:"allies"`
}

type ServerInfo struct {
	Name          string      `json:"name"`
	Map           string      `json:"map"`
	NextMap       string      `json:"next_map"`
	PlayerCount   int         `json:"player_count"`
	Players       PlayerCount `json:"players"`
	MaxPlayers    int         `json:"max_players"`
	RemainingTime string      `json:"remaining_time"`
	GameScore     GameScore   `json:"game_score"`
}
