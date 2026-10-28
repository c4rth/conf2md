package config

import (
	"fmt"
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
	Debug          bool
	Timeout        time.Duration
}

func (args *CmdLineArgs) Validate() error {
	log.Info().Interface("args", args).Msg("validating command line arguments")
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
