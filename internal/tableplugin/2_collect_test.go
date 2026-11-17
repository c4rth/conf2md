package table

import "testing"

func TestReplaceNewlinesWithBreaks(t *testing.T) {
	tests := []struct {
		name string
		cell string
		want string
	}{
		{"single newline", "first\nsecond", "first<br />second"},
		{"consecutive newlines", "first\n\n\nsecond", "first<br />second"},
		{"separate newline runs", "first\n\nsecond\nthird", "first<br />second<br />third"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceNewlinesWithBreaks([]byte(tt.cell))
			if string(got) != tt.want {
				t.Errorf("replaceNewlinesWithBreaks(%q) = %q, want %q", tt.cell, got, tt.want)
			}
		})
	}
}
