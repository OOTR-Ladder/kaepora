//+build docker

package oot_test

import (
	"encoding/json"
	"kaepora/internal/generator/factory"
	"kaepora/internal/generator/oot"
	"testing"
)

func TestOOTSettingsRandomizer(t *testing.T) {
	t.Parallel()

	f := factory.New(nil)
	g, err := f.NewGenerator(oot.SettingsRandomizerName + ":5.2.13")
	if err != nil {
		t.Fatal(err)
	}

	out, err := g.Generate("s3.json", "DEADBEEF")
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

	if state.ID != "" {
		t.Errorf("OOTR seed ID is not empty")
	}
}
