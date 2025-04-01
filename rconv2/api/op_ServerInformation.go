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
	Players []GetPlayerResponse `json:"Players"`
}

type PlayerPlatform string

type PlayerTeam int
type PlayerRole int

type GetPlayerResponse struct {
	Id                   string         `json:"ID"`
	Platform             PlayerPlatform `json:"Platform"`
	Name                 string         `json:"Name"`
	ClanTag              string         `json:"ClanTag"`
	EpicOnlineServicesId string         `json:"EOSID"`
	Level                int            `json:"Level"`
	Team                 PlayerTeam     `json:"Team"`
	Role                 PlayerRole     `json:"Role"`
	Squad                string         `json:"Platoon"`
	Loadout              string         `json:"Loadout"`
	Kills                int            `json:"Kills"`
	Deaths               int            `json:"Deaths"`
	Score                ScoreData      `json:"ScoreData"`
	Position             WorldPosition  `json:"WorldPosition"`
}

type ScoreData struct {
	Combat    int `json:"COMBAT"`
	Offensive int `json:"Offense"`
	Defensive int `json:"Defense"`
	Support   int `json:"Support"`
}

type WorldPosition struct {
	X int `json:"X"`
	Y int `json:"Y"`
	Z int `json:"Z"`
}
