package oot

import (
	"encoding/json"
	"fmt"
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
