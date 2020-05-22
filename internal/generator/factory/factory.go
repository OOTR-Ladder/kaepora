package factory

import (
	"fmt"
	"kaepora/internal/generator"
	"kaepora/internal/generator/oot"
	"kaepora/pkg/ootrapi"
	"strings"
)

// Factory holds the necessary state to create instances of stateful
// generators. ie. the rate limiter of the OOTR API requires us to keep a
// single instance so we have to go all java here.
type Factory struct {
	ootrAPI *ootrapi.API
}

func New(ootrAPI *ootrapi.API) Factory {
	return Factory{
		ootrAPI: ootrAPI,
	}
}

func (f Factory) NewGenerator(id string) (generator.Generator, error) {
	switch name, version := parseID(id); name {
	case oot.RandomizerName:
		return oot.NewRandomizer(version), nil
	case oot.RandomizerAPIName:
		return oot.NewRandomizerAPI(version, f.ootrAPI), nil
	case oot.SettingsRandomizerName:
		return oot.NewSettingsRandomizer(version), nil
	case oot.SettingsRandomizerAPIName:
		return oot.NewSettingsRandomizerAPI(version, f.ootrAPI), nil
	case "test":
		return generator.NewTest(), nil
	default:
		return nil, fmt.Errorf("unknown generator: %s", id)
	}
}

func parseID(id string) (name, version string) {
	parts := strings.SplitN(id, ":", 2)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[1]
	}
}
