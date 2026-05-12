package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/floriansw/go-hll-rcon/codegen"
	"github.com/floriansw/go-hll-rcon/rconv2"
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
	ctx := context.Background()
	var commands *rconv2.GetDisplayableCommandsResponse
	err = p.WithConnection(ctx, func(c *rconv2.Connection) error {
		commands, err = c.GetDisplayableCommands(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	println("Found " + strconv.Itoa(len(commands.Entries)) + " commands")
	var res []codegen.Command
	for _, entry := range commands.Entries {
		println("Requesting metadata for: " + entry.Id)
		err = p.WithConnection(ctx, func(c *rconv2.Connection) error {
			ref, err := c.GetClientReferenceData(ctx, rconv2.GetClientReferenceDataCommand(entry.Id))
			if err != nil {
				println("Failed to get client reference data: " + err.Error())
				return err
			}
			res = append(res, applyOverwrite(codegen.Command{
				Id:           entry.Id,
				FriendlyName: entry.FriendlyName,
				Text:         ref.Text,
				Description:  ref.Description,
				Parameters:   toCommandParameters(*ref),
			}))
			return nil
		})
		if err != nil {
			panic(err)
		}
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

	slices.SortStableFunc(res, func(a, b codegen.Command) int {
		return strings.Compare(a.Id, b.Id)
	})

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
}

func toCommandParameters(resp rconv2.GetClientReferenceDataResponse) (r []codegen.CommandParameter) {
	for _, param := range resp.Parameters {
		display := asSlice(param.DisplayMember)
		value := asSlice(param.ValueMember)
		// always exclude player ID parameter members when it is indicated to be an enum
		if param.Id == "PlayerId" || resp.Name == "SetSectorLayout" {
			display = nil
			value = nil
			param.Type = codegen.CommandParameterTypeString.String()
		}
		var anyValue []any
		isNumber := false
		isBoolean := false
		for _, s := range value {
			if s == "true" {
				isBoolean = true
				anyValue = append(anyValue, true)
				continue
			} else if s == "false" {
				isBoolean = true
				anyValue = append(anyValue, false)
				continue
			} else if isBoolean {
				panic("Found incompatible types for bool slice in " + param.Name)
			}
			n, err := strconv.Atoi(s)
			if err != nil && isNumber {
				panic("Found incompatible types for int slice in " + param.Name)
			} else if err != nil {
				anyValue = append(anyValue, s)
			} else {
				isNumber = true
				anyValue = append(anyValue, n)
			}
		}
		r = append(r, codegen.CommandParameter{
			Type:          codegen.CommandParameterType(param.Type),
			Name:          param.Name,
			Id:            param.Id,
			DisplayMember: display,
			ValueMember:   anyValue,
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
	Request  requestOverwrite  `json:"request"`
	Response *codegen.Response `json:"response"`
}

type requestOverwrite struct {
	Parameters      *[]codegen.CommandParameter `json:"parameters"`
	InlineParameter *bool                       `json:"inlineParameter"`
	CommandName     *string                     `json:"commandName"`
}

func (o overWriteCommand) Apply(cmd codegen.Command) codegen.Command {
	if o.Request.Parameters != nil {
		cmd.Parameters = *o.Request.Parameters
	}
	if o.Request.CommandName != nil {
		cmd.CommandName = o.Request.CommandName
	}
	if o.Request.InlineParameter != nil {
		cmd.InlineParameters = o.Request.InlineParameter
	}
	if o.Response != nil {
		cmd.Response = o.Response
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
