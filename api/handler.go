package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/floriansw/go-hll-rcon/api/response"
	. "github.com/floriansw/go-hll-rcon/api/shiftpath"
	"github.com/floriansw/go-hll-rcon/rcon"
	"net/http"
	"strings"
	"sync"
)

var (
	jr = response.Jr
	b  = response.B
)

type handler struct {
	pool RConPool
}

func NewHandler(p RConPool) *handler {
	return &handler{
		pool: p,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req = ShiftPath(req)

	switch head {
	case "players":
		h.handlePlayers(w, req)
	case "teams":
		h.handleTeams(w, req)
	case "server":
		h.handleServer(w, req)
	default:
		jr.NotFound(w)
	}
}

func (h *handler) handlePlayers(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req = ShiftPath(req)
	switch head {
	case "":
		h.listPlayers(w, req)
	default:
		h.handlePlayer(w, req, head)
	}
}

func (h *handler) listPlayers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jr.MethodNotAllowed(w)
		return
	}
	c, err := h.pool.GetWithContext(req.Context())
	if err != nil {
		jr.InternalServerError(w, b(err.Error()))
		return
	}
	defer h.pool.Return(c)

	r, err := c.PlayerIds()
	if err != nil {
		jr.InternalServerError(w, b(err.Error()))
		return
	}
	jr.Ok(w, asJson(renderPlayerIds(r)))
}

func (h *handler) handlePlayer(w http.ResponseWriter, req *http.Request, name string) {
	if req.Method != http.MethodGet {
		jr.MethodNotAllowed(w)
		return
	}
	c, err := h.pool.GetWithContext(req.Context())
	if err != nil {
		jr.InternalServerError(w, b(err.Error()))
		return
	}
	defer h.pool.Return(c)

	pi, err := c.PlayerInfo(name)
	if errors.Is(err, rcon.CommandFailed) {
		jr.NotFound(w)
		return
	} else if err != nil {
		jr.InternalServerError(w, b(err.Error()))
		return
	}
	jr.Ok(w, asJson(renderPlayerInfo(pi)))
}

func (h *handler) handleServer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jr.MethodNotAllowed(w)
		return
	}

	res := &ServerInfo{}

	wg := sync.WaitGroup{}
	var cErr []error
	wg.Add(3)
	go func(r *ServerInfo) {
		defer wg.Done()
		f := func(c *rcon.Connection) {
			var err error
			r.Name, err = c.ServerName()
			if err != nil {
				cErr = append(cErr, err)
			}
		}

		if err := h.pool.WithConnection(req.Context(), f); err != nil {
			jr.InternalServerError(w, b(err.Error()))
		}
	}(res)
	go func(r *ServerInfo) {
		defer wg.Done()
		f := func(c *rcon.Connection) {
			p, mp, err := c.Slots()
			if err != nil {
				cErr = append(cErr, err)
			}
			res.PlayerCount = p
			res.MaxPlayers = mp
		}

		if err := h.pool.WithConnection(req.Context(), f); err != nil {
			jr.InternalServerError(w, b(err.Error()))
		}
	}(res)
	go func(r *ServerInfo) {
		defer wg.Done()
		f := func(c *rcon.Connection) {
			state, err := c.GameState()
			if err != nil {
				cErr = append(cErr, err)
				return
			}
			r.RemainingTime = state.RemainingTime.String()
			r.Map = state.Map
			r.NextMap = state.NextMap
			r.GameScore.Allies = state.Score.Allies
			r.GameScore.Axis = state.Score.Axis
			r.Players.Allies = state.Players.Allies
			r.Players.Axis = state.Players.Axis
		}

		if err := h.pool.WithConnection(req.Context(), f); err != nil {
			jr.InternalServerError(w, b(err.Error()))
		}
	}(res)

	wg.Wait()
	jr.Ok(w, asJson(res))
}

func (h *handler) handleTeams(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jr.MethodNotAllowed(w)
		return
	}
	tv, err := h.getTeamView(req.Context())
	if err != nil {
		jr.InternalServerError(w, b(err.Error()))
		return
	}
	jr.Ok(w, asJson(tv))
}

func (h *handler) getTeamView(ctx context.Context) (TeamView, error) {
	res := TeamView{}
	c, err := h.pool.GetWithContext(ctx)
	if err != nil {
		return res, err
	}
	defer h.pool.Return(c)

	v, err := c.PlayerIds()
	if err != nil {
		return res, err
	}

	var infos []rcon.PlayerInfo
	var wg sync.WaitGroup
	var errs []error
	for _, l := range v {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			pi, err := h.requestPlayerInfo(ctx, name)
			if err != nil {
				errs = append(errs, err)
			} else {
				infos = append(infos, pi)
			}
		}(l.Name)
	}
	wg.Wait()

	if len(errs) != 0 {
		cErr := errors.New("could not fetch player_info")
		for _, e := range errs {
			cErr = fmt.Errorf(cErr.Error()+": %w", e)
		}
		return res, cErr
	}
	for _, info := range infos {
		if _, ok := res[info.Team]; !ok {
			res[info.Team] = &Team{
				Squads: map[string]*Squad{},
				Score:  Score{},
			}
		}
		team := res[info.Team]
		squadType := h.guessSquadType(info)

		team.Score.Merge(fromRconScore(info.Score))

		if info.Unit.Name == "" && squadType != SquadTypeCommander {
			team.NoSquadPlayers = append(team.NoSquadPlayers, fromPlayerInfo(info))
			continue
		}
		if _, ok := team.Squads[info.Unit.Name]; !ok {
			s := &Squad{
				Type:    squadType,
				Score:   Score{},
				Players: []Player{},
			}
			team.Squads[info.Unit.Name] = s
			if s.Type == SquadTypeCommander {
				team.Commander = s
			}
		}
		squad := team.Squads[info.Unit.Name]
		squad.Players = append(squad.Players, fromPlayerInfo(info))

		squad.Score.Merge(fromRconScore(info.Score))
	}
	return res, nil
}

func (h *handler) guessSquadType(info rcon.PlayerInfo) SquadType {
	r := strings.ToLower(info.Role)
	if r == "tankcommander" || r == "crewman" {
		return SquadTypeArmor
	}
	if r == "spotter" || r == "sniper" {
		return SquadTypeRecon
	}
	if r == "armycommander" {
		return SquadTypeCommander
	}
	return SquadTypeInfantry
}

func (h *handler) requestPlayerInfo(ctx context.Context, name string) (rcon.PlayerInfo, error) {
	r, err := h.pool.GetWithContext(ctx)
	if err != nil {
		return rcon.PlayerInfo{}, err
	}
	defer h.pool.Return(r)
	return r.PlayerInfo(name)
}
