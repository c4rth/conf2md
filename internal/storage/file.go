package storage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	replacePattern = regexp.MustCompile(`[^a-zA-Z0-9.]`)
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

func WriteJSONFile(filePath string, value any) error {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return err
	}
	return WriteFile(filePath, data.Bytes())
}

func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
