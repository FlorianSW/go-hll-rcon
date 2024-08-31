# go-hll-rcon: An implementation of the HLL RCon protocol in Go

An implementation of the Hell Let Loose RCon protocol in Go.
The protocol itself is documented in a Community effort in [this document](https://gist.github.com/timraay/5634d85eab552b5dfafb9fd61273dc52).

## Usage

Import the module as usual with go modules, then use it according to the example:

```go
package main

import (
	"context"
	"github.com/floriansw/go-hll-rcon/rcon"
	"log/slog"
	"os"
	"strconv"
)

func main() {
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

	err = p.WithConnection(context.Background(), func(c *rcon.Connection) error {
		m, err := c.Maps()
		if err != nil {
			println(err.Error())
			return err
		}
		for _, n := range m {
			println(n)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
```

Executing this code will list the available maps of the Hell Let Loose server.

## Command Coverage

`go-hll-rcon` covers a subset of the available RCon commands from HLL.
The available commands are documented in [rcon/connection.go](rcon/connection.go).
