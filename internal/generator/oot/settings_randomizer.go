package oot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kaepora/internal/generator"
	"kaepora/internal/generator/oot/settings"
	"os"
	"path/filepath"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
)

const (
	SettingsRandomizerName = "oot-settings-randomizer"
	SettingsCostBudget     = 20
)

// SettingsRandomizer is the "Shuffled Settings" using the local OOTR.
type SettingsRandomizer struct {
	oot *Randomizer
}

func NewSettingsRandomizer(version string) *SettingsRandomizer {
	return &SettingsRandomizer{
		oot: NewRandomizer(version),
	}
}

func getShuffledSettings(
	seed string, cost int,
	baseDir, shuffledSettingsName string,
) (map[string]interface{}, error) {
	s, err := settings.Load(filepath.Join(baseDir, shuffledSettingsName))
	if err != nil {
		return nil, err
	}

	return s.Shuffle(seed, cost), nil
}

func getMergedShuffledSettingsJSON(
	settings map[string]interface{},
	base, baseSettingsName string,
) ([]byte, error) {
	original, err := ioutil.ReadFile(filepath.Join(base, baseSettingsName))
	if err != nil {
		return nil, err
	}

	shuffledPatch, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	patched, err := jsonpatch.MergePatch(original, shuffledPatch)
	if err != nil {
		return nil, err
	}

	return patched, nil
}

func getMergedShuffledSettingsPath(
	settings map[string]interface{},
	baseDir, baseSettingsName string,
) (string, error) {
	settingsJSON, err := getMergedShuffledSettingsJSON(settings, baseDir, baseSettingsName)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile("", "*.settings.json")
	if err != nil {
		return "", err
	}
	settingsPath := f.Name()
	f.Close()

	if err := ioutil.WriteFile(settingsPath, settingsJSON, 0o600); err != nil {
		return "", err
	}

	return settingsPath, nil
}

// getBaseAndShuffledFromCombinedSettings splits the base settings file and the
// shuffled configuration from the value in the Settings field of a League.
func getBaseAndShuffledFromCombinedSettings(combined string) (string, string, error) {
	parts := strings.Split(combined, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected Settings to be in the form '<basefile.json>:<shuffled.json>', got '%s'", combined)
	}

	return parts[0], parts[1], nil
}

func (r *SettingsRandomizer) Generate(
	combinedSettingsName,
	seed string,
) (generator.Output, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return generator.Output{}, err
	}

	baseSettingsName, shuffledSettingsName, err := getBaseAndShuffledFromCombinedSettings(combinedSettingsName)
	if err != nil {
		return generator.Output{}, err
	}

	settings, err := getShuffledSettings(seed, SettingsCostBudget, baseDir, shuffledSettingsName)
	if err != nil {
		return generator.Output{}, err
	}

	settingsPath, err := getMergedShuffledSettingsPath(settings, baseDir, baseSettingsName)
	defer os.Remove(settingsPath)
	if err != nil {
		return generator.Output{}, fmt.Errorf("unable to get shuffled settings: %w", err)
	}

	outDir, err := ioutil.TempDir("", "oot-settings-randomizer-output-")
	if err != nil {
		return generator.Output{}, fmt.Errorf("unable to create output directory: %s", err)
	}
	defer os.RemoveAll(outDir)

	state, err := patchStateWithSettings(nil, settings)
	if err != nil {
		return generator.Output{}, err
	}

	zpf, spoilerLog, err := r.oot.run(outDir, settingsPath, seed)
	return generator.Output{
		State:      state,
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
