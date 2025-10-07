package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/floriansw/go-hll-rcon/log_loop"
	rcon "github.com/floriansw/go-hll-rcon/rconv2"
)

type Runnable interface {
	Run(ctx context.Context, f func(l []log_loop.StructuredLogLine) bool) error
}

var (
	loop Runnable
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}
	p, err := rcon.NewConnectionPool(rcon.ConnectionPoolOptions{
		Logger:   logger,
		Hostname: os.Getenv("HOST"),
		Port:     port,
		Password: os.Getenv("PASSWORD"),
	})
	if err != nil {
		panic(err)
	}
	loop = log_loop.NewLogLoop(log_loop.LogLoopOptions{
		Logger: logger,
		Pool:   p,
	})
}

func main() {
	if err := loop.Run(context.Background(), func(l []log_loop.StructuredLogLine) bool {
		return true
	}); err != nil {
		panic(err)
	}
}
