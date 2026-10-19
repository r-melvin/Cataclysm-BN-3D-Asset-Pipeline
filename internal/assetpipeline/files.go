package assetpipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func replaceExt(path string, extension string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + extension
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
