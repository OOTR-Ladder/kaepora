package oot

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SpoilerLog is a the solution to a OOTR seed.
type SpoilerLog struct {
	Version        string   `json:":version"`
	FileHash       []string `json:"file_hash"`
	Seed           string   `json:":seed"`
	SettingsString string   `json:":settings_string"`

	Entrances     map[Exit]interface{}                                  `json:"entrances"`
	Settings      map[string]interface{}                                `json:"settings"`
	ItemPool      map[string]int                                        `json:"item_pool"`
	Locations     map[SpoilerLogLocation]SpoilerLogItem                 `json:"locations"`
	WOTHLocations map[SpoilerLogLocation]SpoilerLogItem                 `json:":woth_locations"`
	BarrenRegions []string                                              `json:":barren_regions"`
	GossipStones  map[string]SpoilerLogGossip                           `json:"gossip_stones"`
	Playthrough   map[json.Number]map[SpoilerLogLocation]SpoilerLogItem `json:":playthrough"`
}

// SpoilerLogGossip is the text that is shown when interacting with a Gossip Stone.
type SpoilerLogGossip struct {
	Text   string   `json:"text"`
	Colors []string `json:"colors"`
}

// Exit is a Scene exit/entrance.
type Exit string

// Scene, Exit string

type (
	// SpoilerLogItem is the name of an item that can be randomized in a SpoilerLogLocation.
	SpoilerLogItem string

	// SpoilerLogLocation is the name of a location where a random item can be placed.
	SpoilerLogLocation string

	// SpoilerLogItemCategory is a category a SpoilerLogItem can belong to.
	SpoilerLogItemCategory int
)

func (e Exit) Scene() string {
	parts := strings.Split(string(e), " -> ")
	if len(parts) != 2 {
		panic(fmt.Errorf("expected 2 parts, got %d", len(parts)))
	}

	return parts[0]
}

func (e Exit) Exit() string {
	parts := strings.Split(string(e), " -> ")
	if len(parts) != 2 {
		panic(fmt.Errorf("expected 2 parts, got %d", len(parts)))
	}

	return parts[1]
}

// Spheres returns the playthrough as an ordered slice. The playthrough is
// returned as a map with numeric keys a string which makes iterating over it
// in order impossible.
func (s SpoilerLog) Spheres() []map[SpoilerLogLocation]SpoilerLogItem {
	ret := make([]map[SpoilerLogLocation]SpoilerLogItem, len(s.Playthrough))

	for k := range s.Playthrough {
		iK, _ := k.Int64()
		ret[iK-1] = s.Playthrough[k]
	}

	return ret
}

// Possible item categories, these are arbitrary.
const (
	SpoilerLogItemCategoryItem SpoilerLogItemCategory = iota
	SpoilerLogItemCategoryBossKey
	SpoilerLogItemCategoryIceTrap
	SpoilerLogItemCategoryJunk
	SpoilerLogItemCategoryMedallion
	SpoilerLogItemCategoryPoH
	SpoilerLogItemCategoryBombchu
	SpoilerLogItemCategorySmallKey
	SpoilerLogItemCategorySong
	SpoilerLogItemCategoryTriforce

	SpoilerLogItemCategoryCount // keep this last
)

// GetCategory returns the category the item belongs to.
func (i SpoilerLogItem) GetCategory() SpoilerLogItemCategory {
	if strings.HasPrefix(string(i), "Small Key") {
		return SpoilerLogItemCategorySmallKey
	}
	if strings.HasPrefix(string(i), "Boss Key") {
		return SpoilerLogItemCategoryBossKey
	}
	if strings.HasPrefix(string(i), "Bombchus") {
		return SpoilerLogItemCategoryBombchu
	}
	if strings.HasSuffix(string(i), "Medallion") {
		return SpoilerLogItemCategoryMedallion
	}

	switch i {
	case
		"Arrows (10)", "Arrows (30)", "Arrows (5)",
		"Bombs (10)", "Bombs (20)", "Bombs (5)",
		"Deku Nuts (10)", "Deku Nuts (5)",
		"Deku Seeds (30)", "Deku Stick (1)",
		"Recovery Heart",
		"Rupee (1)", "Rupees (5)", "Rupees (50)",
		"Rupees (20)", "Rupees (200)":
		return SpoilerLogItemCategoryJunk

	case
		"Zeldas Lullaby", "Eponas Song", "Sarias Song",
		"Suns Song", "Song of Time", "Song of Storms",
		"Minuet of Forest", "Bolero of Fire", "Serenade of Water",
		"Nocturne of Shadow", "Requiem of Spirit", "Prelude of Light":
		return SpoilerLogItemCategorySong

	case
		"Kokiri Emerald", "Goron Ruby", "Zora Sapphire":
		return SpoilerLogItemCategoryMedallion
	case
		"Piece of Heart", "Piece of Heart (Treasure Chest Game)",
		"Heart Container", "Double Defense":
		return SpoilerLogItemCategoryPoH
	case "Ice Trap":
		return SpoilerLogItemCategoryIceTrap
	case "Triforce Piece":
		return SpoilerLogItemCategoryTriforce
	default:
		return SpoilerLogItemCategoryItem
	}
}

func (i *SpoilerLogItem) UnmarshalJSON(raw []byte) error {
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		*i = SpoilerLogItem(str)
		return nil
	}

	var sold struct {
		Item string `json:"item"`
	}
	if err := json.Unmarshal(raw, &sold); err == nil {
		*i = SpoilerLogItem(sold.Item)
		return nil
	}

	return fmt.Errorf("unable to parse item: %s", string(raw))
}
