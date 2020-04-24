package oot_test

import (
	"kaepora/internal/generator"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := os.Chdir("../../.."); err != nil { // generators are CWD dependant
		panic(err)
	}
	os.Exit(m.Run())
}

func TestOOTSettingsRandomizer(t *testing.T) {
	t.Parallel()

	g, err := generator.NewGenerator("oot-settings-randomizer:5.2.12")
	if err != nil {
		t.Fatal(err)
	}

	patch, spoilerLog, err := g.Generate("s3.json", "DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}

	// Patches are not reproducible so we are limited to length checks.
	if len(patch) == 0 {
		t.Fatal("got an empty patch")
	}

	if len(spoilerLog) == 0 {
		t.Fatal("got an empty spoiler log")
	}

	if len(patch) < 250*1024 {
		t.Errorf("generated patch seems too small (%d bytes)", len(patch))
	}
	if len(patch) > 350*1024 {
		t.Errorf("generated patch seems too large (%d bytes)", len(patch))
	}
}
