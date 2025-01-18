package files

import (
	"errors"
	"os"
)

func Exists(file string) bool {
	_, err := os.Stat(file)
	return !errors.Is(err, os.ErrNotExist)
}
