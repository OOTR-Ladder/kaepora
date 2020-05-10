package generator

import (
	"encoding/json"
)

// Output is the the full set of data that comes out of a generator.
// Mostly here to avoid having to return n fields.
type Output struct {
	State      []byte
	SeedPatch  []byte
	SpoilerLog []byte
}

type Generator interface {
	Generate(settings, seed string) (Output, error)

	// GetDownloadURL returns an URL where the user can download its seed.  If
	// there is no external service to download the seed, this will return an
	// empty string.
	GetDownloadURL(state []byte) string

	// IsExternal returns true if the generation is not done locally but
	// through an external service (API).
	IsExternal() bool
}

type Test struct{}

func NewTest() *Test {
	return &Test{}
}

func (*Test) Generate(settings, seed string) (Output, error) {
	spoilerStruct := struct {
		Hash []string `json:"file_hash"`
	}{
		Hash: []string{"hash", "for", "seed", seed},
	}

	spoilerLog, err := json.Marshal(spoilerStruct)
	if err != nil {
		return Output{}, err
	}

	return Output{
		State:      nil,
		SeedPatch:  []byte("generated binary for seed: " + seed),
		SpoilerLog: spoilerLog,
	}, nil
}

func (*Test) GetDownloadURL([]byte) string {
	return ""
}

func (*Test) IsExternal() bool {
	return false
}
