package api

type SetWarmupTimer struct {
	GameMode     GameMode `json:"GameMode"`
	WarmupLength int32    `json:"WarmupLength"`
}
