package main

import (
	"os"

	"github.com/carlosprados/og-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
