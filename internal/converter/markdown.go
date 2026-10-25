package converter

import (
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"

	"conf2md/internal/config"
	"conf2md/internal/confluence"
	"conf2md/internal/confluence_plugin"
	"conf2md/internal/gliffy"
	"conf2md/internal/goconfluence"
	"conf2md/internal/mkdocs"
)

var plainTextBody = regexp.MustCompile(
	`(?s)<ac:plain-text-body>\s*<!\[CDATA\[(.*?)\]\]>\s*</ac:plain-text-body>`,
)

func sanitizeConfluenceHTML(input string) string {
	// Replace the CDATA wrappers with just their raw inner content
	return plainTextBody.ReplaceAllStringFunc(input, func(match string) string {
		content := plainTextBody.FindStringSubmatch(match)[1]
		return "<pre><code>" + stdhtml.EscapeString(content) + "</code></pre>"
	})
}

func convertConfluenceToMarkdownString(content string) (string, error) {
	cleanedContent := sanitizeConfluenceHTML(content)
	doc, err := html.Parse(strings.NewReader(cleanedContent))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %w", err)
	}
	conv := converter.NewConverter(
		converter.WithPlugins(
			confluence_plugin.NewConfluencePlugin(),
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(
				commonmark.WithEmDelimiter("_"),
				commonmark.WithStrongDelimiter("**"),
				commonmark.WithBulletListMarker("-"),
			),
			table.NewTablePlugin(),
		),
	)
	markdown, err := conv.ConvertNode(doc)
	if err != nil {
		return "", fmt.Errorf("error converting HTML to Markdown: %w", err)
	}
	return string(markdown), nil
}

func ConvertConfluenceToMarkdown(args *config.CmdLineArgs) error {
	start := time.Now()
	args.Page = extractPageId(args.Page)
	mkDoc := mkdocs.NewMkDocs("root", "root")
	err := error(nil)
	if args.Local {
		log.Info().Msgf("Converting local file: %s", args.Page)
		err = convertLocal(args, mkDoc)
	} else {
		err = convertConfluence(args, mkDoc)
	}
	if err == nil {
		if args.Mkdocs {
			mkDocsYaml := mkDoc.GenerateMkDocs()
			mkDocsFile := filepath.Join(args.OutputDir, "mkdocs.yml")
			os.WriteFile(mkDocsFile, []byte(mkDocsYaml), 0644)
			log.Info().Msgf("File 'mkdocs.yml' written to: %s", mkDocsFile)
		}
		end := time.Now()
		log.Info().Msgf("Conversion took: %s", end.Sub(start))
	}
	return err
}

func convertLocal(args *config.CmdLineArgs, mkDoc *mkdocs.MkDoc) error {
	fileFullPath := args.Page
	fileName := filepath.Base(fileFullPath)
	log.Info().Msgf("Reading local file: %s", fileFullPath)
	content, err := os.ReadFile(fileFullPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading local file: %s", fileFullPath)
		return fmt.Errorf("error opening file: %w", err)
	}
	// convert
	md, err := convertConfluenceToMarkdownString(string(content))
	if err != nil {
		log.Error().Err(err).Msgf("Error converting local file to Markdown: %s", fileFullPath)
		return err
	}
	// output directory and file path
	outDir := filepath.Join(args.OutputDir, "docs")
	os.MkdirAll(outDir, 0755)
	outFile := filepath.Join(outDir, fileName+".md")
	// mkdocs
	mkDoc.AddChild(mkdocs.NewMkDocs(
		filepath.Join("docs", filepath.Base(fileFullPath)),
		fileName,
	))
	// write output
	err = os.WriteFile(outFile, []byte(md), 0644)
	if err != nil {
		return fmt.Errorf("Error writing Markdown file: %w", err)
	}
	log.Info().Msgf("Markdown file written to: %s", outFile)
	return nil
}

func convertConfluence(args *config.CmdLineArgs, mkDoc *mkdocs.MkDoc) error {
	api, err := confluence.NewConfluenceAPI(args)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Confluence API")
		return err
	}
	outDir := filepath.Join(args.OutputDir, "docs")
	os.MkdirAll(outDir, 0755)
	err = crawlConfluence(api, args, 0, "docs", args.Page, mkDoc)
	if err != nil {
		log.Error().Err(err).Msg("Error crawling Confluence pages")
		return err
	}
	return nil
}

var replacePattern = regexp.MustCompile(`[^a-zA-Z0-9.]`)

func sanitizeFilename(filename string) string {
	// 1. Replace characters not matching [a-zA-Z0-9.] with '-'
	result := replacePattern.ReplaceAllString(filename, "-")

	// 2. Apply the string and regex cleanup steps
	result = strings.ReplaceAll(result, " - ", "-")
	result = strings.ReplaceAll(result, "---", "-")
	result = strings.ReplaceAll(result, "--", "-")

	// 3. Convert to lowercase
	return strings.ToLower(result)
}

func crawlConfluence(api *goconfluence.API, args *config.CmdLineArgs, level int, outputDir string, pageId string, mkDoc *mkdocs.MkDoc) error {
	content, err := confluence.GetConfluencePage(api, pageId)
	if err != nil {
		return err
	}
	children, err := confluence.GetChildPages(api, pageId)
	if err != nil {
		return err
	}
	sanitizedTitle := sanitizeFilename(content.Title)
	filename := ""
	if level != 0 && children != nil && len(children.Results) == 0 {
		filename = sanitizedTitle
	} else {
		if level != 0 {
			outputDir = filepath.Join("/", sanitizedTitle)
		}
		filename = "index"
	}
	log.Info().Msgf("%sConverting page %s", strings.Repeat("  ", level), content.Title)
	if args.Debug {
		log.Debug().Msgf("Source for page %s:\n%s", content.Title, content.Body.Storage.Value)
	}
	md, err := convertConfluenceToMarkdownString(content.Body.Storage.Value)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting Confluence page to Markdown: %s", content.Title)
		return err
	}

	subDir := filepath.Join(args.OutputDir, outputDir)
	os.MkdirAll(subDir, 0755)
	outFile := filepath.Join(subDir, filename+".md")
	// mkdocs
	log.Debug().Msgf("Adding page to MkDocs: %s with title: %s", outFile, content.Title)
	childMkDoc := mkdocs.NewMkDocs(
		filepath.Join("docs", filepath.Base(outFile)),
		content.Title,
	)
	mkDoc.AddChild(childMkDoc)
	// write output
	err = os.WriteFile(outFile, []byte(md), 0644)
	if err != nil {
		return fmt.Errorf("Error writing Markdown file: %w", err)
	}
	log.Info().Msgf("Markdown file written to: %s", outFile)

	err = downloadAttachments(api, args, level, outputDir, content)
	if err != nil {
		log.Error().Err(err).Msgf("Error downloading attachments for page: %s", content.Title)
		return err
	}

	if args.ExportChildren {
		err = downloadChildren(api, args, level, outputDir, childMkDoc, content)
		if err != nil {
			log.Error().Err(err).Msgf("Error downloading child pages for page: %s", content.Title)
			return err
		}
	}

	return nil
}

func downloadAttachments(api *goconfluence.API, args *config.CmdLineArgs, level int, outputDir string, content *goconfluence.Content) error {
	log.Info().Msgf("%sCheck attachments for page: %s", strings.Repeat("  ", level), content.Title)
	// TODO pagination ?
	attachments, err := api.GetAttachments(content.ID)
	if err != nil {
		return fmt.Errorf("Error fetching attachments for page: %w", err)
	}
	for _, attachment := range attachments.Results {
		log.Info().Msgf("%sDownload attachment: '%s' type: %s", strings.Repeat("  ", level), attachment.Title, attachment.Metadata.MediaType)
		if attachment.Links != nil && attachment.Links.Download != "" {
			attachmentContent, err := api.GetAttachmentByUrl(attachment.Links.Download)
			if err != nil {
				log.Error().Err(err).Msgf("Error fetching attachment by ID: %s", attachment.ID)
				continue
			}
			filePath := filepath.Join(args.OutputDir, outputDir, "attachments", attachment.Title)
			os.MkdirAll(filepath.Dir(filePath), 0755)
			err = os.WriteFile(filePath, *attachmentContent, 0644)
			if err != nil {
				log.Error().Err(err).Msgf("Error writing attachment to file: %s", filePath)
				continue
			}
			if attachment.Metadata.MediaType == "application/gliffy+json" {
				log.Info().Msgf("%sFound Gliffy diagram: '%s'", strings.Repeat("  ", level), attachment.Title)
				drawio, err := gliffy.Convert(*attachmentContent)
				if err != nil {
					log.Error().Err(err).Msgf("Error converting Gliffy diagram: %s", attachment.Title)
					continue
				}
				filePath := filepath.Join(args.OutputDir, outputDir, "attachments", attachment.Title+".drawio")
				os.MkdirAll(filepath.Dir(filePath), 0755)
				err = os.WriteFile(filePath, []byte(drawio.XML), 0644)
				if err != nil {
					log.Error().Err(err).Msgf("Error writing Draw.io file: %s", filePath)
					continue
				}
			}
		}
	}
	return nil
}

func downloadChildren(api *goconfluence.API, args *config.CmdLineArgs, level int, outputDir string, mkDoc *mkdocs.MkDoc, content *goconfluence.Content) error {
	log.Info().Msgf("%sCheck child pages for page: %s", strings.Repeat("  ", level), content.Title)
	// TODO pagination ?
	children, err := api.GetChildPages(content.ID)
	if err != nil {
		return fmt.Errorf("Error fetching child pages for page: %w", err)
	}
	for _, child := range children.Results {
		log.Info().Msgf("%sFound child page: %s", strings.Repeat("  ", level), child.Title)
		err = crawlConfluence(api, args, level+1, outputDir, child.ID, mkDoc)
		if err != nil {
			log.Error().Err(err).Msgf("Error crawling child page: %s", child.Title)
			return err
		}
	}
	return nil
}

func extractPageId(pageId string) string {
	// Implement the logic to extract the page ID from the given string
	return pageId
}

func playgroundDiagram() {

	const BaseDir = "./tests/gliffy/"

	//var files [3]string = [3]string{"test-gliffy-basic", "test-gliffy-cloud", "gliffy-diagram-application-flow"}
	//var files [1]string = [1]string{"test-gliffy-cloud"}
	var files []string = []string{"test_1.gliffy", "test_2.gliffy", "test_3.gliffy", "test_4.gliffy", "test_6.gliffy"}
	for _, file := range files {
		log.Info().Msgf("Processing file: %s", file)
		path := filepath.Join(BaseDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			log.Error().Err(err).Msgf("Error reading file: %s", path)
			continue
		}
		drawio, err := gliffy.Convert(data)
		if err != nil {
			log.Error().Err(err).Msgf("Error converting Gliffy file: %s", path)
			continue
		}

		for _, warn := range drawio.Warnings {
			log.Warn().Msgf("Warning converting Gliffy file: %s", warn)
		}

		outputPath := filepath.Join(BaseDir, file+"_converted.drawio")
		err = os.WriteFile(outputPath, []byte(drawio.XML), 0644)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing Draw.io file: %s", outputPath)
			continue
		}
		log.Info().Msgf("Converted diagram written to: %s", outputPath)

		log.Info().Msgf("------------------------------")
	}
}
