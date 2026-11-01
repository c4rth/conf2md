package confluence

import (
	"context"
	"fmt"
	"strings"

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

func GetPageID(ctx context.Context, api *goconfluence.API, spaceKey string, pageTitle string) (string, error) {
	spaceKey = strings.Trim(spaceKey, `'"`)
	pageTitle = strings.Trim(pageTitle, `'"`)
	s, err := api.GetContent(ctx, goconfluence.ContentQuery{
		SpaceKey: spaceKey,
		Title:    pageTitle,
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
	return s.Results[0].ID, nil
}
