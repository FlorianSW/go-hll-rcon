package log_loop

import (
	"code.cloudfoundry.org/lager"
	"context"
	"errors"
	"fmt"
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

func NewLogLoop(l lager.Logger, p RConPool) *logLoop {
	return &logLoop{
		logger: l,
		p:      p,
	}
}

func (l *logLoop) Run(ctx context.Context) error {
	log := l.logger.Session("log-loop-run")
	lines := make(chan []string)
	errs := make(chan error)
	log.Info("initializing")
	go func() {
		err := l.p.WithConnection(ctx, func(c *rcon.Connection) {
			log.Info("start")
			for {
				r, err := c.ShowLog(5 * time.Minute)
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
			for _, s := range line {
				if s == "" {
					continue
				}
				logLine, err := ParseLogLine(s)
				if err != nil {
					fmt.Println(err)
				}
				println(logLine.String())
			}
		case err := <-errs:
			return err
		}
	}
}
