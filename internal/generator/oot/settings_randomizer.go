package oot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kaepora/internal/generator"
	"kaepora/internal/generator/oot/settings"
	"os"
	"path/filepath"

	jsonpatch "github.com/evanphx/json-patch"
)

const SettingsRandomizerName = "oot-settings-randomizer"

const SettingsCostBudget = 15

type SettingsRandomizer struct {
	oot *Randomizer
}

func NewSettingsRandomizer(version string) *SettingsRandomizer {
	return &SettingsRandomizer{
		oot: NewRandomizer(version),
	}
}

func getShuffledSettings(seed string, cost int, baseSettingsName string) ([]byte, error) {
	base, err := getBaseDir()
	if err != nil {
		return nil, err
	}

	s, err := settings.Load(filepath.Join(base, settings.DefaultName))
	if err != nil {
		return nil, err
	}

	shuffledPatch, err := json.Marshal(s.Shuffle(seed, cost))
	if err != nil {
		return nil, err
	}

	original, err := ioutil.ReadFile(filepath.Join(base, baseSettingsName))
	if err != nil {
		return nil, err
	}

	patched, err := jsonpatch.MergePatch(original, shuffledPatch)
	if err != nil {
		return nil, err
	}

	return patched, nil
}

func getShuffledSettingsPath(seed string, cost int, baseSettingsName string) (string, error) {
	settings, err := getShuffledSettings(seed, cost, baseSettingsName)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile("", "*.settings.json")
	if err != nil {
		return "", err
	}
	settingsPath := f.Name()
	f.Close()

	if err := ioutil.WriteFile(settingsPath, settings, 0o600); err != nil {
		return "", err
	}

	return settingsPath, nil
}

func (r *SettingsRandomizer) Generate(baseSettingsName, seed string) (generator.Output, error) {
	settingsPath, err := getShuffledSettingsPath(seed, SettingsCostBudget, baseSettingsName)
	defer os.Remove(settingsPath)
	if err != nil {
		return generator.Output{}, fmt.Errorf("unable to get shuffled settings: %w", err)
	}

	outDir, err := ioutil.TempDir("", "oot-settings-randomizer-output-")
	if err != nil {
		return generator.Output{}, fmt.Errorf("unable to create output directory: %s", err)
	}
	defer os.RemoveAll(outDir)

	zpf, spoilerLog, err := r.oot.run(outDir, settingsPath, seed)
	return generator.Output{
		State:      nil,
		SeedPatch:  zpf,
		SpoilerLog: spoilerLog,
	}, err
}

func (r *SettingsRandomizer) GetDownloadURL([]byte) string {
	return ""
}

func (r *SettingsRandomizer) IsExternal() bool {
	return false
}

func (r *SettingsRandomizer) UnlockSpoilerLog([]byte) error {
	return nil
}
