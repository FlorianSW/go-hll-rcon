package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/floriansw/go-hll-rcon/codegen"
	"github.com/floriansw/go-hll-rcon/rconv2"
	"github.com/floriansw/go-hll-rcon/rconv2/api"
)

func main() {
	port, err := strconv.Atoi(os.Getenv("HLL_PORT"))
	if err != nil {
		panic(err)
	}
	p, err := rconv2.NewConnectionPool(rconv2.ConnectionPoolOptions{
		Hostname: os.Getenv("HLL_HOST"),
		Port:     port,
		Password: os.Getenv("HLL_PASSWORD"),
	})
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
	var res []codegen.Command
	err = p.WithConnection(ctx, func(c *rconv2.Connection) error {
		commands, err := c.DisplayableCommands(ctx)
		if err != nil {
			return err
		}
		for _, entry := range commands.Entries {
			ref, err := c.GetClientReferenceData(ctx, entry.Id)
			if err != nil {
				return err
			}
			res = append(res, applyOverwrite(codegen.Command{
				Id:           entry.Id,
				FriendlyName: entry.FriendlyName,
				Text:         ref.Text,
				Description:  ref.Description,
				Parameters:   toCommandParameters(ref.Parameters),
			}))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	op := path.Clean("./definitions/synthetics/")
	dl, err := os.ReadDir(op)
	if err != nil {
		panic(err)
	}
	for _, entry := range dl {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "op_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		f, err := os.Open(path.Join(op, entry.Name()))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		var cmd codegen.Command
		err = json.NewDecoder(f).Decode(&cmd)
		if err != nil {
			panic(err)
		}
		res = append(res, cmd)
	}

	f, err := os.OpenFile("./definitions/hll_rcon.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	err = enc.Encode(res)
	if err != nil {
		panic(err)
	}
	cancel()
}

func toCommandParameters(parameters []api.Parameter) (r []codegen.CommandParameter) {
	for _, param := range parameters {
		display := asSlice(param.DisplayMember)
		value := asSlice(param.ValueMember)
		// always exclude player ID parameter members when it is indicated to be an enum
		if param.Id == "PlayerId" && param.Type == codegen.CommandParameterTypeEnum.String() {
			display = []string{}
			value = []string{}
		}
		r = append(r, codegen.CommandParameter{
			Type:          codegen.CommandParameterType(param.Type),
			Name:          param.Name,
			Id:            param.Id,
			DisplayMember: display,
			ValueMember:   value,
		})
	}
	return
}

func asSlice(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

type overWriteCommand struct {
	Request requestOverwrite `json:"request"`
}

type requestOverwrite struct {
	Parameters *[]codegen.CommandParameter
}

func (o overWriteCommand) Apply(cmd codegen.Command) codegen.Command {
	if o.Request.Parameters != nil {
		cmd.Parameters = *o.Request.Parameters
	}
	return cmd
}

func applyOverwrite(cmd codegen.Command) codegen.Command {
	f, err := os.Open(fmt.Sprintf("./definitions/overwrites/op_%s.json", cmd.Id))
	if errors.Is(err, os.ErrNotExist) {
		return cmd
	}
	defer f.Close()
	var overWrite overWriteCommand
	err = json.NewDecoder(f).Decode(&overWrite)
	if err != nil {
		fmt.Printf("Could not decode overwrite file for command '%s': %s\n", cmd.Id, err)
		return cmd
	}
	return overWrite.Apply(cmd)
}
