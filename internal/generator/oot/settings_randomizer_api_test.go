//+build api

package oot_test

import (
	"encoding/json"
	"kaepora/internal/generator/oot"
	"kaepora/pkg/ootrapi"
	"testing"
)

func TestCreateSettingsRandomizerAPISeed(t *testing.T) {
	t.Parallel()
	api := loadAPI(t)

	testCreateSettingsRandomizerAPISeed_inner(t, api)
}

func testCreateSettingsRandomizerAPISeed_inner(t *testing.T, api *ootrapi.API) {
	g := oot.NewSettingsRandomizerAPI("5.2.0", api)
	out, err := g.Generate("s3.json:shu-shuffled.json", "DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}

	assertGeneratorOutputValid(t, out)

	var state oot.State
	if err := json.Unmarshal(out.State, &state); err != nil {
		t.Fatal(err)
	}

	if len(state.SettingsPatch) == 0 {
		t.Errorf("empty list of shuffled settings")
	}

	if state.ID == "" {
		t.Errorf("empty OOTR seed ID")
	}
}
