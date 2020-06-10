package oot

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SpoilerLog struct {
	Version        string   `json:":version"`
	FileHash       []string `json:"file_hash"`
	Seed           string   `json:":seed"`
	SettingsString string   `json:":settings_string"`
	Settings       json.RawMessage

	ItemPool      map[string]int                            `json:"item_pool"`
	Locations     map[string]SpoilerLogItem                 `json:"locations"`
	WOTHLocations map[string]SpoilerLogItem                 `json:":woth_locations"`
	BarrenRegions []string                                  `json:":barren_regions"`
	GossipStones  map[string]SpoilerLogGossip               `json:"gossip_stones"`
	Playthrough   map[json.Number]map[string]SpoilerLogItem `json:":playthrough"`
}

type SpoilerLogItem string
type SpoilerLogItemCategory int

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

type SpoilerLogGossip string

func (g *SpoilerLogGossip) UnmarshalJSON(raw []byte) error {
	var stone struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &stone); err == nil {
		*g = SpoilerLogGossip(stone.Text)
		return nil
	}

	return fmt.Errorf("unable to parse gossip: %s", string(raw))
}
