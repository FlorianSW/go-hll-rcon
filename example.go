package main

import (
	"code.cloudfoundry.org/lager"
	"context"
	"github.com/floriansw/go-hll-rcon/rcon"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	logger := lager.NewLogger("example")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}
	p := rcon.NewConnectionPool(logger, os.Getenv("HOST"), port, os.Getenv("PASSWORD"))
	ctx, c := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
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

	before := time.Now()
	printPlayerInfos(ctx, p, r)
	after := time.Now()
	log.Printf("Duration(ms): %d", after.UnixMilli()-before.UnixMilli())

	before = time.Now()
	printPlayerInfos(ctx, p, r)
	after = time.Now()
	log.Printf("Duration(ms): %d", after.UnixMilli()-before.UnixMilli())

	before = time.Now()
	printPlayerInfos(ctx, p, r)
	after = time.Now()
	log.Printf("Duration(ms): %d", after.UnixMilli()-before.UnixMilli())

	before = time.Now()
	printPlayerInfos(ctx, p, r)
	after = time.Now()
	log.Printf("Duration(ms): %d", after.UnixMilli()-before.UnixMilli())
}

func printPlayerInfos(ctx context.Context, p *rcon.ConnectionPool, r *rcon.Connection) {
	v, err := r.PlayerIds()
	if err != nil {
		panic(err)
	}
	var infos []rcon.PlayerInfo
	var wg sync.WaitGroup
	for _, l := range v {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			pi, err := requestPlayerInfo(ctx, p, name)
			if err != nil {
				panic(err)
			}
			infos = append(infos, pi)
		}(l.Name)
	}
	wg.Wait()
}

func requestPlayerInfo(ctx context.Context, p *rcon.ConnectionPool, name string) (rcon.PlayerInfo, error) {
	r, err := p.GetWithContext(ctx)
	if err != nil {
		return rcon.PlayerInfo{}, err
	}
	defer p.Return(r)
	return r.PlayerInfo(name)
}
