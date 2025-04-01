package api

import (
	"fmt"
	"slices"
)

type ServerInformationName string

const (
	ServerInformationNamePlayers      = "players"
	ServerInformationNamePlayer       = "player"
	ServerInformationNameMapRotation  = "maprotation"
	ServerInformationNameMapSequence  = "mapsequence"
	ServerInformationNameSession      = "session"
	ServerInformationNameServerConfig = "serverconfig"

	PlayerPlatformSteam = PlayerPlatform("steam")
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

func (s ServerInformation) CommandName() string {
	return "ServerInformation"
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
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type GetServerConfigResponse struct {
	ServerName         string   `json:"serverName"`
	Build              string   `json:"buildNumber"`
	BuildRevision      string   `json:"buildRevision"`
	SupportedPlatforms []string `json:"supportedPlatforms"`
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
