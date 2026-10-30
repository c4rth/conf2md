package storage

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "A  ----B.txt", want: "a-b.txt"},
		{name: " .hidden ", want: "hidden"},
		{name: "----", want: "untitled"},
		{name: "...", want: "untitled"},
		{name: "é漢字", want: "untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeFilename(tt.name); got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
