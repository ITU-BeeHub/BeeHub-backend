//go:build darwin

package documents

import (
	"log"
	"os"
	"path/filepath"
)

// getMacDocumentsDir returns the path to the Documents folder on macOS
func GetMacDocumentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	documentsDir := filepath.Join(home, "Documents")
	return documentsDir
}

// getDocumentsDir returns the Documents directory for macOS
func GetDocumentsDir() (string, error) {
	return GetMacDocumentsDir(), nil
}
