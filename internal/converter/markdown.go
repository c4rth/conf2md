package converter

import (
	"context"
	"fmt"
	stdhtml "html"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"

	"conf2md/internal/config"
	"conf2md/internal/confluence"
	"conf2md/internal/confluenceplugin"
	"conf2md/internal/gliffy"
	"conf2md/internal/goconfluence"
	"conf2md/internal/mkdocs"
	"conf2md/internal/storage"
	"conf2md/internal/tableplugin"
)

var plainTextBody = regexp.MustCompile(
	`(?s)<ac:plain-text-body>\s*<!\[CDATA\[(.*?)\]\]>\s*</ac:plain-text-body>`,
)

// emptyParagraph matches <p></p>, <p> </p>, <p/> and the same forms with attributes.
var emptyParagraph = regexp.MustCompile(`(?i)<p(?:\s[^>]*)?(?:/>|>\s*</p\s*>)`)

var confluenceConverter *converter.Converter

func init() {
	confluenceConverter = converter.NewConverter(
		converter.WithPlugins(
			confluenceplugin.NewConfluencePlugin(),
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(
				commonmark.WithEmDelimiter("_"),
				commonmark.WithStrongDelimiter("**"),
				commonmark.WithBulletListMarker("-"),
			),
			table.NewTablePlugin(
				table.WithNewlineBehavior(table.NewlineBehaviorPreserve),
			),
		),
	)
}

func ConvertConfluenceToMarkdown(ctx context.Context, args *config.CmdLineArgs) error {
	start := time.Now()
	mkDoc := mkdocs.NewMkDocs("root", "root")
	err := error(nil)
	if args.Local {
		log.Info().Msgf("converting local file: %s", args.Page)
		err = convertLocal(args, mkDoc)
	} else {
		err = convertConfluence(ctx, args, mkDoc)
	}
	if err != nil {
		return fmt.Errorf("converting page '%s': %w", args.Page, err)
	}
	if args.Mkdocs {
		mkDocsYaml, err := mkDoc.GenerateMkDocs()
		if err != nil {
			return fmt.Errorf("generating MkDocs configuration: %w", err)
		}
		mkDocsFile := filepath.Join(args.OutputDir, "mkdocs.yml")
		if err := storage.WriteFile(mkDocsFile, []byte(mkDocsYaml)); err != nil {
			return fmt.Errorf("writing 'mkdocs.yml' to '%s': %w", mkDocsFile, err)
		}
		log.Info().Msgf("file 'mkdocs.yml' written to: %s", mkDocsFile)
	}
	end := time.Now()
	log.Info().Msgf("conversion took: %s", end.Sub(start))
	return nil
}

func convertLocal(args *config.CmdLineArgs, mkDoc *mkdocs.MkDoc) error {
	fileFullPath := args.Page
	fileName := filepath.Base(fileFullPath)
	log.Info().Msgf("reading local file: %s", fileFullPath)
	content, err := storage.ReadFile(fileFullPath)
	if err != nil {
		return fmt.Errorf("opening file '%s': %w", fileFullPath, err)
	}
	// convert
	md, err := convertConfluenceToMarkdown(string(content))
	if err != nil {
		return fmt.Errorf("converting local file '%s' to Markdown: %w", fileFullPath, err)
	}
	// output file
	outFile := filepath.Join(args.OutputDir, "docs", fileName+".md")
	// mkdocs
	mkDoc.AddChild(mkdocs.NewMkDocs(
		fileName,
		filepath.Base(fileFullPath)+".md",
	))
	// write output
	err = storage.WriteFile(outFile, md)
	if err != nil {
		return fmt.Errorf("writing Markdown file: %w", err)
	}
	log.Info().Msgf("markdown file written to: %s", outFile)
	return nil
}

func convertPlainTextBodies(input string) string {
	matches := plainTextBody.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input
	}

	var result strings.Builder
	lastEnd := 0
	for _, match := range matches {
		result.WriteString(input[lastEnd:match[0]])
		result.WriteString("<pre><code>")
		result.WriteString(stdhtml.EscapeString(input[match[2]:match[3]]))
		result.WriteString("</code></pre>")
		lastEnd = match[1]
	}
	result.WriteString(input[lastEnd:])
	return result.String()
}

func sanitizeConfluenceHTML(input string) string {
	// Code blocks are converted first so that their escaped content is never altered.
	return emptyParagraph.ReplaceAllString(convertPlainTextBodies(input), "")
}

func convertConfluenceToMarkdown(content string) ([]byte, error) {
	cleanedContent := sanitizeConfluenceHTML(content)
	doc, err := html.Parse(strings.NewReader(cleanedContent))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	markdown, err := confluenceConverter.ConvertNode(doc)
	if err != nil {
		return nil, fmt.Errorf("converting HTML to Markdown: %w", err)
	}
	return markdown, nil
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

func (c *Crawler) retrievePageID() (string, error) {
	if c.args.Space == "" {
		return c.args.Page, nil
	}
	pageID, err := confluence.SearchPageID(c.ctx, c.api, c.args.Space, c.args.Page)
	if err != nil {
		return "", err
	}
	c.logger.Info().Str("page_id", pageID).Msgf("found page ID for title '%s' in space '%s'", c.args.Page, c.args.Space)
	return pageID, nil
}

func (c *Crawler) crawlConfluence(level int, outputDir string, pageId string, pageName string, mkDoc *mkdocs.MkDoc) error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	content, err := confluence.GetConfluencePage(c.ctx, c.api, pageId)
	if err != nil {
		return err
	}
	children := content.Children.Page
	outputDir, filename := pageOutputLocation(content, children, level, outputDir, pageName)
	c.logger.Info().Str("page_id", content.ID).Int("depth", level).Msgf("converting page %s", content.Title)
	if c.args.Debug {
		c.logger.Debug().Msgf("source for page %s:\n%s", content.Title, content.Body.Storage.Value)
	}
	childMkDoc, err := c.convertAndWritePage(content, outputDir, filename, mkDoc)
	if err != nil {
		return err
	}
	// Only page metadata is needed for attachments and recursive child exports.
	content.Body = goconfluence.Body{}

	err = c.downloadAttachments(outputDir, content)
	if err != nil {
		return fmt.Errorf("downloading attachments for page '%s': %w", content.Title, err)
	}

	if c.args.ExportChildren {
		err = c.downloadChildren(level, outputDir, childMkDoc, content)
		if err != nil {
			return fmt.Errorf("downloading child pages for page '%s': %w", content.Title, err)
		}
	}

	return nil
}

func pageOutputLocation(content *goconfluence.Content, children *goconfluence.Search, level int, outputDir, pageName string) (string, string) {
	if pageName == "" {
		pageName = storage.SanitizeFilename(content.Title)
	}
	if level != 0 && children != nil && len(children.Results) == 0 {
		return outputDir, pageName
	}
	if level != 0 {
		outputDir = filepath.Join(outputDir, pageName)
	}
	return outputDir, "index"
}

func uniqueChildPageNames(children []goconfluence.Results) []string {
	names := make([]string, len(children))
	counts := make(map[string]int, len(children))
	for _, child := range children {
		counts[storage.SanitizeFilename(child.Title)]++
	}

	used := make(map[string]struct{}, len(children))
	for i, child := range children {
		sanitizeFilename := storage.SanitizeFilename(child.Title)
		name := sanitizeFilename
		if counts[sanitizeFilename] > 1 {
			suffix := storage.SanitizeFilename(child.ID)
			if suffix == "untitled" {
				suffix = strconv.Itoa(i + 1)
			}
			name += "-" + suffix
		}

		candidate := name
		for suffix := 2; ; suffix++ {
			if _, exists := used[candidate]; !exists {
				break
			}
			candidate = name + "-" + strconv.Itoa(suffix)
		}
		names[i] = candidate
		used[candidate] = struct{}{}
	}
	return names
}

func (c *Crawler) convertAndWritePage(content *goconfluence.Content, outputDir string, filename string, mkDoc *mkdocs.MkDoc) (*mkdocs.MkDoc, error) {
	md, err := convertConfluenceToMarkdown(content.Body.Storage.Value)
	if err != nil {
		return nil, fmt.Errorf("converting Confluence page '%s' to Markdown: %w", content.Title, err)
	}

	outFile := filepath.Join(c.args.OutputDir, outputDir, filename+".md")
	c.logger.Debug().Msgf("adding page to MkDocs: %s with title: %s", outFile, content.Title)
	relativePath, err := filepath.Rel(c.args.OutputDir, outFile)
	if err != nil {
		return nil, fmt.Errorf("resolve MkDocs page path: %w", err)
	}
	childMkDoc := mkdocs.NewMkDocs(
		content.Title,
		filepath.ToSlash(relativePath),
	)
	mkDoc.AddChild(childMkDoc)
	if err := storage.WriteFile(outFile, md); err != nil {
		return nil, fmt.Errorf("writing Markdown file: %w", err)
	}
	c.logger.Info().Msgf("markdown file written to: %s", outFile)
	return childMkDoc, nil
}

func (c *Crawler) downloadAttachments(outputDir string, content *goconfluence.Content) error {
	c.logger.Info().Str("page_id", content.ID).Msgf("checking attachments for page: %s", content.Title)
	attachments, err := c.api.GetAttachments(c.ctx, content.ID)
	if err != nil {
		return fmt.Errorf("fetching attachments for page: %w", err)
	}
	for _, attachment := range attachments.Results {
		c.downloadAttachment(c.ctx, outputDir, attachment)
		// Individual attachment failures are logged, but cancellation must stop the export.
		if err := c.ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Crawler) downloadAttachment(ctx context.Context, outputDir string, attachment goconfluence.Results) {
	mediaType := ""
	if attachment.Metadata != nil {
		mediaType = attachment.Metadata.MediaType
	}
	c.logger.Info().Str("attachment_id", attachment.ID).Msgf("downloading attachment: '%s' type: %s", attachment.Title, mediaType)
	if attachment.Links == nil || attachment.Links.Download == "" {
		return
	}

	attachmentContent, err := c.api.GetAttachmentByUrl(ctx, attachment.Links.Download)
	if err != nil {
		c.logger.Error().Err(err).Msgf("error fetching attachment by URL: %s", attachment.Links.Download)
		return
	}
	filePath := filepath.Join(c.args.OutputDir, outputDir, "attachments", storage.SanitizeFilename(attachment.Title))
	if err := storage.WriteFile(filePath, *attachmentContent); err != nil {
		c.logger.Error().Err(err).Msgf("error writing attachment to file: %s", filePath)
		return
	}
	if mediaType == "application/gliffy+json" {
		c.convertGliffyAttachment(outputDir, attachment, *attachmentContent)
	}
}

func (c *Crawler) convertGliffyAttachment(outputDir string, attachment goconfluence.Results, content []byte) {
	c.logger.Info().Msgf("found Gliffy diagram: '%s'", attachment.Title)
	drawio, err := gliffy.Convert(content)
	if err != nil {
		c.logger.Error().Err(err).Msgf("error converting Gliffy diagram: %s", attachment.Title)
		return
	}
	filePath := filepath.Join(c.args.OutputDir, outputDir, "attachments", storage.SanitizeFilename(attachment.Title)+".drawio")
	if err := storage.WriteFile(filePath, drawio.XML); err != nil {
		c.logger.Error().Err(err).Msgf("error writing Draw.io file: %s", filePath)
	}
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
		err := c.crawlConfluence(level+1, outputDir, child.ID, childNames[i], mkDoc)
		if err != nil {
			return fmt.Errorf("crawling child page '%s': %w", child.Title, err)
		}
	}
	return nil
}
