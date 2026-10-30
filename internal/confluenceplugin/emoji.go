package confluenceplugin

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
)

//go:embed emoji.json
var emojiJson []byte 

var emojiMap map[string]string
var emojiOnce sync.Once

type EmojiData struct {
	Unicode string `json:"unicode"`
}

func loadEmojis() {
	defer func() { emojiJson = nil }()
	emojiMap = make(map[string]string)
	if err := json.Unmarshal(emojiJson, &emojiMap); err != nil {
		log.Error().Err(err).Msg("error loading emoji.json")
		return
	}
}

func GetEmoji(unicode string) string {
	emojiOnce.Do(loadEmojis)
	if val, ok := emojiMap[unicode]; ok {
		return val
	}
	return ":interrobang:"
}
