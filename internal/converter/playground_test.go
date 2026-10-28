//go:build playground

package converter

import (
	"path/filepath"
	"testing"

	"conf2md/internal/gliffy"
	"conf2md/internal/storage"
)

func TestPlaygroundDiagram(t *testing.T) {
	baseDir := filepath.Join("..", "..", "tests", "gliffy")
	files := []string{"test_1.gliffy", "test_2.gliffy", "test_3.gliffy", "test_4.gliffy", "test_6.gliffy"}
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
			outputPath := filepath.Join(baseDir, file+"_converted.drawio")
			if err := storage.WriteFile(outputPath, drawio.XML); err != nil {
				t.Fatalf("write Draw.io file %s: %v", outputPath, err)
			}
			t.Logf("Converted diagram written to: %s", outputPath)
		})
	}
}
