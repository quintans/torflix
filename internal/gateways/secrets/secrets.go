package secrets

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

type Secrets struct{}

func NewSecrets() *Secrets {
	return &Secrets{}
}

func (s *Secrets) GetOpenSubtitles() (string, error) {
	password, err := keyring.Get("torflix/opensubtitles", "password")
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("could not get OpenSubtitles password: %w", err)
	}

	return password, nil
}

func (s *Secrets) SetOpenSubtitles(value string) error {
	err := keyring.Set("torflix/opensubtitles", "password", value)
	if err != nil {
		return fmt.Errorf("could not save OpenSubtitles password: %w", err)
	}

	return nil
}
