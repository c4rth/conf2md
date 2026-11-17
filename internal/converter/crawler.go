package converter

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"conf2md/internal/config"
	"conf2md/internal/confluence"
	"conf2md/internal/goconfluence"
	"conf2md/internal/mkdocs"
	"conf2md/internal/storage"
)

// Crawler holds the dependencies shared by a Confluence export.
type Crawler struct {
	api    *goconfluence.API
	args   *config.CmdLineArgs
	logger zerolog.Logger
	ctx    context.Context
}

func newCrawler(api *goconfluence.API, args *config.CmdLineArgs, ctx context.Context) *Crawler {
	return &Crawler{
		api:    api,
		args:   args,
		logger: log.With().Str("component", "crawler").Logger(),
		ctx:    ctx,
	}
}

func convertConfluence(ctx context.Context, args *config.CmdLineArgs, mkDoc *mkdocs.MkDoc) error {
	api, err := confluence.NewConfluenceAPI(args)
	if err != nil {
		return fmt.Errorf("creating Confluence API: %w", err)
	}
	crawler := newCrawler(api, args, ctx)
	pageID, err := crawler.retrievePageID()
	if err != nil {
		return err
	}
	args.Page = pageID
	return crawler.crawlConfluence(0, "docs", args.Page, "", mkDoc)
}

func (c *Crawler) retrievePageID() (string, error) {
	if c.args.Space == "" {
		return c.args.Page, nil
	}
	pageID, err := confluence.GetPageID(c.ctx, c.api, c.args.Space, c.args.Page)
	if err != nil {
		return "", err
	}
	c.logger.Info().Str("page_id", pageID).Msgf("found page ID for title '%s' in space '%s'", c.args.Page, c.args.Space)
	return pageID, nil
}

func (c *Crawler) crawlConfluence(level int, outputDir string, pageID string, pageName string, mkDoc *mkdocs.MkDoc) error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	content, err := confluence.GetConfluencePage(c.ctx, c.api, pageID, c.args.ExportChildren)
	if err != nil {
		return err
	}
	if c.args.DebugLevel != "" {
		jsonFile := filepath.Join(c.args.OutputDir, "json", pageID+".json")
		if err := storage.WriteJSONFile(jsonFile, content); err != nil {
			c.logger.Error().Err(err).Msgf("failed to write page '%s' JSON to '%s'", pageID, jsonFile)
		}

	}
	children := content.Children.Page
	outputDir, filename := pageOutputLocation(content, children, level, outputDir, pageName)
	c.logger.Info().Str("page_id", content.ID).Int("depth", level).Msgf("converting page %s", content.Title)
	if c.args.DebugLevel != "" {
		c.logger.Debug().Msgf("source for page %s:\n%s", content.Title, content.Body.Storage.Value)
	}
	childMkDoc, err := c.convertAndWritePage(content, outputDir, filename, mkDoc)
	if err != nil {
		return err
	}

	if err := c.downloadAttachments(outputDir, content); err != nil {
		return fmt.Errorf("downloading attachments for page '%s': %w", content.Title, err)
	}
	// Only page metadata is needed for recursive child exports.
	content.Body = goconfluence.Body{}

	if c.args.ExportChildren {
		if err := c.downloadChildren(level, outputDir, childMkDoc, content); err != nil {
			return fmt.Errorf("downloading child pages for page '%s': %w", content.Title, err)
		}
	}
	return nil
}

func (c *Crawler) downloadChildren(level int, outputDir string, mkDoc *mkdocs.MkDoc, content *goconfluence.Content) error {
	c.logger.Info().Str("page_id", content.ID).Msgf("checking child pages for page: %s", content.Title)
	children := content.Children.Page
	if children == nil {
		return nil
	}
	childNames := uniqueChildPageNames(children.Results)
	for i, child := range children.Results {
		c.logger.Info().Str("page_id", child.ID).Msgf("found child page: %s", child.Title)
		if err := c.crawlConfluence(level+1, outputDir, child.ID, childNames[i], mkDoc); err != nil {
			return fmt.Errorf("crawling child page '%s': %w", child.Title, err)
		}
	}
	return nil
}
