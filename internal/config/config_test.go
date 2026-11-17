package config

import "testing"

func TestNormalizeDebugLevel(t *testing.T) {
	tests := map[string]string{
		"":       "",
		"debug":  "debug",
		"TRACE":  "trace",
		" true ": "debug",
		"false":  "",
	}
	for input, want := range tests {
		if got := NormalizeDebugLevel(input); got != want {
			t.Errorf("NormalizeDebugLevel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidateDebugLevel(t *testing.T) {
	for _, level := range []string{"", "debug", "trace"} {
		if err := (&CmdLineArgs{DebugLevel: level}).ValidateDebugLevel(); err != nil {
			t.Errorf("ValidateDebugLevel(%q) returned an error: %v", level, err)
		}
	}
	if err := (&CmdLineArgs{DebugLevel: "verbose"}).ValidateDebugLevel(); err == nil {
		t.Error("ValidateDebugLevel(\"verbose\") returned no error")
	}
}
