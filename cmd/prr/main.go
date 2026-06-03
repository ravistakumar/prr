package main

import (
	"fmt"
	"os"

	"github.com/ravistakumar/prr/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "prr:", err)
		os.Exit(1)
	}
}
