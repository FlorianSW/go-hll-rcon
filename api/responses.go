package api

import (
	"encoding/json"
	"github.com/floriansw/go-hll-rcon/rcon"
)

func renderPlayerIds(pids []rcon.PlayerId) map[string]interface{} {
	var result []map[string]interface{}
	for _, pid := range pids {
		result = append(result, renderPlayerId(pid))
	}

	return map[string]interface{}{
		"player_ids": result,
	}
}

func renderPlayerId(pid rcon.PlayerId) map[string]interface{} {
	return map[string]interface{}{
		"name":        pid.Name,
		"steam_id_64": pid.SteamId64,
	}
}

func renderPlayerInfo(pid rcon.PlayerInfo) map[string]interface{} {
	return map[string]interface{}{
		"name":        pid.Name,
		"steam_id_64": pid.SteamId64,
		"team":        pid.Team,
		"role":        pid.Role,
		"loadout":     pid.Loadout,
		"unit":        renderUnit(pid.Unit),
		"score":       renderScore(pid.Score),
		"kills":       pid.Kills,
		"deaths":      pid.Deaths,
		"level":       pid.Level,
	}
}

func renderScore(s rcon.Score) map[string]interface{} {
	return map[string]interface{}{
		"combat_effectiveness": s.CombatEffectiveness,
		"offensive":            s.Offensive,
		"defensive":            s.Defensive,
		"support":              s.Support,
	}
}

func renderUnit(u rcon.Unit) map[string]interface{} {
	return map[string]interface{}{
		"id":   u.Id,
		"name": u.Name,
	}
}

func asJson(c interface{}) []byte {
	r, _ := json.Marshal(c)
	return r
}
