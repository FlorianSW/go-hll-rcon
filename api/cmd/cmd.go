package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/floriansw/go-hll-rcon/api"
	"github.com/floriansw/go-hll-rcon/rcon"
	"net/http"
	"os"
	"strconv"
)

var (
	handler http.Handler
)

func init() {
	logger := lager.NewLogger("example")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}
	p := rcon.NewConnectionPool(logger, os.Getenv("HOST"), port, os.Getenv("PASSWORD"))

	handler = api.NewHandler(p)
}

func main() {
	if err := http.ListenAndServe(":8080", handler); err != http.ErrServerClosed {
		panic(err)
	}
}
