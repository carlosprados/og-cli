package cmd

import (
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
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
	enableRecursiveHelp(rootCmd)
	return rootCmd.Execute()
}

// enableRecursiveHelp walks the command tree and installs a 'help [subcmd]'
// subcommand on every parent so the user can drill down at any level —
// e.g. `og workspace help unwrap` works the same as `og help workspace unwrap`.
//
// Cobra adds this automatically only on the root command; this helper extends
// the behaviour to intermediate parents.
func enableRecursiveHelp(c *cobra.Command) {
	for _, child := range c.Commands() {
		if !child.HasSubCommands() {
			continue
		}
		addHelpSubcommand(child)
		enableRecursiveHelp(child)
	}
}

func addHelpSubcommand(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:                   "help [subcommand]",
		Short:                 "Help about " + parent.Name() + " or one of its subcommands",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				_ = parent.Help()
				return
			}
			target, _, err := parent.Find(args)
			if err != nil || target == nil || target == parent {
				_ = parent.Help()
				return
			}
			_ = target.Help()
		},
	})
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

// newWebClient builds a Client configured for Web API access with transparent
// auto-refresh: if a 401 is received, the client re-signs in and retries once.
// The refreshed token is persisted back to the active profile.
func newWebClient(p *config.Profile) *client.Client {
	c := client.New(p.Host, p.Token).WithWebToken(p.WebToken)

	if p.Email == "" || p.Domain == "" || p.UserProfile == "" || p.Workgroup == "" {
		return c
	}

	profileName := profile
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}

	req := client.WebSignInRequest{
		Email:     p.Email,
		Domain:    p.Domain,
		Profile:   p.UserProfile,
		Workgroup: p.Workgroup,
	}

	onRefresh := func(newToken string) {
		err := config.SaveCredentials(profileName, config.Credentials{
			Token:        p.Token,
			WebToken:     newToken,
			APIKey:       p.APIKey,
			Organization: p.Organization,
			Email:        p.Email,
			Domain:       p.Domain,
			UserProfile:  p.UserProfile,
			Workgroup:    p.Workgroup,
		}, cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: refreshed web token but failed to persist it: %v\n", err)
		}
	}

	return c.WithWebRefresh(req, onRefresh)
}
