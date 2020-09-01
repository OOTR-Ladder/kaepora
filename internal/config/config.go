package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// Config holds the state shared by all components of the ladder. The
// configuration is read from file and overridden with env vars.
type Config struct {
	Discord Discord

	CookieHashKey, CookieBlockKey string
	OOTRAPIKey                    string

	Domain string

	// DevMode weakens security during development and disables HTTPS.
	DevMode bool
}

// Discord holds discord-specific configuration for both the bot and the web backend.
type Discord struct {
	Token                  string
	ClientID, ClientSecret string

	// DiscordListenIDs is a list of channel ID where the bot will listen and
	// accept commands. PMs are always listened to.
	ListenIDs []string

	// Who is allowed to use `!dev` commands.
	AdminUserIDs []string

	// Who is not allowed to do anything.
	BannedUserIDs []string
}

func (c Discord) CanRunBot() bool {
	return c.Token != ""
}

func (c Discord) CanSetupOAuth2() bool {
	return c.ClientSecret != "" && c.ClientID != ""
}

func (c Discord) OAuth2(conf *Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL: fmt.Sprintf(
			"%s://%s/auth/oauth2/discord",
			conf.Scheme(),
			conf.Domain,
		),
		Scopes: []string{"identify"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
}

func (c *Config) Scheme() string {
	if c.DevMode {
		return "http"
	}
	return "https"
}

// IsDiscordIDBanned returns true if the given Discord user ID is a banned
// from the ladder, meaning he can't even talk to the bot.
func (c *Config) IsDiscordIDBanned(id string) bool {
	for _, v := range c.Discord.BannedUserIDs {
		if v == id {
			return true
		}
	}

	return false
}

// IsDiscordIDAdmin returns true if the given Discord user ID is a Kaepora
// admin, meaning he has access to extra data and dangerous commands.
func (c *Config) IsDiscordIDAdmin(id string) bool {
	for _, v := range c.Discord.AdminUserIDs {
		if v == id {
			return true
		}
	}

	return false
}

// NewFromUserConfigDir reads a config file from a standard directory.
func NewFromUserConfigDir() (*Config, error) {
	c := &Config{}
	if err := c.reloadFromUserConfigDir(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) expandFromEnv() {
	vars := []struct {
		src string
		dst *string
	}{
		{"KAEPORA_DISCORD_TOKEN", &c.Discord.Token},
		{"KAEPORA_DISCORD_CLIENT_SECRET", &c.Discord.ClientSecret},
		{"KAEPORA_DISCORD_CLIENT_ID", &c.Discord.ClientID},

		{"KAEPORA_OOTR_API_KEY", &c.OOTRAPIKey},
		{"KAEPORA_COOKIE_HASH_KEY", &c.CookieHashKey},
		{"KAEPORA_COOKIE_BLOCK_KEY", &c.CookieBlockKey},
	}

	for _, v := range vars {
		if str := os.Getenv(v.src); str != "" {
			*v.dst = str
		}
	}
}

func (c *Config) reloadFromUserConfigDir() error {
	defer c.expandFromEnv()

	path, err := getOrCreateUserConfigPath()
	if err != nil {
		return err
	}
	log.Printf("debug: reading conf from %s", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		*c = Config{}
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(c)
}

func getOrCreateUserConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(configDir, "kaepora")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

func (c *Config) Write() error {
	path, err := getOrCreateUserConfigPath()
	if err != nil {
		return err
	}
	log.Printf("debug: writing conf to %s", path)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(c); err != nil {
		if err2 := f.Close(); err2 != nil {
			return fmt.Errorf("unable to close file (%s) after error: %w", err2, err)
		}

		return err
	}

	return f.Close()
}
