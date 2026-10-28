package confluence

import (
	"context"
	"fmt"

	"conf2md/internal/config"
	"conf2md/internal/goconfluence"
	"github.com/rs/zerolog/log"
)

func NewConfluenceAPI(args *config.CmdLineArgs) (*goconfluence.API, error) {
	api, err := goconfluence.NewAPI(args.ConfluenceUrl, args.ConfluenceApi, "", args.Token)
	if err != nil {
		return nil, err
	}
	api.VerifyTLS(false)
	api.SetTimeout(args.Timeout)
	return api, nil
}

func GetConfluencePage(ctx context.Context, api *goconfluence.API, pageID string) (*goconfluence.Content, error) {
	c, err := api.GetContentByID(ctx, pageID, goconfluence.ContentQuery{
		Expand: []string{"body.storage", "children.page"},
	})
	if err != nil {
		return nil, fmt.Errorf("get content by ID: %w", err)
	}

	return c, nil
}

func SearchPageID(ctx context.Context, api *goconfluence.API, spaceKey string, pageTitle string) (string, error) {
	s, err := api.Search(ctx, goconfluence.SearchQuery{
		CQL: fmt.Sprintf("space=\"%s\"+AND+title=\"%s\"", spaceKey, pageTitle),
	})
	if err != nil {
		return "", fmt.Errorf("searching page ID: %w", err)
	}
	if len(s.Results) == 0 {
		return "", fmt.Errorf("no page found with title '%s' in space '%s'", pageTitle, spaceKey)
	}
	if len(s.Results) > 1 {
		log.Warn().Msgf("multiple pages found with title '%s' in space '%s' -> take the 1st one", pageTitle, spaceKey)
	}
	return s.Results[0].Content.ID, nil
}
