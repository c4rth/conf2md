//go:build emojigen
 
package confluenceplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const emojiDBURL = "https://raw.githubusercontent.com/facelessuser/pymdown-extensions/main/pymdownx/emoji1_db.py"

// TestGenerateEmojiMap regenerates emoji.json from the
// pymdown-extensions Emojione database. It needs network access, so it only
// runs with the emojigen build tag.
func TestGenerateEmojiMap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	t.Log("starting emoji generation")
	t.Log("fetching emoji database")
	source, err := fetchEmojiDB(ctx, emojiDBURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("extracting emoji dictionary")
	emojiJSON, err := extractEmojiDict(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("validating extracted emoji dictionary")
	var emojis map[string]EmojiData
	if err := json.Unmarshal(emojiJSON, &emojis); err != nil {
		t.Fatalf("emoji dict is not valid JSON: %v", err)
	}
	if len(emojis) == 0 || emojis[":100:"].Unicode != "1f4af" {
		t.Fatalf("emoji dict has unexpected content (%d entries)", len(emojis))
	}
	t.Log("building emoji map")
	emojiMap = make(map[string]string)
	for key, data := range emojis {
		if data.Unicode != "" {
			emojiMap[data.Unicode] = key
		}
	}
	t.Logf("built emoji map with %d entries", len(emojiMap))
	emojiJSON, err = json.Marshal(emojiMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("emoji.json", emojiJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d emojis to emoji.json", len(emojis))
}

func fetchEmojiDB(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// extractEmojiDict returns the literal assigned to the top-level `emoji`
// variable. The upstream file writes this dict as JSON, so it is kept verbatim.
func extractEmojiDict(source []byte) ([]byte, error) {
	const startMarker = "\nemoji = "
	start := bytes.Index(source, []byte(startMarker))
	if start < 0 {
		return nil, fmt.Errorf("emoji dict not found")
	}
	start += len(startMarker)
	// The dict ends at the first closing brace at the start of a line.
	end := bytes.Index(source[start:], []byte("\n}"))
	if end < 0 {
		return nil, fmt.Errorf("end of emoji dict not found")
	}
	return source[start : start+end+len("\n}")], nil
}
