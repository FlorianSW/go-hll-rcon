package log_loop

import (
	"context"
	"io"
	"log/slog"
	"time"

	rcon "github.com/floriansw/go-hll-rcon/rconv2"
)

type RConPool interface {
	WithConnection(ctx context.Context, f func(c *rcon.Connection) error) error
}

type LogLoop struct {
	logger             *slog.Logger
	p                  RConPool
	initialLogDuration time.Duration

	lastSeen       *StructuredLogLine
	reconnectTries int
}

type LogLoopOptions struct {
	Logger            *slog.Logger
	Pool              RConPool
	InitialLogMinutes *int
}

// NewLogLoop instantiates a log loop, which periodically requests logs from the game server, parses them and exposes them
// in batches to the caller.
//
// Not all events are currently parsed, however, each log line is added to the batches at least with the raw message.
func NewLogLoop(opts LogLoopOptions) *LogLoop {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	}
	initialLogDuration := 60 * time.Minute
	if opts.InitialLogMinutes != nil {
		initialLogDuration = time.Duration(*opts.InitialLogMinutes) * time.Minute
	}
	return &LogLoop{
		logger:             opts.Logger,
		p:                  opts.Pool,
		initialLogDuration: initialLogDuration,
	}
}

// Run starts polling logs from the server. Each time at least one new log line is discovered, the passed function f
// is called with an undefined number of log lines as it's only argument. Whenever f is called, at least one log line
// will be in the log line slice.
//
// The return value of f is a boolean indicating if polling for new log lines should continue. Returning true will stop
// this run with no error. Polling can be restarted by calling Run again.
// Returning false will result in Run to continue polling for new log lines.
func (l *LogLoop) Run(ctx context.Context, f func(l []StructuredLogLine) bool) error {
	log := l.logger.With("action", "log-loop-run")
	lines := make(chan []string)
	errs := make(chan error)
	l.lastSeen = nil
	d := l.initialLogDuration
	log.Info("initializing")
	go func() {
		log.Info("start")
		for {
			err := l.p.WithConnection(ctx, func(c *rcon.Connection) error {
				r, err := c.ShowLog(d)
				if err != nil {
					log.Error("read", err)
					errs <- err
				} else {
					log.Debug("read", "no", len(r))
					lines <- r
				}
				d = time.Minute
				return err
			})
			if err != nil {
				log.Error("init", err)
				errs <- err
			}
			time.Sleep(5 * time.Second)
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
				if l.lastSeen == nil || (l.lastSeen != nil && l.lastSeen.Timestamp.Before(logLine.Timestamp)) {
					pl = append(pl, logLine)
				}
			}
			if len(pl) != 0 {
				l.lastSeen = &pl[len(pl)-1]
			}
			if stop := f(pl); stop {
				return nil
			}
		case err := <-errs:
			if !rcon.IsBrokenHllConnection(err) {
				return err
			}
		}
	}
}
