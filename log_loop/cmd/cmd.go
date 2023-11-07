package main

import (
	"code.cloudfoundry.org/lager"
	"context"
	"github.com/floriansw/go-hll-rcon/log_loop"
	"github.com/floriansw/go-hll-rcon/rcon"
	"os"
	"strconv"
)

type Runnable interface {
	Run(ctx context.Context) error
}

var (
	loop Runnable
)

func init() {
	logger := lager.NewLogger("example")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}
	p := rcon.NewConnectionPool(logger, os.Getenv("HOST"), port, os.Getenv("PASSWORD"))

	loop = log_loop.NewLogLoop(logger, p)
}

func main() {
	if err := loop.Run(context.Background()); err != nil {
		panic(err)
	}
}
