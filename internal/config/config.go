package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds the state shared by all components of the ladder. The
// configuration is read from file and overridden with env vars.
type Config struct {
	// DiscordListenIDs is a list of channel ID where the bot will listen and
	// accept commands. PMs are always listened to.
	DiscordListenIDs []string

	// Who is allowed to use `!dev` commands.
	DiscordAdminUserIDs []string

	// Who is not allowed to do anything.
	DiscordBannedUserIDs []string

	DiscordToken, WebToken, OOTRAPIKey string

	Domain string
}

// IsDiscordIDBanned returns true if the given Discord user ID is a banned
// from the ladder, meaning he can't even talk to the bot.
func (c *Config) IsDiscordIDBanned(id string) bool {
	for _, v := range c.DiscordBannedUserIDs {
		if v == id {
			return true
		}
	}

	return false
}

// IsDiscordIDAdmin returns true if the given Discord user ID is a Kaepora
// admin, meaning he has access to extra data and dangerous commands.
func (c *Config) IsDiscordIDAdmin(id string) bool {
	for _, v := range c.DiscordAdminUserIDs {
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
		{"KAEPORA_OOTR_API_KEY", &c.OOTRAPIKey},
		{"KAEPORA_DISCORD_TOKEN", &c.DiscordToken},
		{"KAEPORA_WEB_TOKEN", &c.WebToken},
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
