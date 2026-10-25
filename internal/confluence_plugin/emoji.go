package confluence_plugin

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed emoji.json
var emojiJson []byte

var emojiMap map[string]string

type EmojiData struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Unicode  string `json:"unicode"`
}

func init() {
	var rawMap map[string]EmojiData
	if err := json.Unmarshal(emojiJson, &rawMap); err != nil {
		fmt.Println("Error loading emoji.json:", err)
		return
	}

	emojiMap = make(map[string]string)
	for key, data := range rawMap {
		if data.Unicode != "" {
			emojiMap[data.Unicode] = key
		}
	}
}

// GetEmoji returns the emoji shortcode (e.g. :100:) for a given unicode hex string.
func GetEmoji(unicode string) string {
	if val, ok := emojiMap[unicode]; ok {
		return val
	}
	return ""
}
