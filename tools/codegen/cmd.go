package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/floriansw/go-hll-rcon/codegen"
)

const (
	definitionFileSuffix     = "_rcon.json"
	definitionFilesDirectory = "./definitions/"
	packageName              = "rconv2"
	operationsPrefix         = "op_"
)

type generator interface {
	Read(def io.Reader) ([]codegen.Command, error)
}

func main() {
	entries, err := os.ReadDir(definitionFilesDirectory)
	if err != nil {
		fmt.Printf("Could not read definition files directory: %s\n", err)
		return
	}

	gen := codegen.NewGenerator(packageName, operationsPrefix)
	commands := map[string][]codegen.Command{}
	var files []string
	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), definitionFileSuffix) {
			continue
		}
		files = append(files, f.Name())
		gm := strings.Replace(f.Name(), definitionFileSuffix, "", 1)
		cmds, err := readFile(gen, f.Name())
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		commands[gm] = cmds
	}

	err = codegen.NewGenerator(packageName, operationsPrefix).Generate(commands)
	if err != nil {
		fmt.Printf("Error generating API code based on definitions file: %s\n", err)
		return
	}
	fmt.Printf("Generated API code based on definitions files: %s\n", strings.Join(files, ", "))
}

func readFile(gen generator, fn string) ([]codegen.Command, error) {
	f, err := os.Open(definitionFilesDirectory + fn)
	if err != nil {
		return nil, fmt.Errorf("Could not open definitions file: %s\n.You might need to run go run ./tools/generate_definitions/cmd.go first.\n", err)
	}
	defer f.Close()
	return gen.Read(f)
}
