package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// `og whoami` answers "am I logged in, as whom, and for how much longer".
//
// Until now the only way to find out was to run a real command and read the
// failure, which is a poor way to ask a question: it needs the network, it needs
// an organization, and a 401 does not distinguish "never logged in" from "the
// token expired an hour ago". Both of those happened repeatedly while building
// the editor plugins, which is what this is for — a plugin needs to know whether
// to offer a login before the user hits a wall.
//
// Local by default. Reading the token's own claims is instant, works offline and
// has no side effects; --check adds a request when it matters whether the
// platform still accepts it.

var whoamiCheck bool

// session is the answer, and the shape emitted under -o json.
type session struct {
	Profile      string `json:"profile"`
	Host         string `json:"host"`
	Organization string `json:"organization,omitempty"`

	LoggedIn bool   `json:"loggedIn"`
	User     string `json:"user,omitempty"`
	Name     string `json:"name,omitempty"`

	IssuedAt  string `json:"issuedAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Expired   bool   `json:"expired"`
	// ExpiresIn is human-readable, e.g. "21h34m". Empty once expired.
	ExpiresIn string `json:"expiresIn,omitempty"`

	// WebSession reports whether workspace and dashboard commands will work.
	WebSession bool `json:"webSession"`
	// APIKey says only whether one is stored. Never its value.
	APIKey bool `json:"apiKey"`

	// Accepted is set by --check: whether the platform honoured the token.
	Accepted *bool  `json:"accepted,omitempty"`
	Problem  string `json:"problem,omitempty"`
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current session: who, where, and until when",
	Long: `Show what session this profile holds.

Reads the token's own claims, so it is instant, works offline and makes no API
call. That is enough to answer the question that matters most often — whether
there is a session at all, whose it is, and whether it has expired.

With --check it also makes one request, which is what distinguishes a token that
merely looks valid from one the platform still honours.

Exit codes: 0 when there is a usable session, 1 when there is none or it has
expired (or, with --check, was rejected), 2 on failure. That makes it usable as
a guard in a script or an editor plugin.

Examples:
  og whoami
  og whoami --check
  og whoami -o json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := cfg.ActiveProfile(profile)
		if err != nil {
			return err
		}

		s := session{Profile: profileName(), Host: p.Host, Organization: p.Organization}
		s.WebSession = p.WebToken != ""
		s.APIKey = p.APIKey != ""

		if p.Token == "" {
			s.Problem = "no token in this profile — run `og login`"
		} else if claims, cErr := readClaims(p.Token); cErr != nil {
			// A token that cannot be read is not a token we can vouch for, but
			// it may still be one the platform accepts, so --check can still
			// have the last word.
			s.Problem = fmt.Sprintf("the stored token could not be read: %v", cErr)
		} else {
			s.LoggedIn = true
			s.User, s.Name = claims.Sub, claims.Name
			if claims.IssuedAt > 0 {
				s.IssuedAt = time.Unix(claims.IssuedAt, 0).Format(time.RFC3339)
			}
			if claims.Expires > 0 {
				expiry := time.Unix(claims.Expires, 0)
				s.ExpiresAt = expiry.Format(time.RFC3339)
				if remaining := time.Until(expiry); remaining > 0 {
					s.ExpiresIn = remaining.Round(time.Minute).String()
				} else {
					s.Expired = true
					s.LoggedIn = false
					s.Problem = "the token expired — run `og login`"
				}
			}
		}

		if whoamiCheck && p.Token != "" {
			c := opengate.New(p.Host, p.Token, p.ClientOptions()...)
			// The connector function catalogue is a platform-level read: it
			// needs no organization, so --check works on a profile that has
			// none configured.
			_, apiErr := c.ConnectorFunctionsCatalog(cmd.Context())
			accepted := apiErr == nil
			s.Accepted = &accepted
			if apiErr != nil {
				s.LoggedIn = false
				s.Problem = apiErr.Error()
			}
		}

		if outFmt == output.FormatJSON {
			if err := output.PrintEnvelope(os.Stdout, "whoami", s); err != nil {
				return err
			}
		} else {
			printSession(s)
		}

		if !s.LoggedIn {
			return &ExitError{Code: ExitDiff}
		}
		return nil
	},
}

func printSession(s session) {
	if s.User != "" {
		name := s.User
		if s.Name != "" {
			name = fmt.Sprintf("%s (%s)", s.User, s.Name)
		}
		fmt.Printf("%s\n", name)
	} else {
		fmt.Println("not logged in")
	}

	fmt.Printf("  profile       %s\n", s.Profile)
	fmt.Printf("  host          %s\n", s.Host)
	if s.Organization != "" {
		fmt.Printf("  organization  %s\n", s.Organization)
	}
	switch {
	case s.Expired:
		fmt.Printf("  token         expired %s\n", s.ExpiresAt)
	case s.ExpiresIn != "":
		fmt.Printf("  token         valid for %s (until %s)\n", s.ExpiresIn, s.ExpiresAt)
	}
	if s.WebSession {
		fmt.Printf("  web session   yes — workspace and dashboard commands are available\n")
	} else {
		fmt.Printf("  web session   no — workspace and dashboard commands will fail\n")
	}
	if s.Accepted != nil {
		if *s.Accepted {
			fmt.Printf("  platform      accepted the token\n")
		} else {
			fmt.Printf("  platform      rejected the token\n")
		}
	}
	if s.Problem != "" {
		fmt.Printf("\n%s\n", s.Problem)
	}
}

// jwtClaims is the subset of the token's payload worth reporting.
//
// Deliberately not every field: the payload also carries X-ApiKey, and a
// command whose job is to describe a session should not be a way to print a
// secret to a terminal or a log.
type jwtClaims struct {
	Sub      string `json:"sub"`
	Name     string `json:"name"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

// readClaims decodes a JWT payload.
//
// This reads the claims; it does not validate the token. Only the platform can
// do that, which is what --check is for. Anything reported from here describes
// what the stored token says about itself.
func readClaims(token string) (jwtClaims, error) {
	var claims jwtClaims

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, fmt.Errorf("not a JWT (%d segments)", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("decoding the payload: %w", err)
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, fmt.Errorf("parsing the payload: %w", err)
	}
	return claims, nil
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiCheck, "check", false,
		"also make one request, to confirm the platform still accepts the token")
	rootCmd.AddCommand(whoamiCmd)
}
