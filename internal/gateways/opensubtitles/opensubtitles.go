package opensubtitles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
)

func init() {
	// if APIKey was not set on build, try to get it from the environment
	if APIKey == "" {
		APIKey = os.Getenv("OS_API_KEY")
	}
}

var APIKey string

const (
	AppVersion = "0.1"
	AppName    = "torflix"
	BaseURL    = "https://api.opensubtitles.com/api/v1"
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
	username string
	password string
}

func New(username, password string) *OpenSubtitles {
	return &OpenSubtitles{
		username: username,
		password: password,
	}
}

// Login authenticates and retrieves a Bearer token using username and password
func (o *OpenSubtitles) Login() (string, error) {
	url := fmt.Sprintf("%s/login", BaseURL)
	loginData := LoginRequest{
		Username: o.username,
		Password: o.password,
	}

	var loginResp LoginResponse
	err := request(http.MethodPost, url, "", loginData, &loginResp)
	if err != nil {
		return "", fmt.Errorf("logging in: %w", err)
	}

	return loginResp.Token, nil
}

// Logout invalidates the access token
func (o *OpenSubtitles) Logout(token string) error {
	url := fmt.Sprintf("%s/logout", BaseURL)

	err := request(http.MethodDelete, url, token, nil, nil)
	if err != nil {
		return fmt.Errorf("logging out: %w", err)
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

	url := fmt.Sprintf("%s/subtitles?%s", BaseURL, sb.String())

	var searchResp SearchResponse
	err := request(http.MethodGet, url, "", nil, &searchResp)
	if err != nil {
		return nil, fmt.Errorf("searching subtitles: %w", err)
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
	url := fmt.Sprintf("%s/download", BaseURL)
	body := map[string]string{"file_id": strconv.Itoa(fileID)}

	var res app.DownloadResponse
	err := request(http.MethodPost, url, token, body, &res)
	if err != nil {
		return res, fmt.Errorf("downloading subtitle: %w", err)
	}

	return res, nil
}

func request(method, url, token string, request any, response any) error {
	return retry.Do(func() error {
		return retryRequest(method, url, token, request, response)
	}, retry.WithDelayFunc(https.DelayFunc))
}

func retryRequest(method, url, token string, request any, response any) error {
	var body io.Reader
	if request != nil {
		bodyJSON, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("marshalling request (%+v): %w", request, err)
		}
		body = bytes.NewBuffer(bodyJSON)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s", AppName, AppVersion))
	req.Header.Set("Api-Key", APIKey)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return fails.New("too many requests", "url", url, "retry-after", resp.Header.Get("Retry-After"))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return retry.NewPermanentError(fmt.Errorf("reading response body: %w", err))
		}
		return retry.NewPermanentError(fmt.Errorf("response status code %d for %s; response: %s", resp.StatusCode, url, string(body)))
	}

	if response != nil {
		err = json.NewDecoder(resp.Body).Decode(response)
		if err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
