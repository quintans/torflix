package extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/tidwall/gjson"
)

type apiConfig struct {
	Url         string          `json:"url"`
	QueryInPath bool            `json:"queryInPath"`
	List        string          `json:"list"`
	Result      json.RawMessage `json:"result"`
}

type Api struct {
	extractors map[string]apiConfig
}

func NewApi(cfg []byte) (*Api, error) {
	extractors := map[string]apiConfig{}
	err := json.Unmarshal(cfg, &extractors)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search config: %w", err)
	}

	return &Api{
		extractors: extractors,
	}, nil
}

func (s *Api) Accept(slug string) bool {
	_, ok := s.extractors[slug]
	return ok
}

func (s *Api) Slugs() []string {
	slugs := make([]string, 0, len(s.extractors))
	for k := range s.extractors {
		slugs = append(slugs, k)
	}
	return slugs
}

func (a *Api) Extract(slug string, query string) ([]Result, error) {
	xtr, ok := a.extractors[slug]
	if !ok {
		return nil, fmt.Errorf("no scraper found for %s", slug)
	}

	if xtr.QueryInPath {
		query = url.PathEscape(query)
	} else {
		query = url.QueryEscape(query)
	}

	u, err := replaceData(xtr.Url, map[string]string{"query": query})
	if err != nil {
		return nil, fmt.Errorf("failed to replace data: %w", err)
	}

	r, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to get API data for '%s': %w", slug, err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get API data for '%s', status code: %d", slug, r.StatusCode)
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API data for '%s': %w", slug, err)
	}

	res, err := transform(xtr, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to transform data: %w", err)
	}

	return res, nil
}

func transform(endpoint apiConfig, data string) ([]Result, error) {
	apiRes := apiFieldsQuery{}
	err := json.Unmarshal(endpoint.Result, &apiRes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	if endpoint.List != "" {
		res := gjson.Get(data, endpoint.List)
		data = res.String()
	}

	values := gjson.Parse(data).Array()
	res := make([]Result, len(values))
	for k, value := range values {
		v := apiFieldsResult{
			Name:   value.Get(apiRes.Name).String(),
			Seeds:  value.Get(apiRes.Seeds).String(),
			Magnet: value.Get(apiRes.Magnet).String(),
			Hash:   value.Get(apiRes.Hash).String(),
			SSize:  value.Get(apiRes.SSize).String(),
			HSize:  value.Get(apiRes.HSize).String(),
			Size:   value.Get(apiRes.Size).Uint(),
		}

		switch {
		case v.Size > 0:
			v.HSize = humanize.Bytes(v.Size)
		case v.SSize != "":
			s, err := humanize.ParseBytes(v.SSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse size: %w", err)
			}
			v.HSize = humanize.Bytes(s)
		}

		if v.Magnet == "" && v.Hash != "" {
			v.Magnet = fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", v.Hash, url.PathEscape(v.Name))
		}

		res[k] = Result{
			Name:   v.Name,
			Size:   v.HSize,
			Seeds:  v.Seeds,
			Magnet: v.Magnet,
		}
	}

	return res, nil
}

// using text/template to replace data
func replaceData(data string, replacements map[string]string) (string, error) {
	tmpl, err := template.New("data").Parse(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, replacements)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

type apiFieldsQuery struct {
	Name   string `json:"name"`
	Magnet string `json:"magnet"`
	Hash   string `json:"hash"`
	Seeds  string `json:"seeds"`
	HSize  string `json:"hsize"` // Human readable size. No formatting required.
	SSize  string `json:"ssize"` // Size as a String. Formatting required.
	Size   string `json:"size"`  // Size as a number. Formatting required.
}

type apiFieldsResult struct {
	Name   string
	Magnet string
	Hash   string
	Seeds  string
	HSize  string
	SSize  string
	Size   uint64
}
