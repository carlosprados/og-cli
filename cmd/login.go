package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosprados/og-cli/internal/config"
	"github.com/carlosprados/og-cli/pkg/opengate"
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
	loginEmail       string
	loginPassword    string
	loginTwoFaCode   string
	loginTwoFaSecret string
	loginDomain      string
	loginWorkgroup   string
	loginProfile     string
	loginNoWeb       bool
)

func init() {
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "OpenGate email (or OG_EMAIL env var)")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "OpenGate password (or OG_PASSWORD env var)")
	loginCmd.Flags().StringVar(&loginTwoFaCode, "2fa-code", "", "6-digit TOTP code for accounts with 2FA enabled (or OG_2FA_CODE env var; prompted if omitted)")
	loginCmd.Flags().StringVar(&loginTwoFaSecret, "2fa-secret", "", "base32 TOTP secret; stored in the profile so og generates the code itself on every login (or OG_2FA_SECRET env var, which is not persisted)")
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

	twoFaCode := loginTwoFaCode
	if twoFaCode == "" {
		twoFaCode = os.Getenv("OG_2FA_CODE")
	}

	// Resolve the TOTP secret (flag > env > stored profile). With a secret and
	// no explicit code, og generates the 6-digit code itself — fully
	// non-interactive 2FA. An explicit --2fa-code always wins.
	twoFaSecret := loginTwoFaSecret
	if twoFaSecret == "" {
		twoFaSecret = os.Getenv("OG_2FA_SECRET")
	}
	if twoFaSecret == "" {
		twoFaSecret = p.TOTPSecret
	}
	if twoFaCode == "" && twoFaSecret != "" {
		code, gerr := opengate.GenerateTOTPCode(twoFaSecret)
		if gerr != nil {
			return gerr
		}
		twoFaCode = code
	}

	c := opengate.New(p.Host, "")
	result, err := c.Login(cmd.Context(), email, password, twoFaCode)
	if err != nil && opengate.Is2FAChallenge(err) {
		// The account has 2FA enabled and we need a (fresh) TOTP code.
		code, perr := prompt2FACode(twoFaCode != "")
		if perr != nil {
			return perr
		}
		result, err = c.Login(cmd.Context(), email, password, code)
	}
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
		// Persist the TLS escape hatches so later commands inherit them
		// (e.g. logging in to a self-signed server with --insecure).
		Insecure: effInsecure,
		CAFile:   effCAFile,
		// Persist the TOTP secret only when given explicitly via flag; the
		// OG_2FA_SECRET env path is intentionally never written to disk.
		TOTPSecret: loginTwoFaSecret,
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
			webResult, err := c.WebSignIn(cmd.Context(), opengate.WebSignInRequest{
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

// prompt2FACode asks the user for a TOTP code interactively. retried is true
// when a code was already supplied (via flag/env) and rejected by the server.
// On a non-interactive stdin it returns an actionable error instead of hanging.
func prompt2FACode(retried bool) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		if retried {
			return "", fmt.Errorf("2FA code invalid or expired; supply a fresh code via --2fa-code or OG_2FA_CODE")
		}
		return "", fmt.Errorf("this account requires a 2FA code; supply it via --2fa-code or OG_2FA_CODE")
	}
	if retried {
		fmt.Fprintln(os.Stderr, "Invalid or expired 2FA code, try again.")
	}
	fmt.Print("2FA code: ")
	var code string
	fmt.Scanln(&code)
	return strings.TrimSpace(code), nil
}
