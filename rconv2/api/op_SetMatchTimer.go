package api

type SetMatchTimer struct {
	GameMode    GameMode `json:"GameMode"`
	MatchLength int32    `json:"MatchLength"`
}
