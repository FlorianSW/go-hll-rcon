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

type GetPlayerResponse struct {
	Id                   string         `json:"id"`
	Platform             PlayerPlatform `json:"platform"`
	Name                 string         `json:"name"`
	ClanTag              string         `json:"clanTag"`
	EpicOnlineServicesId string         `json:"eOSId"`
	Level                int            `json:"level"`
	Team                 string         `json:"team"`
	Role                 string         `json:"role"`
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
