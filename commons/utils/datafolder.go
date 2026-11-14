package utils

import (
	"os"
	"path/filepath"
)

// GetDataFolderPath keeps the database, bot folders and attachments under one DATA_FOLDER so a single volume backs them all up.
func GetDataFolderPath(elements ...string) string {
	path := os.Getenv("DATA_FOLDER")
	if path == "" {
		path = "./data"
	}

	return filepath.Join(append([]string{path}, elements...)...)
}
