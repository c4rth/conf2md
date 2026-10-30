package confluenceplugin

import "testing"

func TestGetEmoji(t *testing.T) {
	for range 8 {
		t.Run("concurrent lookup", func(t *testing.T) {
			t.Parallel()
			if got := GetEmoji("1f4af"); got != ":100:" {
				t.Errorf("GetEmoji(1f4af) = %q, want :100:", got)
			}
			if got := GetEmoji("unknown"); got != ":interrobang:" {
				t.Errorf("GetEmoji(unknown) = %q, want :interrobang:", got)
			}
			if emojiJson != nil {
				t.Error("embedded JSON buffer retained after initialization")
			}
		})
	}
}
