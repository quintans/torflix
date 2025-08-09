package opensubtitles

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
)

func init() {
	// if APIKey was not set on build, try to get it from the environment
	if apiKey == "" {
		apiKey = os.Getenv("OS_API_KEY")
	}
}

var apiKey string

const (
	BaseURL = "https://api.opensubtitles.com/api/v1"
)

type LoginResponse struct {
	Token string `json:"token"`
}

type SearchResponse struct {
	Data []struct {
		Attributes SubtitleAttributes `json:"attributes"`
	} `json:"data"`
}

type SubtitleAttributes struct {
	SubtitleID string `json:"subtitle_id"`
	Language   string `json:"language"`
	Files      []struct {
		FileID   int    `json:"file_id"`
		Filename string `json:"file_name"`
	} `json:"files"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type OpenSubtitles struct {
	client   https.Client
	username string
	password string
}

func New(username, password string) *OpenSubtitles {
	return &OpenSubtitles{
		client: https.Client{
			BaseURL: BaseURL,
			Header: http.Header{
				"Api-Key":      {apiKey},
				"User-Agent":   {fmt.Sprintf("%s v%s", app.Name, app.Version)},
				"Accept":       {"application/json"},
				"Content-Type": {"application/json"},
			},
		},
		username: username,
		password: password,
	}
}

// Login authenticates and retrieves a Bearer token using username and password
func (o *OpenSubtitles) Login() (string, error) {
	loginData := LoginRequest{
		Username: o.username,
		Password: o.password,
	}

	var loginResp LoginResponse
	err := o.request(http.MethodPost, "/login", "", loginData, &loginResp)
	if err != nil {
		return "", faults.Errorf("logging in: %w", err)
	}

	return loginResp.Token, nil
}

// Logout invalidates the access token
func (o *OpenSubtitles) Logout(token string) error {
	err := o.request(http.MethodDelete, "/logout", token, nil, nil)
	if err != nil {
		return faults.Errorf("logging out: %w", err)
	}

	return nil
}

// Search searches for subtitles in specified languages for a given query
func (o *OpenSubtitles) Search(query string, season, episode int, languages []string) ([]app.SubtitleAttributes, error) {
	type Params struct {
		Key, Value string
	}

	slices.Sort(slices.Clone(languages))

	query = strings.TrimSpace(query)
	query = strings.ToLower(query)
	params := []Params{
		{"query", url.QueryEscape(query)},
	}
	if len(languages) > 0 {
		params = append(params, Params{"languages", strings.Join(languages, ",")})
	}
	if episode > 0 {
		params = append(params, Params{"episode_number", strconv.Itoa(episode)})
	}
	if season > 0 {
		params = append(params, Params{"season_number", strconv.Itoa(season)})
	}

	if season == 0 && episode == 0 {
		params = append(params, Params{"type", "movie"})
	} else {
		params = append(params, Params{"type", "episode"})
	}

	slices.SortFunc(params, func(a, b Params) int {
		return strings.Compare(a.Key, b.Key)
	})

	sb := strings.Builder{}
	for k, p := range params {
		if k > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(p.Key)
		sb.WriteString("=")
		sb.WriteString(p.Value)
	}

	uri := fmt.Sprintf("/subtitles?%s", sb.String())

	var searchResp SearchResponse
	err := o.request(http.MethodGet, uri, "", nil, &searchResp)
	if err != nil {
		return nil, faults.Errorf("searching subtitles: %w", err)
	}

	var subtitles []app.SubtitleAttributes
	for _, data := range searchResp.Data {
		attr := data.Attributes
		var fileId int
		var filename string
		if len(attr.Files) > 0 {
			fileId = attr.Files[0].FileID
			filename = attr.Files[0].Filename
		}
		subtitles = append(subtitles, app.SubtitleAttributes{
			ID:       attr.SubtitleID,
			Language: attr.Language,
			FileID:   fileId,
			Filename: filename,
		})
	}

	return subtitles, nil
}

// Download retrieves the download link for a given subtitle ID
func (o *OpenSubtitles) Download(token string, fileID int) (app.DownloadResponse, error) {
	body := map[string]string{"file_id": strconv.Itoa(fileID)}

	var res app.DownloadResponse
	err := o.request(http.MethodPost, "/download", token, body, &res)
	if err != nil {
		return res, faults.Errorf("downloading subtitle: %w", err)
	}

	return res, nil
}

func (o *OpenSubtitles) request(method, url, token string, request any, response any) error {
	return retry.Do(func() error {
		return o.retryRequest(method, url, token, request, response)
	}, retry.WithDelayFunc(https.DelayFunc))
}

func (o *OpenSubtitles) retryRequest(method, uri, token string, request any, response any) error {
	var header http.Header
	if token != "" {
		header = http.Header{"Authorization": {"Bearer " + token}}
	}

	err := o.client.Request(method, uri, request, response, header)
	if err != nil {
		return faults.Errorf("requesting %s: %w", uri, err)
	}

	return nil
}
