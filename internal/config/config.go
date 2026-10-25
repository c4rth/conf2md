package config

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type CmdLineArgs struct {
	Page          string 
	Token         string 
	OutputDir     string 
	ConfluenceUrl string 
	ExportChildren bool  
	Local          bool  
	Mkdocs         bool  
	Debug          bool  
}

func (args *CmdLineArgs) Validate() error {
	log.Info().Interface("args", args).Msg("Validating command line arguments")
	if (args.Page == "") {
		return fmt.Errorf("page is required")
	}
	if (!args.Local && args.ConfluenceUrl == "") {
		return fmt.Errorf("Confluence URL is required in non-local mode")
	}
	return nil
}