//go:build windows

package documents

import (
	"golang.org/x/sys/windows"
)

// getWindowsDocumentsDir returns the path to the Documents folder on Windows
func GetWindowsDocumentsDir() (string, error) {
	var rfid = windows.FOLDERID_Documents
	path, err := windows.KnownFolderPath(rfid, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", err
	}
	return path, nil
}

// getDocumentsDir returns the Documents directory for Windows
func GetDocumentsDir() (string, error) {
	return GetWindowsDocumentsDir()
}
