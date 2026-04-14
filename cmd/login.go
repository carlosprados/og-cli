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
	loginEmail    string
	loginPassword string
)

func init() {
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "OpenGate email (or OG_EMAIL env var)")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "OpenGate password (or OG_PASSWORD env var)")
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
	}
	if err := config.SaveCredentials(profileName, creds, cfgFile); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Printf("Logged in successfully. Credentials stored in profile %q.\n", profileName)
	return nil
}
