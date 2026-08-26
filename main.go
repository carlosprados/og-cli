package main

import (
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/v2/cmd"
)

func main() {
	cmd.SetSkillsFS(skillsFS)

	err := cmd.Execute()
	if cmd.ShouldPrint(err) {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	os.Exit(cmd.ExitCode(err))
}
