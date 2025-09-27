package betterstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://uptime.betterstack.com/api/v2"

// Client interacts with the Better Stack REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client

	Monitors *MonitorService
}

// MonitorService provides monitor-specific Better Stack operations.
type MonitorService struct {
	client *Client
}

// Monitor represents a Better Stack monitor.
type Monitor struct {
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

type monitorEnvelope struct {
	Data monitorData `json:"data"`
}

type monitorData struct {
	ID         string         `json:"id,omitempty"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

type monitorListEnvelope struct {
	Data       []monitorData  `json:"data"`
	Pagination paginationInfo `json:"pagination"`
}

type paginationInfo struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
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
	return client
}

// Create creates a monitor in Better Stack.
func (s *MonitorService) Create(ctx context.Context, attrs map[string]any) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/monitors", attrs, &respEnvelope); err != nil {
		return Monitor{}, err
	}
	return Monitor{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// List retrieves monitors, optionally filtering and paginating results.
func (s *MonitorService) List(ctx context.Context, opts ListMonitorsOptions) (MonitorList, error) {
	values := url.Values{}
	if opts.URL != "" {
		values.Set("url", opts.URL)
	}
	if opts.PronounceableName != "" {
		values.Set("pronounceable_name", opts.PronounceableName)
	}
	if opts.Page > 0 {
		values.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		values.Set("per_page", strconv.Itoa(opts.PerPage))
	}

	path := "/monitors"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	var resp monitorListEnvelope
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return MonitorList{}, err
	}

	monitors := make([]Monitor, 0, len(resp.Data))
	for _, item := range resp.Data {
		monitors = append(monitors, Monitor{ID: item.ID, Attributes: item.Attributes})
	}

	return MonitorList{
		Monitors:   monitors,
		Pagination: Pagination(resp.Pagination),
	}, nil
}

// Get retrieves a monitor by ID.
func (s *MonitorService) Get(ctx context.Context, id string) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/monitors/%s", url.PathEscape(id)), nil, &respEnvelope); err != nil {
		return Monitor{}, err
	}
	return Monitor{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Update updates an existing monitor in Better Stack.
func (s *MonitorService) Update(ctx context.Context, id string, attrs map[string]any) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodPut, fmt.Sprintf("/monitors/%s", url.PathEscape(id)), attrs, &respEnvelope); err != nil {
		return Monitor{}, err
	}
	if respEnvelope.Data.ID == "" {
		respEnvelope.Data.ID = id
	}
	return Monitor{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Delete removes a monitor. Returns nil if the monitor is already absent.
func (s *MonitorService) Delete(ctx context.Context, id string) error {
	err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/monitors/%s", url.PathEscape(id)), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
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

// ListMonitorsOptions controls filtering and pagination for MonitorService.List.
type ListMonitorsOptions struct {
	URL               string
	PronounceableName string
	Page              int
	PerPage           int
}

// Pagination describes paginated link relations returned by the API.
type Pagination struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

// MonitorList contains monitors with pagination metadata.
type MonitorList struct {
	Monitors   []Monitor
	Pagination Pagination
}
