package confluence

import (
	"github.com/rs/zerolog/log"

	"conf2md/internal/config"
	"conf2md/internal/goconfluence"
)

func NewConfluenceAPI(args *config.CmdLineArgs) (*goconfluence.API, error) {
	if args.Debug {
		goconfluence.SetDebug(true)
	}
	api, err := goconfluence.NewAPI(args.ConfluenceUrl, "", args.Token)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Confluence API")
		return nil, err
	}
	api.VerifyTLS(false)
	return api, nil
}

func GetConfluencePage(api *goconfluence.API, pageID string) (*goconfluence.Content, error) {
	c, err := api.GetContentByID(pageID, goconfluence.ContentQuery{
		Expand: []string{"body.storage", "children.page"},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to get content by ID")
		return nil, err
	}

	return c, nil
}

func GetChildPages(api *goconfluence.API, parentID string) (*goconfluence.Search, error) {
	c, err := api.GetChildPages(parentID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get children")
		return nil, err
	}

	return c, nil
}
