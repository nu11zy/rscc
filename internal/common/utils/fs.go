package utils

import "os"

// IsDir checks if supplied path is real directory
func IsDir(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return s.IsDir(), nil
}
