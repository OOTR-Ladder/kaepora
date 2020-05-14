//+build api

package oot_test

import (
	"kaepora/internal/generator/oot"
	"kaepora/pkg/ootrapi"
	"os"
	"testing"
)

func TestCreateSeed(t *testing.T) {
	api := loadAPI(t)

	testCreateSeed_inner(t, api)

	/*
		for i := 0; i < 10; i++ {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()
				testCreateSeed_inner(t, api)
			})
		}
	*/
}

func testCreateSeed_inner(t *testing.T, api *ootrapi.API) {
	g := oot.NewRandomizerAPI("5.2.0", api)
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

func loadAPI(t *testing.T) *ootrapi.API {
	key := os.Getenv("KAEPORA_OOTR_API_KEY")
	if key == "" {
		t.Skip("KAEPORA_OOTR_API_KEY not provided")
	}

	return ootrapi.New(key)
}
