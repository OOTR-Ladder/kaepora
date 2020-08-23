package back

import (
	"encoding/json"
	"fmt"
	"kaepora/internal/generator/oot"
	"os"
)

// SettingsDocumentation is the human-readable version of OOTR parameters.
type SettingsDocumentation map[string]SettingsDocumentationEntry

// SettingsDocumentationEntry is the entry for a single parameter and all its possible values.
type SettingsDocumentationEntry struct {
	Title  string
	Values []SettingsDocumentationValueEntry
}

// GetValueEntry returns the documentation entry of a setting value.
func (e *SettingsDocumentationEntry) GetValueEntry(value interface{}) SettingsDocumentationValueEntry {
	for _, entry := range e.Values {
		if value == entry.Value {
			return entry
		}
	}

	return SettingsDocumentationValueEntry{}
}

// SettingsDocumentationValueEntry is the documentation for a single parameter value.
type SettingsDocumentationValueEntry struct {
	Value              interface{}
	Title, Description string
}

// LoadSettingsDocumentation loads a localized documentation from file.
func LoadSettingsDocumentation(locale string) (SettingsDocumentation, error) {
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

	var doc SettingsDocumentation
	dec := json.NewDecoder(f)
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}

	return doc, nil
}
