package trakt

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/https"
)

func init() {
	// if vars were not set on build, try to get them from the environment
	if clientID == "" {
		clientID = os.Getenv("TRAKT_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("TRAKT_CLIENT_SECRET")
	}
}

var (
	clientID     string
	clientSecret string
)

type Trakt struct {
	client       https.Client
	accessToken  string
	expiresAt    time.Time
	refreshToken string
	onRefresh    func(NewArgs)
}

type NewArgs struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func New(args NewArgs, onRefresh func(NewArgs)) *Trakt {
	return &Trakt{
		client: https.Client{
			BaseURL: baseURL,
			Header: http.Header{
				"User-Agent":        {fmt.Sprintf("%s v%s", app.Name, app.Version)},
				"trakt-api-version": {"2"},
				"trakt-api-key":     {clientID},
				"Authorization":     {"Bearer " + args.AccessToken},
				"Accept":            {"application/json"},
				"Content-Type":      {"application/json"},
			},
		},
		accessToken:  args.AccessToken,
		expiresAt:    args.ExpiresAt,
		refreshToken: args.RefreshToken,
		onRefresh:    onRefresh,
	}
}

func (t *Trakt) RefreshToken() error {
	data := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": t.refreshToken,
		"grant_type":    "refresh_token",
	}

	var tr app.TokenResponse
	err := t.client.Post("/oauth/token", data, &tr, nil)
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}

	t.client.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	t.accessToken = tr.AccessToken
	t.refreshToken = tr.RefreshToken
	t.expiresAt = time.Unix(int64(tr.CreatedAt), 0).Add(time.Second * time.Duration(tr.ExpiresIn))

	if t.onRefresh != nil {
		t.onRefresh(NewArgs{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    t.expiresAt,
		})
	}

	return nil
}
