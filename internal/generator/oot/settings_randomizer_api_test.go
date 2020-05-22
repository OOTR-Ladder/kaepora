//+build api

package oot_test

import (
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
	out, err := g.Generate("s3.json", "DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}

	// patches are not reproducible so we are limited to length checks.
	if len(out.SeedPatch) == 0 {
		t.Fatal("got an empty patch")
	}

	if len(out.SpoilerLog) == 0 {
		t.Fatal("got an empty spoiler log")
	}

	if err := g.UnlockSpoilerLog(out.State); err != nil {
		t.Error(err)
	}

	if len(out.SeedPatch) < 250*1024 {
		t.Errorf("generated patch seems too small (%d bytes)", len(out.SeedPatch))
	}
	if len(out.SeedPatch) > 350*1024 {
		t.Errorf("generated patch seems too large (%d bytes)", len(out.SeedPatch))
	}
}
