package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/carlosprados/og-cli/internal/config"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/tui"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	profile    string
	outputFlag string
	org        string

	flagInsecure   bool
	flagCAFile     string
	flagAPIVersion string
	flagRetries    int

	cfg    *config.Config
	outFmt output.Format

	// Effective TLS settings resolved at startup (flag > profile). Read by
	// commands that build their own transport (e.g. MQTT in cmd/iot.go).
	effInsecure bool
	effCAFile   string
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
		if err := resolveTLS(cmd); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		return tui.Run(cmd.Context(), cfg, p, profile, cfgFile)
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.og/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "config profile to use")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "output format: json|table")
	rootCmd.PersistentFlags().StringVar(&org, "org", "", "organization name (or OG_ORG env var)")
	rootCmd.PersistentFlags().BoolVar(&flagInsecure, "insecure", false, "skip TLS certificate verification (escape hatch for self-signed HTTP/MQTT servers)")
	rootCmd.PersistentFlags().StringVar(&flagCAFile, "ca-file", "", "PEM file with extra CA/chain certs to trust (HTTP and MQTT)")
	rootCmd.PersistentFlags().StringVar(&flagAPIVersion, "api-version", "", "API version segment to target (default v80; for on-premises instances)")
	rootCmd.PersistentFlags().IntVar(&flagRetries, "retry", 0, "attempts per request; retries HTTP 429 and 5xx with backoff (0/1 = no retry)")
	rootCmd.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false, "skip confirmation prompts for destructive operations (delete/cancel)")
}

// resolveTLS computes the effective TLS settings (CLI flag overrides the active
// profile / env) and applies them process-wide via opengate.ConfigureTLS, so
// every North/South HTTP client created afterwards honours them. MQTT reads the
// resolved effInsecure/effCAFile directly. Profile resolution is best-effort:
// commands like `login` may run before a profile exists.
func resolveTLS(cmd *cobra.Command) error {
	effInsecure, effCAFile = flagInsecure, flagCAFile

	if p, err := activeProfile(); err == nil {
		if !cmd.Flags().Changed("insecure") {
			effInsecure = p.Insecure
		}
		if !cmd.Flags().Changed("ca-file") {
			effCAFile = p.CAFile
		}
	}

	// The CLI talks to exactly one endpoint per invocation, which is the one case
	// the process-wide setting models correctly. Library consumers use WithTLS.
	if err := opengate.ConfigureTLS(effInsecure, effCAFile); err != nil { //nolint:staticcheck // SA1019: intentional, see above
		return err
	}
	if effInsecure {
		fmt.Fprintln(os.Stderr, "Warning: TLS certificate verification disabled (--insecure).")
	}
	return nil
}

// Execute runs the root command with a context cancelled by SIGINT/SIGTERM, so
// every in-flight request aborts on Ctrl-C instead of running to completion.
// Commands reach it via cmd.Context().
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	enableRecursiveHelp(rootCmd)
	return rootCmd.ExecuteContext(ctx)
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

// activeProfile returns the resolved profile from config, with the global CLI
// flags applied on top.
//
// Resolution order is flag > env > profile, matching --insecure/--ca-file. The
// overrides land on the profile rather than in package variables so that every
// surface reached from here — including the TUI and the MCP server, which are
// handed this profile — configures its client the same way.
func activeProfile() (*config.Profile, error) {
	p, err := cfg.ActiveProfile(profile)
	if err != nil {
		return nil, err
	}
	if flagAPIVersion != "" {
		p.APIVersion = flagAPIVersion
	}
	if flagRetries > 0 {
		p.Retries = flagRetries
	}
	return p, nil
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
func newWebClient(p *config.Profile) *opengate.Client {
	c := opengate.New(p.Host, p.Token, p.ClientOptions()...).WithWebToken(p.WebToken)

	if p.Email == "" || p.Domain == "" || p.UserProfile == "" || p.Workgroup == "" {
		return c
	}

	profileName := profile
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}

	req := opengate.WebSignInRequest{
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
