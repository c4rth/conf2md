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
