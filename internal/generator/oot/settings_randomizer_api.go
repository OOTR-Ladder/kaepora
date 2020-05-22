package oot

import (
	"encoding/json"
	"kaepora/internal/generator"
	"kaepora/pkg/ootrapi"
)

const SettingsRandomizerAPIName = "oot-settings-randomizer-api"

type SettingsRandomizerAPI struct {
	oot *RandomizerAPI
}

func NewSettingsRandomizerAPI(version string, api *ootrapi.API) *SettingsRandomizerAPI {
	return &SettingsRandomizerAPI{
		oot: NewRandomizerAPI(version, api),
	}
}

func (r *SettingsRandomizerAPI) Generate(baseSettingsName, seed string) (generator.Output, error) {
	settingsJSON, err := getShuffledSettings(seed, SettingsCostBudget, baseSettingsName)
	if err != nil {
		return generator.Output{}, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return generator.Output{}, err
	}

	return r.oot.generateFromSettings(settings, seed)
}

func (r *SettingsRandomizerAPI) GetDownloadURL(stateJSON []byte) string {
	return r.oot.GetDownloadURL(stateJSON)
}

func (r *SettingsRandomizerAPI) IsExternal() bool {
	return r.oot.IsExternal()
}

func (r *SettingsRandomizerAPI) UnlockSpoilerLog(stateJSON []byte) error {
	return r.oot.UnlockSpoilerLog(stateJSON)
}
