package settings_test

import (
	"kaepora/internal/generator/oot/settings"
	"os"
	"reflect"
	"testing"
)

func TestShuffle(t *testing.T) {
	if err := os.Chdir("../../../.."); err != nil { // CWD dependant
		t.Fatal(err)
	}

	// HARDCODED
	s, err := settings.Load("resources/oot-randomizer/shu-shuffled.json")
	if err != nil {
		t.Fatal(err)
	}

	if len(s) == 0 {
		t.Fatal("empty settings")
	}

	shuf1 := s.Shuffle("seed", 20)
	if len(shuf1) == 0 {
		t.Error("empty shuffled settings")
	}
	if len(shuf1) == len(s) {
		t.Error("too many settings")
	}

	shuf2 := s.Shuffle("seed", 20)
	if !reflect.DeepEqual(shuf1, shuf2) {
		t.Error("same seed produced different settings")
	}

	shuf3 := s.Shuffle("Seed", 20)
	if reflect.DeepEqual(shuf2, shuf3) {
		t.Error("different seed produced same settings")
	}

	shuf4 := s.Shuffle("Seed", 40)
	if reflect.DeepEqual(shuf3, shuf4) {
		t.Error("diffent weights produced same settings")
	}
}
