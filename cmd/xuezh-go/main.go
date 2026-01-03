package main

import (
	"os"

	"github.com/joshp123/xuezh/internal/xuezh/cli"
)

func main() {
	exitCode := cli.Run(os.Args[1:])
	os.Exit(exitCode)
}
