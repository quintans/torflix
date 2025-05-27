package secrets

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/quintans/torflix/internal/app"
	"github.com/zalando/go-keyring"
)

type Secrets struct{}

func NewSecrets() *Secrets {
	return &Secrets{}
}

func (s *Secrets) GetOpenSubtitles() (app.OpenSubtitlesSecret, error) {
	data, err := keyring.Get("torflix", "opensubtitles")
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return app.OpenSubtitlesSecret{}, nil
		}
		return app.OpenSubtitlesSecret{}, fmt.Errorf("could not get OpenSubtitles: %w", err)
	}

	var secret app.OpenSubtitlesSecret
	err = json.Unmarshal([]byte(data), &secret)
	if err != nil {
		return app.OpenSubtitlesSecret{}, fmt.Errorf("could not unmarshal OpenSubtitles: %w", err)
	}

	return secret, nil
}

func (s *Secrets) SetOpenSubtitles(value app.OpenSubtitlesSecret) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal OpenSubtitles: %w", err)
	}
	err = keyring.Set("torflix", "opensubtitles", string(data))
	if err != nil {
		return fmt.Errorf("could not save OpenSubtitles: %w", err)
	}

	return nil
}
