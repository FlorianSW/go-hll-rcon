package rconv2

import (
	"errors"
	"fmt"
)

const (
	PlayerPlatformSteam = PlayerPlatform("steam")
)

var (
	ErrInvalidCredentials = errors.New("wrong password")
)

type UnexpectedStatus struct {
	code    int
	message string
}

func NewUnexpectedStatus(code int, message string) *UnexpectedStatus {
	return &UnexpectedStatus{
		code:    code,
		message: message,
	}
}

func (u UnexpectedStatus) Error() string {
	return fmt.Sprintf("invalid status code received, got %d with message %s", u.code, u.message)
}

type Command interface {
	CommandName() string
}

type GetPlayersResponse struct {
	Players []Player `json:"players"`
}

type PlayerPlatform string

type Player struct {
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

func newConnectionRequestTimeout(currentPoolSize int) connectionRequestTimeout {
	return connectionRequestTimeout{
		openConnections: currentPoolSize,
	}
}

type connectionRequestTimeout struct {
	openConnections int
}

func (c connectionRequestTimeout) Error() string {
	return fmt.Sprintf("connection request timed out before a connection was available. Open connections: %d", c.openConnections)
}

func (c connectionRequestTimeout) Timeout() bool {
	return true
}
