package converter

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"golang.org/x/net/html"

	"conf2md/internal/gliffy"
	"conf2md/internal/goconfluence"
	"conf2md/internal/storage"
)

type gliffyAttachmentReference struct {
	pageID string
	name   string
}

func (c *Crawler) downloadAttachments(outputDir string, content *goconfluence.Content) error {
	c.logger.Info().Str("page_id", content.ID).Msgf("checking attachments for page: %s", content.Title)
	attachments, err := c.api.GetAttachments(c.ctx, content.ID)
	if err != nil {
		return fmt.Errorf("fetching attachments for page: %w", err)
	}
	for _, attachment := range attachments.Results {
		c.downloadAttachment(c.ctx, outputDir, attachment, content.ID)
		// Individual attachment failures are logged, but cancellation must stop the export.
		if err := c.ctx.Err(); err != nil {
			return err
		}
	}

	references, err := gliffyAttachmentReferences(content.Body.Storage.Value, content.ID)
	if err != nil {
		return fmt.Errorf("finding Gliffy attachment references: %w", err)
	}
	attachmentsByPage := make(map[string]*goconfluence.Search)
	for _, reference := range references {
		pageAttachments, found := attachmentsByPage[reference.pageID]
		if !found {
			pageAttachments, err = c.api.GetAttachments(c.ctx, reference.pageID)
			if err != nil {
				return fmt.Errorf("fetching attachments for referenced page %s: %w", reference.pageID, err)
			}
			attachmentsByPage[reference.pageID] = pageAttachments
		}

		foundAttachment := false
		for _, attachment := range pageAttachments.Results {
			if attachment.Title == reference.name || attachment.Title == reference.name+".png" {
				c.downloadAttachment(c.ctx, outputDir, attachment, reference.pageID)
				foundAttachment = true
				break
			}
		}
		if !foundAttachment {
			c.logger.Warn().
				Str("page_id", reference.pageID).
				Str("attachment_name", reference.name).
				Msg("referenced Gliffy attachment not found")
		}
	}
	return nil
}

func gliffyAttachmentReferences(storageValue, currentPageID string) ([]gliffyAttachmentReference, error) {
	doc, err := html.Parse(strings.NewReader(storageValue))
	if err != nil {
		return nil, err
	}

	var references []gliffyAttachmentReference
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "ac:structured-macro" &&
			dom.GetAttributeOr(node, "ac:name", "") == "gliffy" {
			pageID := strings.TrimSpace(getMacroParameterText(node, "pageid"))
			name := strings.TrimSpace(getMacroParameterText(node, "name"))
			if pageID != "" && pageID != currentPageID && name != "" {
				references = append(references, gliffyAttachmentReference{pageID: pageID, name: name})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	return references, nil
}

func getMacroParameterText(node *html.Node, parameterName string) string {
	parameter := dom.FindFirstNode(node, func(candidate *html.Node) bool {
		return dom.NodeName(candidate) == "ac:parameter" &&
			dom.GetAttributeOr(candidate, "ac:name", "") == parameterName
	})
	if parameter == nil {
		return ""
	}
	return dom.CollectText(parameter)
}

func (c *Crawler) downloadAttachment(ctx context.Context, outputDir string, attachment goconfluence.Results, pageID string) {
	mediaType := ""
	if attachment.Metadata != nil {
		mediaType = attachment.Metadata.MediaType
	}
	c.logger.Info().Str("attachment_id", attachment.ID).Msgf("downloading attachment: '%s' type: %s for page: %s", attachment.Title, mediaType, pageID)
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
