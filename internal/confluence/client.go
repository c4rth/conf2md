package confluence

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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

func GetConfluencePage(ctx context.Context, api *goconfluence.API, pageID string, includeChildren bool) (*goconfluence.Content, error) {
	expand := []string{"body.storage", "children.page"}
	if includeChildren {
		expand[1] = "children.page.extensions.position"
	}
	c, err := api.GetContentByID(ctx, pageID, goconfluence.ContentQuery{
		Expand: expand,
	})
	if err != nil {
		return nil, fmt.Errorf("get content by ID: %w", err)
	}
	if includeChildren {
		if err := populateOrderedChildPages(ctx, api, c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func populateOrderedChildPages(ctx context.Context, api *goconfluence.API, page *goconfluence.Content) error {
	if page == nil || page.Children == nil || page.Children.Page == nil {
		return nil
	}
	children := page.Children.Page
	if len(children.Results) == 0 {
		return nil
	}
	if needsFullChildPagesFetch(children) {
		allChildren, err := api.GetChildPages(ctx, page.ID)
		if err != nil {
			return fmt.Errorf("get child pages for %s: %w", page.ID, err)
		}
		children = allChildren
		page.Children.Page = children
	}
	SortChildPages(children.Results)
	return nil
}

func needsFullChildPagesFetch(children *goconfluence.Search) bool {
	if children.Links.Next != "" {
		return true
	}
	if children.Limit > 0 && len(children.Results) >= children.Limit {
		return true
	}
	for _, child := range children.Results {
		if child.Extensions == nil || len(bytes.TrimSpace(child.Extensions.Position)) == 0 {
			return true
		}
	}
	return false
}

// SortChildPages sorts Confluence child pages in the order shown in the Confluence sidebar:
// pages with an explicit numeric extensions.position come first (ascending), followed by
// pages without an explicit position ("none" or null) sorted case-insensitively by title.
func SortChildPages(children []goconfluence.Results) {
	slices.SortStableFunc(children, func(a, b goconfluence.Results) int {
		posA, hasPosA := childPagePosition(a)
		posB, hasPosB := childPagePosition(b)
		if hasPosA && hasPosB {
			if c := cmp.Compare(posA, posB); c != 0 {
				return c
			}
		} else if hasPosA != hasPosB {
			if hasPosA {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)); c != 0 {
			return c
		}
		return cmp.Compare(a.Title, b.Title)
	})
}

func childPagePosition(child goconfluence.Results) (int64, bool) {
	if child.Extensions == nil {
		return 0, false
	}
	raw := bytes.TrimSpace(child.Extensions.Position)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil || s == "" || strings.EqualFold(s, "none") {
			return 0, false
		}
		pos, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return pos, true
	}
	var num json.Number
	if err := json.Unmarshal(raw, &num); err != nil {
		return 0, false
	}
	if pos, err := num.Int64(); err == nil {
		return pos, true
	}
	f, err := num.Float64()
	if err != nil {
		return 0, false
	}
	return int64(f), true
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
