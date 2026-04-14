package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultConfigDir  = ".og"
	DefaultConfigFile = "config.yaml"
	EnvPrefix         = "OG"
)

// Profile holds connection settings for an OpenGate instance.
type Profile struct {
	Host         string `mapstructure:"host"`
	Token        string `mapstructure:"token"`
	APIKey       string `mapstructure:"api_key"`
	Organization string `mapstructure:"organization"`
}

// Config is the top-level configuration.
type Config struct {
	DefaultProfile string             `mapstructure:"default_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"`
}

// configDir returns ~/.og, creating it if needed.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, DefaultConfigDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// ConfigFilePath returns the full path to the config file.
func ConfigFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFile), nil
}

// Load reads configuration from file, env vars and .env.
// customPath overrides the default config file location if non-empty.
func Load(customPath string) (*Config, error) {
	// Load .env from cwd (best-effort)
	_ = godotenv.Load()

	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()

	if customPath != "" {
		v.SetConfigFile(customPath)
	} else {
		dir, err := configDir()
		if err != nil {
			return nil, err
		}
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(dir)
	}

	// Defaults
	v.SetDefault("default_profile", "default")
	v.SetDefault("profiles", map[string]Profile{
		"default": {Host: "https://api.opengate.es"},
	})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// ActiveProfile returns the profile selected by the --profile flag or default.
func (c *Config) ActiveProfile(name string) (*Profile, error) {
	if name == "" {
		name = c.DefaultProfile
	}
	// Env var override
	if envProfile := os.Getenv(EnvPrefix + "_PROFILE"); envProfile != "" && name == c.DefaultProfile {
		name = envProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in config", name)
	}
	// Env var overrides for host and token
	if h := os.Getenv(EnvPrefix + "_HOST"); h != "" {
		p.Host = h
	}
	if t := os.Getenv(EnvPrefix + "_TOKEN"); t != "" {
		p.Token = t
	}
	if o := os.Getenv(EnvPrefix + "_ORG"); o != "" {
		p.Organization = o
	}
	return &p, nil
}

// Credentials holds the values to persist after login.
type Credentials struct {
	Token        string
	APIKey       string
	Organization string
}

// SaveCredentials persists login credentials into the named profile.
func SaveCredentials(profileName string, creds Credentials, configPath string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		dir, err := configDir()
		if err != nil {
			return err
		}
		cfgFile := filepath.Join(dir, DefaultConfigFile)
		v.SetConfigFile(cfgFile)
	}

	// Read existing config or initialize defaults
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			v.Set("default_profile", profileName)
		}
	}

	prefix := fmt.Sprintf("profiles.%s", profileName)

	// Preserve host if already set, otherwise set default
	if v.GetString(prefix+".host") == "" {
		v.Set(prefix+".host", "https://api.opengate.es")
	}

	v.Set(prefix+".token", creds.Token)

	if creds.APIKey != "" {
		v.Set(prefix+".api_key", creds.APIKey)
	}
	if creds.Organization != "" {
		// Only set if not already configured
		if v.GetString(prefix+".organization") == "" {
			v.Set(prefix+".organization", creds.Organization)
		}
	}

	return v.WriteConfig()
}
