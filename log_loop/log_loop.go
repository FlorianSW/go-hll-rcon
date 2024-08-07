package log_loop

import (
	"code.cloudfoundry.org/lager"
	"context"
	"errors"
	"github.com/floriansw/go-hll-rcon/rcon"
	"time"
)

type RConPool interface {
	WithConnection(ctx context.Context, f func(c *rcon.Connection)) error
}

type logLoop struct {
	logger lager.Logger
	p      RConPool
}

// NewLogLoop instantiates a log loop, which periodically requests logs from the game server, parses them and exposes them
// in batches to the caller.
//
// Not all events are currently parsed, however, each log line is added to the batches at least with the raw message.
func NewLogLoop(l lager.Logger, p RConPool) *logLoop {
	return &logLoop{
		logger: l,
		p:      p,
	}
}

func (l *logLoop) Run(ctx context.Context, f func(l []StructuredLogLine) bool) error {
	log := l.logger.Session("log-loop-run")
	lines := make(chan []string)
	errs := make(chan error)
	log.Info("initializing")
	go func() {
		err := l.p.WithConnection(ctx, func(c *rcon.Connection) {
			log.Info("start")
			for {
				r, err := c.ShowLog(60 * time.Minute)
				if err != nil {
					log.Error("read", err)
					errs <- err
					break
				} else {
					log.Debug("read", lager.Data{"no": len(r)})
					lines <- r
				}
				time.Sleep(5 * time.Second)
			}
		})
		if err != nil {
			log.Error("init", err)
			errs <- errors.New("test")
		}
	}()

	for {
		select {
		case line := <-lines:
			pl := make([]StructuredLogLine, len(lines))
			for _, s := range line {
				if s == "" {
					continue
				}
				logLine, err := ParseLogLine(s)
				if err != nil {
					return err
				}
				pl = append(pl, logLine)
			}
			if stop := f(pl); stop {
				return nil
			}
		case err := <-errs:
			return err
		}
	}
}
