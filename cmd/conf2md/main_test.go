package main

import "testing"

func TestDebugFlagAcceptsLevelParameter(t *testing.T) {
	for _, name := range []string{"--debug", "-d"} {
		command := newRootCommand()
		if err := command.ParseFlags([]string{name, "trace"}); err != nil {
			t.Fatalf("ParseFlags(%q, trace) error = %v", name, err)
		}
		if got := command.Flags().Lookup("debug").Value.String(); got != "trace" {
			t.Errorf("debug flag value = %q, want trace", got)
		}
	}
}

func TestDebugFlagDefaultsToDebugWithoutLevel(t *testing.T) {
	for _, name := range []string{"--debug", "-d"} {
		command := newRootCommand()
		if err := command.ParseFlags(defaultDebugLevel([]string{name})); err != nil {
			t.Fatalf("ParseFlags(%q) error = %v", name, err)
		}
		if got := command.Flags().Lookup("debug").Value.String(); got != "debug" {
			t.Errorf("debug flag value = %q, want debug", got)
		}
	}
}

func TestDefaultDebugLevelPreservesExplicitLevel(t *testing.T) {
	args := []string{"-d", "trace"}
	if got := defaultDebugLevel(args); len(got) != len(args) || got[0] != "-d" || got[1] != "trace" {
		t.Errorf("defaultDebugLevel(%v) = %v, want unchanged args", args, got)
	}
}
