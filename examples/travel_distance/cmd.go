package main

import (
	"context"
	"fmt"
	"github.com/floriansw/go-hll-rcon/rconv2"
	"github.com/floriansw/go-hll-rcon/rconv2/api"
	"github.com/rivo/tview"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// This example tracks players position over the time the tool is running.
// It then uses the collected player positions to calculate the distance a player was traveling throughout the game.
// The distance calculated includes travelling by any vehicle or by foot regardless, the same as moving horizontally or vertically,
// hence taking changes in terrain into account as well. This might result in discrepancies compared to the actual player
// speed.
//
// The example also does not take into effect map switches per se. However, a map switch should work the same way as the
// players' death, meaning that the distance simply accumulates. However, on map switch, the distance counter for a player is
// not reset automatically (while this can certainly be implemented).
//
// To run the example, set the following environment variables first:
//   - host -> the IP address of the Hell Let Loose server
//   - port -> the RCon port of the Hell Let Loose server
//   - password -> the RCon password of the Hell Let Loose server
//
// all of this information you can get from your Game Service Provider.
// Then start the example like:
//
//	go run examples/travel_distance/cmd.go
//
// Printing the results is only possible in an ANSI terminal usually found in Linux and Unix (OSX) systems. Windows on
// other hand does not provide such a terminal and might not work.
func main() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	p, err := rconv2.NewConnectionPool(rconv2.ConnectionPoolOptions{
		Logger:   l,
		Hostname: os.Getenv("host"),
		Port:     getEnvInt("port"),
		Password: os.Getenv("password"),
	})
	if err != nil {
		l.Error("create-connection-pool", "error", err)
		return
	}

	ctx := context.Background()
	r := NewRecorder(l, p)
	go r.Run(ctx)

	// printing results as often as every second should be enough for a good overview of the travelled distances
	ticker := time.NewTicker(time.Second)
	app := tview.NewApplication()
	t := tview.NewTable()
	started := time.Now()
	go func() {
		for {
			select {
			case tick := <-ticker.C:
				t.Clear()
				row := 0
				distances := r.Distances()
				slices.SortFunc(distances, func(e PlayerDistance, e2 PlayerDistance) int {
					return strings.Compare(e.Name, e2.Name)
				})
				for _, distance := range distances {
					t.SetCellSimple(row, 0, distance.Name).SetCellSimple(row, 1, fmt.Sprintf("%.2fm", distance.Distance.Meters()))
					row++
				}

				t.SetCellSimple(row+2, 0, "Time elapsed")
				t.SetCellSimple(row+2, 1, tick.Sub(started).String())
				app.Draw()
			}
		}
	}()
	if err := app.SetRoot(t, true).Run(); err != nil {
		panic(err)
	}
}

type recorder struct {
	l       *slog.Logger
	p       *rconv2.ConnectionPool
	t       *time.Ticker
	closeCh chan bool

	positions map[string]*positionData
}

type positionData struct {
	Name      string
	Positions []api.WorldPosition
}

func NewRecorder(l *slog.Logger, p *rconv2.ConnectionPool) *recorder {
	return &recorder{
		l:         l,
		p:         p,
		t:         time.NewTicker(500 * time.Millisecond),
		positions: map[string]*positionData{},
		closeCh:   make(chan bool),
	}
}

type PlayerDistance struct {
	Name     string
	Distance api.Distance
}

// Distances calculates the distances travelled by each player based on the data that is already recorded.
// The distances, by default, are represented in centimeters (cm).
func (r *recorder) Distances() []PlayerDistance {
	var result []PlayerDistance
	for _, data := range r.positions {
		var d api.Distance
		for i, position := range data.Positions {
			if i == 0 {
				continue
			}
			previous := data.Positions[i-1]

			if !previous.IsSpawned() || !position.IsSpawned() {
				continue
			}
			d = d.Add(position.Distance(previous))
		}
		result = append(result, PlayerDistance{
			Name:     data.Name,
			Distance: d,
		})
	}
	return result
}

func (r *recorder) Run(ctx context.Context) {
	for {
		select {
		case <-r.closeCh:
			return
		case _ = <-r.t.C:
			err := r.p.WithConnection(ctx, func(c *rconv2.Connection) error {
				players, err := c.Players(ctx)
				if err != nil {
					r.l.Error("read-players", "error", err)
					return err
				}

				for _, player := range players.Players {
					if _, ok := r.positions[player.Id]; !ok {
						r.positions[player.Id] = &positionData{Name: player.Name}
					}

					v := r.positions[player.Id]
					if len(v.Positions) != 0 {
						last := v.Positions[len(v.Positions)-1]
						if last.Equal(player.Position) {
							continue
						}
					}
					v.Positions = append(r.positions[player.Id].Positions, player.Position)
				}
				return nil
			})
			if err != nil {
				r.l.Error("process-tick", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func getEnvInt(name string) int {
	if v, ok := os.LookupEnv(name); !ok {
		panic("Missing environment variable " + name)
	} else if iv, err := strconv.Atoi(v); err != nil {
		panic(fmt.Sprintf("error converting string env var %s value %s to int: %s", name, v, err.Error()))
	} else {
		return iv
	}
}
