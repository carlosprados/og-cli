package main

import (
	"os"

	"github.com/carlosprados/og-cli/v2/cmd"
)

func main() {
	cmd.SetSkillsFS(skillsFS)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
