package storage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	replacePattern    = regexp.MustCompile(`[^a-zA-Z0-9.]`)
	consecutiveDashes = regexp.MustCompile(`-+`)
)

func SanitizeFilename(filename string) string {
	result := replacePattern.ReplaceAllString(filename, "-")
	result = consecutiveDashes.ReplaceAllString(result, "-")
	result = strings.Trim(strings.ToLower(result), "-.")
	if result == "" {
		return "untitled"
	}
	return result
}

func WriteFile(filePath string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
