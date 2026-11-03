//go:build playground
package converter

import (
	"os"
	"path/filepath"
	"testing"

	"conf2md/internal/gliffy"
	"conf2md/internal/storage"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestPlaygroundDiagram(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)
	baseDir := filepath.Join("..", "..", "tests", "gliffy")
	files := []string{
		"test_1.gliffy",
		"test_2.gliffy",
		"test_3.gliffy",
		"test_4.gliffy",
		"test_6.gliffy",
		"test_7.gliffy",
		"test_8.gliffy",
		"test_9.gliffy",
		"test_10.gliffy",
		"test_11.gliffy",
		"test_v2_1.gliffy",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(baseDir, file)
			data, err := storage.ReadFile(path)
			if err != nil {
				t.Fatalf("read Gliffy file %s: %v", path, err)
			}
			drawio, err := gliffy.Convert(data)
			if err != nil {
				t.Fatalf("convert Gliffy file %s: %v", path, err)
			}
			for _, warning := range drawio.Warnings {
				t.Logf("conversion warning: %s", warning)
			}
			outputPath := filepath.Join(baseDir, "drawio", file+"_converted.drawio")
			if err := storage.WriteFile(outputPath, drawio.XML); err != nil {
				t.Fatalf("write Draw.io file %s: %v", outputPath, err)
			}
			t.Logf("Converted diagram written to: %s", outputPath)
		})
	}
}
