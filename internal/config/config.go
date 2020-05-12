package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	// DiscordListenIDs is a list of channel ID where the bot will listen and
	// accept commands. PMs are always listened to.
	DiscordListenIDs []string

	// Who is allowed to use `!dev` commands.
	DiscordAdminUserIDs []string
}

func NewFromUserConfigDir() (*Config, error) {
	c := &Config{}
	if err := c.ReloadFromUserConfigDir(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) ReloadFromUserConfigDir() error {
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
