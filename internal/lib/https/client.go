package https

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/retry"
)

type Client struct {
	BaseURL string
	Header  http.Header
}

func (c *Client) Get(uri string, response any, header http.Header) error {
	return c.Request(http.MethodGet, uri, nil, response, header)
}

func (c *Client) Post(uri string, request any, response any, header http.Header) error {
	return c.Request(http.MethodPost, uri, request, response, header)
}

func (c *Client) Put(uri string, request any, header http.Header) error {
	return c.Request(http.MethodPut, uri, request, nil, header)
}

func (c *Client) Delete(uri string, header http.Header) error {
	return c.Request(http.MethodDelete, uri, nil, nil, header)
}

func (c *Client) Request(method, uri string, request any, response any, header http.Header) error {
	var body io.Reader
	if request != nil {
		bodyJSON, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("marshalling request (%+v): %w", request, err)
		}
		body = bytes.NewBuffer(bodyJSON)
	}

	// clone header
	h := make(http.Header, len(c.Header))
	for k, v := range c.Header {
		h[k] = slices.Clone(v)
	}
	for k, v := range header {
		h[k] = slices.Clone(v)
	}

	url := c.BaseURL + uri
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header = h
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
