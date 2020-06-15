package back // nolint:testpackage

import (
	"os"
	"testing"
)

func TestSettingsDocumentation(t *testing.T) {
	if err := os.Chdir("../../"); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSettingsDocumentation("en")
	if err != nil {
		t.Fatal(err)
	}

	if len(doc) == 0 {
		t.Fatal("empty documentation")
	}

	s := doc["starting_age"]
	v := s.GetValueEntry("adult")

	if s.Title != "Starting age" {
		t.Errorf("expected 'Starting Age' got '%s'", s.Title)
	}
	if v.Title != "adult" {
		t.Errorf("expected 'adult' got '%s'", v.Title)
	}

	s = doc["shopsanity"]
	v = s.GetValueEntry("4")

	if s.Title != "Shopsanity" {
		t.Errorf("expected 'Shopsanity' got '%s'", s.Title)
	}
	if v.Title != "4" {
		t.Errorf("expected '4' got '%s'", v.Title)
	}
	if s.GetValueEntry(4).Title != "" {
		t.Errorf("bad type int for shopsanity should have no value")
	}
}
