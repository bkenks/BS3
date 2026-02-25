package main

import (
	"os"

	"github.com/bkenks/bs3-cli/internal/cli"
)

func main() {
	cli.Run(os.Args[1:])
}
