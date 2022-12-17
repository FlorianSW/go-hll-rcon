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

	SquadTypeCommander = SquadType("Commander")
	SquadTypeRecon     = SquadType("Recon")
	SquadTypeArmor     = SquadType("Armor")
	SquadTypeInfantry  = SquadType("Infantry")
)

type RConPool interface {
	GetWithContext(ctx context.Context) (*rcon.Connection, error)
	Return(c *rcon.Connection)
}

type TeamView map[string]*Team

type Team struct {
	Squads         map[string]*Squad
	NoSquadPlayers []rcon.PlayerInfo
	Commander      *Squad
	Score          rcon.Score
}

type SquadType string

type Squad struct {
	Type    SquadType
	Score   rcon.Score
	Players []rcon.PlayerInfo
}

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
				Score:  rcon.Score{},
			}
		}
		team := res[info.Team]
		squadType := h.guessSquadType(info)

		team.Score.Offensive += info.Score.Offensive
		team.Score.Defensive += info.Score.Defensive
		team.Score.Support += info.Score.Support
		team.Score.CombatEffectiveness += info.Score.CombatEffectiveness

		if info.Unit.Name == "" && squadType != SquadTypeCommander {
			team.NoSquadPlayers = append(team.NoSquadPlayers, info)
			continue
		}
		if _, ok := team.Squads[info.Unit.Name]; !ok {
			s := &Squad{
				Type:    squadType,
				Score:   rcon.Score{},
				Players: []rcon.PlayerInfo{},
			}
			team.Squads[info.Unit.Name] = s
			if s.Type == SquadTypeCommander {
				team.Commander = s
			}
		}
		squad := team.Squads[info.Unit.Name]
		squad.Players = append(squad.Players, info)

		squad.Score.Offensive += info.Score.Defensive
		squad.Score.Defensive += info.Score.Defensive
		squad.Score.Support += info.Score.Support
		squad.Score.CombatEffectiveness += info.Score.CombatEffectiveness
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
