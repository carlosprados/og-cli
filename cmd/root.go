package cmd

import (
	"fmt"

	"github.com/carlosprados/og-cli/internal/config"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	profile    string
	outputFlag string
	org        string

	cfg    *config.Config
	outFmt output.Format
)

var rootCmd = &cobra.Command{
	Use:   "og",
	Short: "OpenGate CLI — interact with the OpenGate IoT platform",
	Long:  "og is a command-line interface for the OpenGate REST API by Amplía Soluciones.\n\nRun without arguments to launch the interactive TUI.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		outFmt, err = output.ParseFormat(outputFlag)
		if err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		return tui.Run(cfg, p, cfgFile)
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.og/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "config profile to use")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "output format: json|table")
	rootCmd.PersistentFlags().StringVar(&org, "org", "", "organization name (or OG_ORG env var)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// activeProfile returns the resolved profile from config.
func activeProfile() (*config.Profile, error) {
	return cfg.ActiveProfile(profile)
}

// resolveOrg returns the organization from --org flag, profile config, or error.
func resolveOrg(p *config.Profile) (string, error) {
	if org != "" {
		return org, nil
	}
	if p.Organization != "" {
		return p.Organization, nil
	}
	return "", fmt.Errorf("organization is required (use --org flag, OG_ORG env var, or set it in your profile)")
}
