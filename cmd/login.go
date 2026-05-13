package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate against OpenGate and store the JWT token",
	Long:  "Logs in with email/password and stores the JWT token in the active profile.",
	RunE:  runLogin,
}

var (
	loginEmail     string
	loginPassword  string
	loginDomain    string
	loginWorkgroup string
	loginProfile   string
	loginNoWeb     bool
)

func init() {
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "OpenGate email (or OG_EMAIL env var)")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "OpenGate password (or OG_PASSWORD env var)")
	loginCmd.Flags().StringVar(&loginDomain, "domain", "", "domain for Web API signin (default: from north login response)")
	loginCmd.Flags().StringVar(&loginWorkgroup, "workgroup", "default", "workgroup for Web API signin")
	loginCmd.Flags().StringVar(&loginProfile, "user-profile", "", "user profile for Web API signin (default: from north login response)")
	loginCmd.Flags().BoolVar(&loginNoWeb, "no-web", false, "skip Web API signin (workspace/dashboard commands will be unavailable)")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	email := loginEmail
	if email == "" {
		email = os.Getenv("OG_EMAIL")
	}
	if email == "" {
		fmt.Print("Email: ")
		fmt.Scanln(&email)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	password := loginPassword
	if password == "" {
		password = os.Getenv("OG_PASSWORD")
	}
	if password == "" {
		fmt.Print("Password: ")
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		fmt.Println()
		password = string(pwBytes)
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	c := client.New(p.Host, "")
	result, err := c.Login(email, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	profileName := profile
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}

	creds := config.Credentials{
		Token:        result.JWT,
		APIKey:       result.APIKey,
		Organization: result.Domain,
		Email:        email,
	}

	// Attempt Web API signin (workspaces/dashboards) unless skipped.
	if !loginNoWeb {
		c.Token = result.JWT

		domain := loginDomain
		if domain == "" {
			domain = result.Domain
		}
		userProfile := loginProfile
		if userProfile == "" {
			userProfile = result.Profile
		}

		if domain == "" || userProfile == "" {
			fmt.Fprintln(os.Stderr, "Warning: Web API signin skipped (north login did not return domain or profile, and no override flags given). Workspace/dashboard commands will be unavailable.")
		} else {
			webResult, err := c.WebSignIn(client.WebSignInRequest{
				Email:     email,
				Domain:    domain,
				Profile:   userProfile,
				Workgroup: loginWorkgroup,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Web API signin failed (%v). Workspace/dashboard commands will be unavailable. Re-run with --no-web to silence.\n", err)
			} else {
				creds.WebToken = webResult.JWT
				creds.Domain = domain
				creds.UserProfile = userProfile
				creds.Workgroup = loginWorkgroup
			}
		}
	}

	if err := config.SaveCredentials(profileName, creds, cfgFile); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Printf("Logged in successfully. Credentials stored in profile %q.\n", profileName)
	if creds.WebToken != "" {
		fmt.Println("Web API access enabled (workspace/dashboard commands available).")
	}
	return nil
}
