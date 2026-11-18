package converter

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"conf2md/internal/config"
	"conf2md/internal/mkdocs"
	"conf2md/internal/storage"
)

func ConvertConfluenceToMarkdown(ctx context.Context, args *config.CmdLineArgs) error {
	start := time.Now()
	mkDoc := mkdocs.NewMkDocs("root", "root")
	err := error(nil)
	storage.RemoveDir(args.OutputDir, "docs") // ignore error for now
	if args.Local {
		log.Info().Msgf("converting local file: %s", args.Page)
		err = convertLocal(args, mkDoc)
	} else {
		err = convertConfluence(ctx, args, mkDoc)
	}
	if err != nil {
		return err
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
	outFile := filepath.Join(args.OutputDir, "docs", "index.md")
	// mkdocs
	mkDoc.AddChild(mkdocs.NewMkDocs(
		fileName,
		"index.md",
	))
	// write output
	err = storage.WriteFile(outFile, md)
	if err != nil {
		return fmt.Errorf("writing Markdown file: %w", err)
	}
	log.Info().Msgf("markdown file written to: %s", outFile)
	return nil
}
