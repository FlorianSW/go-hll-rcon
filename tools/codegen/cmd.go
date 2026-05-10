package main

import (
	"fmt"
	"os"

	"github.com/floriansw/go-hll-rcon/codegen"
)

const (
	definitionsFile  = "./definitions/hll_rcon.json"
	packageName      = "rconv2"
	operationsPrefix = "op_"
)

func main() {
	f, err := os.Open(definitionsFile)
	if err != nil {
		fmt.Printf("Could not open definitions file: %s\n.You might need to run go run ./tools/generate_definitions/cmd.go first.\n", err)
		return
	}
	defer f.Close()
	err = codegen.NewGenerator(packageName, operationsPrefix).Generate(f)
	if err != nil {
		fmt.Printf("Error generating API code based on definitions file: %s\n", err)
		return
	}
	fmt.Printf("Generated API code based on definitions file: %s\n", definitionsFile)
}
