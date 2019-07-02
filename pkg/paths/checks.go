package paths

import (
	"os"
	"github.com/apex/log"
)

// IsDir returns true if path is a directory, will fail otherwise
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatalf("unable to get os.Stat for %s", path)
	}

	return fi.IsDir()
}

// IsFile will return true if path is a file
func IsFile(path string) bool {
	return !IsDir(path)
}