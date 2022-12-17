package main

import (
	"context"
	"github.com/floriansw/go-hll-rcon/rcon"
	"os"
	"strconv"
	"time"
)

func main() {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}
	p := rcon.NewConnectionPool(os.Getenv("HOST"), port, os.Getenv("PASSWORD"))
	ctx, c := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer c()

	r, err := p.GetWithContext(ctx)
	if err != nil {
		panic(err)
	}
	defer p.Return(r)

	s, err := r.Command("get slots")
	if err != nil {
		panic(err)
	}
	println(s)

	l, err := r.ShowLog(time.Minute * 30)
	if err != nil {
		panic(err)
	}
	for _, ls := range l {
		println(ls)
	}

	v, err := r.ListCommand("get vipids")
	if err != nil {
		panic(err)
	}
	for _, l := range v {
		println(l)
	}
}
