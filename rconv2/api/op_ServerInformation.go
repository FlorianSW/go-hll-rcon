package api

import (
	"fmt"
	"math"
	"slices"
)

type ServerInformationName string

type SupportedPlatform string

const (
	ServerInformationNamePlayers      = "players"
	ServerInformationNamePlayer       = "player"
	ServerInformationNameMapRotation  = "maprotation"
	ServerInformationNameMapSequence  = "mapsequence"
	ServerInformationNameSession      = "session"
	ServerInformationNameServerConfig = "serverconfig"

	PlayerPlatformSteam = PlayerPlatform("steam")

	SupportedPlatformSteam   = "Steam"
	SupportedPlatformWindows = "WinGDK"
	SupportedPlatformEos     = "eos"
)

const (
	PlayerTeamGer = iota
	PlayerTeamUs
	PlayerTeamRus
	PlayerTeamGb
	PlayerTeamDak
	PlayerTeamB8a
)

const (
	PlayerRoleRifleman = iota
	PlayerRoleAssault
	PlayerRoleAutomaticRifleman
	PlayerRoleMedic
	PlayerRoleSpotter
	PlayerRoleSupport
	PlayerRoleHeavyMachineGunner
	PlayerRoleAntiTank
	PlayerRoleEngineer
	PlayerRoleOfficer
	PlayerRoleSniper
	PlayerRoleCrewman
	PlayerRoleTankCommander
	PlayerRoleArmyCommander
)

var (
	requiresValue = []ServerInformationName{
		ServerInformationNamePlayer,
	}
)

type ServerInformation struct {
	Name  ServerInformationName `json:"Name"`
	Value string                `json:"Value"`
}

func (s ServerInformation) Validate() error {
	if slices.Contains(requiresValue, s.Name) && s.Value == "" {
		return fmt.Errorf("%s command requires a Value", s.Name)
	}
	return nil
}

type GetPlayersResponse struct {
	Players []GetPlayerResponse `json:"players"`
}

type PlayerPlatform string

type PlayerTeam int
type PlayerRole int

type GetPlayerResponse struct {
	Id                   string         `json:"iD"`
	Platform             PlayerPlatform `json:"platform"`
	Name                 string         `json:"name"`
	ClanTag              string         `json:"clanTag"`
	EpicOnlineServicesId string         `json:"eOSID"`
	Level                int            `json:"level"`
	Team                 PlayerTeam     `json:"team"`
	Role                 PlayerRole     `json:"role"`
	Squad                string         `json:"platoon"`
	Loadout              string         `json:"loadout"`
	Kills                int            `json:"kills"`
	Deaths               int            `json:"deaths"`
	Score                ScoreData      `json:"scoreData"`
	Position             WorldPosition  `json:"worldPosition"`
}

type ScoreData struct {
	Combat    int `json:"cOMBAT"`
	Offensive int `json:"offense"`
	Defensive int `json:"defense"`
	Support   int `json:"support"`
}

type WorldPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (w WorldPosition) Equal(o WorldPosition) bool {
	return w.X == o.X && w.Y == o.Y && w.Z == o.Y
}

// IsSpawned indicates that the player is currently not on the map, e.g. in the spawn or team selection screen.
func (w WorldPosition) IsSpawned() bool {
	return (w.X + w.Y + w.Z) != 0
}

// Distance calculates the distance of this and another position in the game world. This includes movement on the x axis
// (as represented in changed values of X and Y) as well as on the y axis (represented by changed Z values).
// This is calculated as if the distance was travelled in a straight line without observing obstacles. It depends on the
// resolution of when the two involved positions were obtained how accurate the calculated distance is.
func (w WorldPosition) Distance(o WorldPosition) Distance {
	return Distance(math.Sqrt(math.Pow(w.X-o.X, 2) + math.Pow(w.Y-o.Y, 2) + math.Pow(w.Z-o.Z, 2)))
}

// Distance is supposed to be in centimeters (default unit of worlds in Unreal Engine)
type Distance float64

func (d Distance) Meters() float64 {
	return float64(d) / 100
}

func (d Distance) Add(o Distance) Distance {
	return d + o
}

type GetServerConfigResponse struct {
	ServerName         string              `json:"serverName"`
	Build              string              `json:"buildNumber"`
	BuildRevision      string              `json:"buildRevision"`
	SupportedPlatforms []SupportedPlatform `json:"supportedPlatforms"`
}

type GetSessionResponse struct {
	ServerName       string `json:"serverName"`
	MapName          string `json:"mapName"`
	GameMode         string `json:"gameMode"`
	MaxPlayerCount   int    `json:"maxPlayerCount"`
	PlayerCount      int    `json:"playerCount"`
	MaxQueueCount    int    `json:"maxQueueCount"`
	QueueCount       int    `json:"queueCount"`
	MaxVIPQueueCount int    `json:"maxVIPQueueCount"`
	VIPQueueCount    int    `json:"vIPQueueCount"`
}

type GetMapRotationResponse struct {
	Maps []Map `json:"mAPS"`
}

type GetMapSequenceResponse struct {
	Maps []Map `json:"mAPS"`
}

type Map struct {
	Name      string `json:"name"`
	GameMode  string `json:"gameMode"`
	TimeOfDay string `json:"timeOfDay"`
	Id        string `json:"iD"`
	Position  int    `json:"position"`
}
