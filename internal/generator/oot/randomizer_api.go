package oot

import (
	"encoding/json"
	"errors"
	"fmt"
	"kaepora/internal/generator"
	"kaepora/pkg/ootrapi"
	"log"
	"os"
	"path/filepath"
	"time"
)

const RandomizerAPIName = "oot-randomizer-api"

type RandomizerAPI struct {
	version string
	api     *ootrapi.API
}

func NewRandomizerAPI(version string, api *ootrapi.API) *RandomizerAPI {
	return &RandomizerAPI{
		version: version,
		api:     api,
	}
}

func (g *RandomizerAPI) Generate(settingsName, seed string) (generator.Output, error) {
	settings, err := loadSettings(settingsName)
	if err != nil {
		return generator.Output{}, err
	}

	return g.generateFromSettings(settings, seed)
}

func (g *RandomizerAPI) generateFromSettings(settings map[string]interface{}, seed string) (generator.Output, error) {
	settings["seed"] = seed

	id, err := g.api.CreateSeed(g.version, settings)
	if err != nil {
		return generator.Output{}, fmt.Errorf("API error: %w", err)
	}
	if id == "" {
		return generator.Output{}, errors.New("API returned an empty seed ID")
	}

	if err := g.waitForSeedgen(id); err != nil {
		return generator.Output{}, err
	}

	spoilerLog, err := g.api.GetSeedSpoilerLog(id)
	if err != nil {
		return generator.Output{}, fmt.Errorf("API error: %w", err)
	}

	patch, err := g.api.GetSeedPatch(id)
	if err != nil {
		return generator.Output{}, fmt.Errorf("API error: %w", err)
	}

	state, err := json.Marshal(State{ID: id})
	if err != nil {
		return generator.Output{}, err
	}

	return generator.Output{
		State:      state,
		SpoilerLog: spoilerLog,
		SeedPatch:  patch,
	}, nil
}

func (g *RandomizerAPI) waitForSeedgen(id string) error {
	timeout := time.Now().Add(120 * time.Second)
	for {
		time.Sleep(5 * time.Second)

		status, err := g.api.GetSeedStatus(id)
		if err != nil {
			return fmt.Errorf("API error: %w", err)
		}

		switch status {
		case ootrapi.SeedStatusGenerating: // NOP
		case ootrapi.SeedStatusDone, ootrapi.SeedStatusDoneWithLink:
			return nil
		case ootrapi.SeedStatusFailed:
			return errors.New("API error: seed failed to generate, got SeedStatusFailed")
		}

		if time.Now().After(timeout) {
			return fmt.Errorf("reading back seed '%s' timed out", id)
		}
	}
}

func loadSettings(name string) (map[string]interface{}, error) {
	base, err := GetBaseDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Join(base, name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var ret map[string]interface{}

	if err := dec.Decode(&ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (g *RandomizerAPI) GetDownloadURL(stateJSON []byte) string {
	var state State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		log.Printf("error: unable to unmarshal state JSON: %s", err)
		return ""
	}

	return fmt.Sprintf("https://ootrandomizer.com/seed/get?id=%s", state.ID)
}

func (g *RandomizerAPI) IsExternal() bool {
	return true
}

func (g *RandomizerAPI) UnlockSpoilerLog(stateJSON []byte) error {
	var state State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return fmt.Errorf("error: unable to unmarshal state JSON: %s", err)
	}

	return g.api.UnlockSeedSpoilerLog(state.ID)
}
