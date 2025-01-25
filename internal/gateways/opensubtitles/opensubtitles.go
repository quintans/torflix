package opensubtitles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
)

const (
	AppVersion = "0.1"
	AppName    = "torflix"
	APIKey     = "KL70xnoAJTJMPaMeJr2qye4tzlDkYNsa" // Replace with your OpenSubtitles API key
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

// login authenticates and retrieves a Bearer token using username and password
func (o *OpenSubtitles) Login() (token string, err error) {
	err = retry.Do(func() error {
		token, err = o.login()
		return err
	}, retry.WithDelayFunc(https.DelayFunc))
	return
}

func (o *OpenSubtitles) login() (string, error) {
	url := fmt.Sprintf("%s/login", BaseURL)
	loginData := LoginRequest{
		Username: o.username,
		Password: o.password,
	}
	bodyJSON, _ := json.Marshal(loginData)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s", AppName, AppVersion))
	req.Header.Set("Api-Key", APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("posting login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", fails.New("too many requests for login", "retry-after", resp.Header.Get("Retry-After"))
		}
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to login, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return "", fmt.Errorf("decoding login response: %w", err)
	}

	return loginResp.Token, nil
}

// logout invalidates the access token
func (o *OpenSubtitles) Logout(token string) error {
	return retry.Do(func() error {
		return o.logout(token)
	}, retry.WithDelayFunc(https.DelayFunc))
}

func (o *OpenSubtitles) logout(token string) error {
	url := fmt.Sprintf("%s/logout", BaseURL)

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s", AppName, AppVersion))
	req.Header.Set("Api-Key", APIKey)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to logout, %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return fails.New("too many requests for logout", "retry-after", resp.Header.Get("Retry-After"))
		}
		return retry.NewPermanentError(fmt.Errorf("failed to logout, status code: %d", resp.StatusCode))
	}

	return nil
}

// Search searches for subtitles in specified languages for a given query
func (o *OpenSubtitles) Search(query string, season, episode int, languages []string) (attrs []app.SubtitleAttributes, err error) {
	err = retry.Do(func() error {
		attrs, err = o.search(query, season, episode, languages)
		return err
	}, retry.WithDelayFunc(https.DelayFunc))
	return
}

func (o *OpenSubtitles) search(query string, season, episode int, languages []string) ([]app.SubtitleAttributes, error) {
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
	req, _ := http.NewRequest("GET", url, nil)
	// req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s", AppName, AppVersion))
	req.Header.Set("Api-Key", APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting subtitle search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fails.New("too many requests for search", "retry-after", resp.Header.Get("Retry-After"))
		}
		return nil, retry.NewPermanentError(fmt.Errorf("failed to search subtitles, status code: %d", resp.StatusCode))
	}

	var searchResp SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&searchResp)
	if err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
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

// downloadSubtitle retrieves the download link for a given subtitle ID
func (o *OpenSubtitles) Download(token string, fileID int) (res app.DownloadResponse, err error) {
	err = retry.Do(func() error {
		res, err = o.download(token, fileID)
		return err
	}, retry.WithDelayFunc(https.DelayFunc))
	return
}

func (o *OpenSubtitles) download(token string, fileID int) (app.DownloadResponse, error) {
	url := fmt.Sprintf("%s/download", BaseURL)
	body := map[string]string{"file_id": strconv.Itoa(fileID)}
	bodyJSON, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s", AppName, AppVersion))
	req.Header.Set("Api-Key", APIKey)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return app.DownloadResponse{}, fmt.Errorf("requesting subtitle download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return app.DownloadResponse{}, fails.New("too many requests for download", "retry-after", resp.Header.Get("Retry-After"))
		}
		return app.DownloadResponse{}, retry.NewPermanentError(fmt.Errorf("downloading subtitle, status code: %d", resp.StatusCode))
	}

	var res app.DownloadResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return app.DownloadResponse{}, fmt.Errorf("decoding download response: %w", err)
	}

	return res, nil
}
