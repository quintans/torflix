package files

import (
	"errors"
	"os"
	"path/filepath"
)

func Exists(path ...string) bool {
	file := filepath.Join(path...)
	_, err := os.Stat(file)
	return !errors.Is(err, os.ErrNotExist)
}
