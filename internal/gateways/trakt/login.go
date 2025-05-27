package trakt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/quintans/torflix/internal/app"
)

const (
	baseURL       = "https://api.trakt.tv"
	deviceCodeURL = baseURL + "/oauth/device/code"
	tokenURL      = baseURL + "/oauth/device/token"
)

type Auth struct{}

func (Auth) GetDeviceCode() (app.DeviceCodeResponse, error) {
	data := map[string]string{
		"client_id": clientID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return app.DeviceCodeResponse{}, fmt.Errorf("marshalling data: %w", err)
	}

	resp, err := http.Post(deviceCodeURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return app.DeviceCodeResponse{}, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var deviceCodeResponse app.DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCodeResponse); err != nil {
		return app.DeviceCodeResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return deviceCodeResponse, nil
}

func (Auth) PollForToken(deviceCodeResponse app.DeviceCodeResponse) (app.TokenResponse, error) {
	data := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          deviceCodeResponse.DeviceCode,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return app.TokenResponse{}, fmt.Errorf("marshalling data for token poll: %w", err)
	}

	for {
		res, err := getToken(jsonData)
		if err != nil {
			return app.TokenResponse{}, fmt.Errorf("getting token: %w", err)
		}
		if res != (app.TokenResponse{}) {
			return res, nil
		}

		time.Sleep(time.Duration(deviceCodeResponse.Interval) * time.Second)
	}
}

func getToken(jsonData []byte) (app.TokenResponse, error) {
	resp, err := http.Post(tokenURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return app.TokenResponse{}, fmt.Errorf("sending request for token poll: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return app.TokenResponse{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return app.TokenResponse{}, fmt.Errorf("response status code %d for token poll", resp.StatusCode)
	}

	var tokenResponse app.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return app.TokenResponse{}, fmt.Errorf("decoding response for token poll: %w", err)
	}
	return tokenResponse, nil
}
