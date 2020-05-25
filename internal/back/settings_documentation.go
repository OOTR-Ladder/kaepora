package back

import (
	"encoding/json"
	"fmt"
	"kaepora/internal/generator/oot"
	"os"
)

type settingsDocumentation map[string]settingsDocumentationEntry

type settingsDocumentationEntry struct {
	Title  string
	Values []settingsDocumentationValueEntry
}

func (e *settingsDocumentationEntry) getValueEntry(value interface{}) settingsDocumentationValueEntry {
	for _, entry := range e.Values {
		if value == entry.Value {
			return entry
		}
	}

	return settingsDocumentationValueEntry{}
}

type settingsDocumentationValueEntry struct {
	Value              interface{}
	Title, Description string
}

func loadSettingsDocumentation(locale string) (settingsDocumentation, error) {
	dir, err := oot.GetBaseDir()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/settings_documentation.%s.json", dir, locale)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var doc settingsDocumentation
	dec := json.NewDecoder(f)
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}

	return doc, nil
}
