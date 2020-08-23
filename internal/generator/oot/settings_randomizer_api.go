package oot

import (
	"encoding/json"
	"kaepora/internal/generator"
	"kaepora/pkg/ootrapi"
)

const SettingsRandomizerAPIName = "oot-settings-randomizer-api"

// SettingsRandomizerAPI is the "Shuffled Settings" using the OOTR API.
type SettingsRandomizerAPI struct {
	oot *RandomizerAPI
}

func NewSettingsRandomizerAPI(version string, api *ootrapi.API) *SettingsRandomizerAPI {
	return &SettingsRandomizerAPI{
		oot: NewRandomizerAPI(version, api),
	}
}

func (r *SettingsRandomizerAPI) Generate(baseSettingsName, seed string) (generator.Output, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return generator.Output{}, err
	}

	settingsDiff, err := getShuffledSettings(seed, SettingsCostBudget, baseDir)
	if err != nil {
		return generator.Output{}, err
	}

	settingsJSON, err := getMergedShuffledSettingsJSON(settingsDiff, baseDir, baseSettingsName)
	if err != nil {
		return generator.Output{}, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return generator.Output{}, err
	}

	out, err := r.oot.generateFromSettings(settings, seed)
	if err != nil {
		return generator.Output{}, err
	}

	out.State, err = patchStateWithSettings(out.State, settingsDiff)
	if err != nil {
		return generator.Output{}, err
	}

	return out, nil
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

func patchStateWithSettings(stateJSON []byte, settings map[string]interface{}) ([]byte, error) {
	if len(stateJSON) == 0 {
		return json.Marshal(State{SettingsPatch: settings})
	}

	var state State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, err
	}

	state.SettingsPatch = settings

	return json.Marshal(state)
}
