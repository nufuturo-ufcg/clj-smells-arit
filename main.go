package main

import (
	"os"

	"github.com/thlaurentino/arit/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {

		os.Exit(1)
	}
}
