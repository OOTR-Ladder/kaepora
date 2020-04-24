package settings

import (
	"os"
	"testing"
)

func TestShuffle(t *testing.T) {
	if err := os.Chdir("../../../.."); err != nil { // CWD dependant
		t.Fatal(err)
	}

	s, err := Load("resources/oot-randomizer/" + DefaultName)
	if err != nil {
		t.Fatal(err)
	}

	if len(s) == 0 {
		t.Fatal("empty settings")
	}

	shuf := s.Shuffle("seed", 20)
	if len(shuf) == 0 {
		t.Error("empty shuffled settings")
	}

	if len(shuf) == len(s) {
		t.Error("too many settings")
	}
}
