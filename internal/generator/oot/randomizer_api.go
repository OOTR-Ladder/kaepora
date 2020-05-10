package oot

import (
	"errors"
	"kaepora/internal/generator"
	"kaepora/pkg/ootrapi"
)

type RandomizerAPI struct {
	version string
	api     *ootrapi.API
}

func NewRandomizerAPI(version string) *RandomizerAPI {
	return &RandomizerAPI{
		version: version,
	}
}

func (g *RandomizerAPI) Generate(settingsName, seed string) (generator.Output, error) {
	return generator.Output{}, errors.New("not implemented")
}

func (g *RandomizerAPI) SetAPI(ootrAPI *ootrapi.API) {
	g.api = ootrAPI
}
