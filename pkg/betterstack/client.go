package betterstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://uptime.betterstack.com/api/v2"

// Client interacts with the Better Stack REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client

	Monitors      *MonitorService
	MonitorGroups *MonitorGroupService
	Heartbeats    *HeartbeatService
}

// APIError describes an error response from Better Stack.
type APIError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("better uptime api returned %d: %s", e.StatusCode, e.Message)
}

// NewClient creates a Better Stack API client.
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	client := &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		token:      token,
		httpClient: httpClient,
	}
	client.Monitors = &MonitorService{client: client}
	client.MonitorGroups = &MonitorGroupService{client: client}
	client.Heartbeats = &HeartbeatService{client: client}
	return client
}

// IsNotFound checks whether the provided error represents a 404 from Better Stack.
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

func (c *Client) do(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	if resp.StatusCode == http.StatusNoContent {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func parseAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)

	type apiErrorPayload struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	payload := apiErrorPayload{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &payload)
	}

	message := strings.TrimSpace(string(data))
	if len(payload.Errors) > 0 {
		parts := make([]string, 0, len(payload.Errors))
		for _, item := range payload.Errors {
			if item.Detail != "" {
				parts = append(parts, item.Detail)
				continue
			}
			if item.Title != "" {
				parts = append(parts, item.Title)
			}
		}
		if len(parts) > 0 {
			message = strings.Join(parts, "; ")
		}
	} else if payload.Error != "" {
		message = payload.Error
	} else if payload.Message != "" {
		message = payload.Message
	}

	if message == "" {
		message = resp.Status
	}

	return &APIError{StatusCode: resp.StatusCode, Message: message}
}
