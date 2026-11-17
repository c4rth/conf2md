package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type CmdLineArgs struct {
	Space          string
	Page           string
	Token          string
	OutputDir      string
	ConfluenceUrl  string
	ConfluenceApi  string
	ExportChildren bool
	Local          bool
	Mkdocs         bool
	DebugLevel     string
	Timeout        time.Duration
}

func (args *CmdLineArgs) Validate() error {
	log.Info().Interface("args", args).Msg("validating command line arguments")
	if err := args.ValidateDebugLevel(); err != nil {
		return err
	}
	if args.Page == "" {
		return fmt.Errorf("page is required")
	}
	if !args.Local && args.ConfluenceUrl == "" {
		return fmt.Errorf("confluence URL is required in non-local mode")
	}
	if args.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	return nil
}

func (args *CmdLineArgs) ValidateDebugLevel() error {
	if args.DebugLevel != "" && args.DebugLevel != "debug" && args.DebugLevel != "trace" {
		return fmt.Errorf("debug level must be 'debug' or 'trace', got %q", args.DebugLevel)
	}
	return nil
}

func NormalizeDebugLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "true":
		return "debug"
	case "false":
		return ""
	default:
		return level
	}
}
