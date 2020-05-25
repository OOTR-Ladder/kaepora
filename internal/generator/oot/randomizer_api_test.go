//+build api

package oot_test

import (
	"encoding/json"
	"kaepora/internal/generator/oot"
	"kaepora/pkg/ootrapi"
	"os"
	"testing"
)

func TestCreateSeed(t *testing.T) {
	t.Parallel()
	api := loadAPI(t)

	testCreateSeed_inner(t, api)
}

func testCreateSeed_inner(t *testing.T, api *ootrapi.API) {
	g := oot.NewRandomizerAPI("5.2.0", api)
	out, err := g.Generate("s3.json", "DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}

	assertGeneratorOutputValid(t, out)

	var state oot.State
	if err := json.Unmarshal(out.State, &state); err != nil {
		t.Fatal(err)
	}

	if len(state.SettingsPatch) != 0 {
		t.Errorf("non empty list of shuffled settings")
	}

	if state.ID == "" {
		t.Errorf("empty OOTR seed ID")
	}
}

var ootrAPI *ootrapi.API

func loadAPI(t *testing.T) *ootrapi.API {
	if ootrAPI != nil {
		return ootrAPI
	}

	key := os.Getenv("KAEPORA_OOTR_API_KEY")
	if key == "" {
		t.Skip("KAEPORA_OOTR_API_KEY not provided")
	}

	ootrAPI = ootrapi.New(key)
	return ootrAPI
}
