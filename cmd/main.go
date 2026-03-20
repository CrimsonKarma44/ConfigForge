package main

import (
	"fmt"
	"os"

	"github.com/deus/configforge/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
