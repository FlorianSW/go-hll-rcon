package api

import (
	"context"
	"errors"
	"github.com/floriansw/go-hll-rcon/api/response"
	. "github.com/floriansw/go-hll-rcon/api/shiftpath"
	"github.com/floriansw/go-hll-rcon/rcon"
	"net/http"
)

var (
	jr = response.Jr
	b  = response.B
)

type RConPool interface {
	GetWithContext(ctx context.Context) (*rcon.Connection, error)
	Return(c *rcon.Connection)
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
	default:
		jr.NotFound(w)
	}
}

func (h *handler) handlePlayers(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req = ShiftPath(req)
	switch head {
	case "":
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
	default:
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

		pi, err := c.PlayerInfo(head)
		if errors.Is(err, rcon.CommandFailed) {
			jr.NotFound(w)
			return
		} else if err != nil {
			jr.InternalServerError(w, b(err.Error()))
			return
		}
		jr.Ok(w, asJson(renderPlayerInfo(pi)))
	}
}
