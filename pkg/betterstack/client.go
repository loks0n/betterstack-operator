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

	Monitors   *MonitorService
	Heartbeats *HeartbeatService
}

// MonitorService provides monitor-specific Better Stack operations.
type MonitorService struct {
	client *Client
}

// MonitorRequestHeader describes an HTTP header to include with a monitor check.
type MonitorRequestHeader struct {
	ID      *string `json:"id,omitempty"`
	Name    string  `json:"name,omitempty"`
	Value   string  `json:"value,omitempty"`
	Destroy *bool   `json:"_destroy,omitempty"`
}

// MonitorRequest captures the writable attributes for monitor operations.
type MonitorRequest struct {
	TeamName             *string                `json:"team_name,omitempty"`
	MonitorType          *string                `json:"monitor_type,omitempty"`
	URL                  *string                `json:"url,omitempty"`
	PronounceableName    *string                `json:"pronounceable_name,omitempty"`
	Email                *bool                  `json:"email,omitempty"`
	SMS                  *bool                  `json:"sms,omitempty"`
	Call                 *bool                  `json:"call,omitempty"`
	Push                 *bool                  `json:"push,omitempty"`
	CriticalAlert        *bool                  `json:"critical_alert,omitempty"`
	CheckFrequency       *int                   `json:"check_frequency,omitempty"`
	RequestHeaders       []MonitorRequestHeader `json:"request_headers,omitempty"`
	ExpectedStatusCodes  []int                  `json:"expected_status_codes,omitempty"`
	DomainExpiration     *int                   `json:"domain_expiration,omitempty"`
	SSLExpiration        *int                   `json:"ssl_expiration,omitempty"`
	PolicyID             *string                `json:"policy_id,omitempty"`
	ExpirationPolicyID   *string                `json:"expiration_policy_id,omitempty"`
	FollowRedirects      *bool                  `json:"follow_redirects,omitempty"`
	RequiredKeyword      *string                `json:"required_keyword,omitempty"`
	TeamWait             *int                   `json:"team_wait,omitempty"`
	Paused               *bool                  `json:"paused,omitempty"`
	Port                 *string                `json:"port,omitempty"`
	Regions              []string               `json:"regions,omitempty"`
	MonitorGroupID       *string                `json:"monitor_group_id,omitempty"`
	RecoveryPeriod       *int                   `json:"recovery_period,omitempty"`
	VerifySSL            *bool                  `json:"verify_ssl,omitempty"`
	ConfirmationPeriod   *int                   `json:"confirmation_period,omitempty"`
	HTTPMethod           *string                `json:"http_method,omitempty"`
	RequestTimeout       *int                   `json:"request_timeout,omitempty"`
	RequestBody          *string                `json:"request_body,omitempty"`
	AuthUsername         *string                `json:"auth_username,omitempty"`
	AuthPassword         *string                `json:"auth_password,omitempty"`
	MaintenanceDays      []string               `json:"maintenance_days,omitempty"`
	MaintenanceFrom      *string                `json:"maintenance_from,omitempty"`
	MaintenanceTo        *string                `json:"maintenance_to,omitempty"`
	MaintenanceTimezone  *string                `json:"maintenance_timezone,omitempty"`
	RememberCookies      *bool                  `json:"remember_cookies,omitempty"`
	PlaywrightScript     *string                `json:"playwright_script,omitempty"`
	ScenarioName         *string                `json:"scenario_name,omitempty"`
	EnvironmentVariables map[string]string      `json:"environment_variables,omitempty"`
	IPVersion            *string                `json:"ip_version,omitempty"`
	AdditionalAttributes map[string]any         `json:"-"`
}

// MarshalJSON ensures additional attributes are merged into the serialized payload.
func (r MonitorRequest) MarshalJSON() ([]byte, error) {
	type alias MonitorRequest
	data, err := json.Marshal(alias(r))
	if err != nil {
		return nil, err
	}
	if len(r.AdditionalAttributes) == 0 {
		return data, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	for k, v := range r.AdditionalAttributes {
		payload[k] = v
	}
	return json.Marshal(payload)
}

// MonitorCreateRequest describes fields accepted when creating a monitor.
type MonitorCreateRequest = MonitorRequest

// MonitorUpdateRequest describes fields accepted when updating a monitor.
type MonitorUpdateRequest = MonitorRequest

// HeartbeatService provides heartbeat-specific Better Stack operations.
type HeartbeatService struct {
	client *Client
}

// Monitor represents a Better Stack monitor.
type Monitor struct {
	ID         string            `json:"id"`
	Attributes MonitorAttributes `json:"attributes"`
}

// MonitorAttributes describe the configuration and runtime state of a monitor.
type MonitorAttributes struct {
	URL                  string            `json:"url"`
	PronounceableName    string            `json:"pronounceable_name"`
	MonitorType          string            `json:"monitor_type"`
	MonitorGroupID       *string           `json:"monitor_group_id"`
	LastCheckedAt        *time.Time        `json:"last_checked_at"`
	Status               MonitorStatus     `json:"status"`
	PolicyID             *int              `json:"policy_id"`
	ExpirationPolicyID   *int              `json:"expiration_policy_id"`
	TeamName             string            `json:"team_name"`
	RequiredKeyword      string            `json:"required_keyword"`
	VerifySSL            bool              `json:"verify_ssl"`
	CheckFrequency       int               `json:"check_frequency"`
	FollowRedirects      bool              `json:"follow_redirects"`
	RememberCookies      bool              `json:"remember_cookies"`
	Call                 bool              `json:"call"`
	SMS                  bool              `json:"sms"`
	Email                bool              `json:"email"`
	Push                 bool              `json:"push"`
	CriticalAlert        bool              `json:"critical_alert"`
	Paused               bool              `json:"paused"`
	TeamWait             *int              `json:"team_wait"`
	HTTPMethod           string            `json:"http_method"`
	RequestTimeout       int               `json:"request_timeout"`
	RecoveryPeriod       int               `json:"recovery_period"`
	RequestHeaders       []MonitorHeader   `json:"request_headers"`
	RequestBody          string            `json:"request_body"`
	PausedAt             *time.Time        `json:"paused_at"`
	CreatedAt            *time.Time        `json:"created_at"`
	UpdatedAt            *time.Time        `json:"updated_at"`
	SSLExpiration        *int              `json:"ssl_expiration"`
	DomainExpiration     *int              `json:"domain_expiration"`
	Regions              []string          `json:"regions"`
	Port                 *string           `json:"port"`
	ConfirmationPeriod   int               `json:"confirmation_period"`
	ExpectedStatusCodes  []int             `json:"expected_status_codes"`
	MaintenanceDays      []string          `json:"maintenance_days"`
	MaintenanceFrom      string            `json:"maintenance_from"`
	MaintenanceTo        string            `json:"maintenance_to"`
	MaintenanceTimezone  string            `json:"maintenance_timezone"`
	PlaywrightScript     string            `json:"playwright_script"`
	EnvironmentVariables map[string]string `json:"environment_variables"`
	IPVersion            *string           `json:"ip_version"`
}

// MonitorHeader represents headers returned by the API.
type MonitorHeader struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Value   string `json:"value"`
	Destroy bool   `json:"_destroy"`
}

// MonitorStatus enumerates monitor states.
type MonitorStatus string

const (
	MonitorStatusUp          MonitorStatus = "up"
	MonitorStatusDown        MonitorStatus = "down"
	MonitorStatusValidating  MonitorStatus = "validating"
	MonitorStatusPaused      MonitorStatus = "paused"
	MonitorStatusPending     MonitorStatus = "pending"
	MonitorStatusMaintenance MonitorStatus = "maintenance"
)

// Heartbeat represents a Better Stack heartbeat.
type Heartbeat struct {
	ID         string              `json:"id"`
	Attributes HeartbeatAttributes `json:"attributes"`
}

// HeartbeatAttributes describe the configuration and runtime state of a heartbeat.
type HeartbeatAttributes struct {
	URL                 string          `json:"url"`
	Name                string          `json:"name"`
	Period              int             `json:"period"`
	Grace               int             `json:"grace"`
	Call                bool            `json:"call"`
	SMS                 bool            `json:"sms"`
	Email               bool            `json:"email"`
	Push                bool            `json:"push"`
	CriticalAlert       bool            `json:"critical_alert"`
	TeamWait            *int            `json:"team_wait"`
	HeartbeatGroupID    *int            `json:"heartbeat_group_id"`
	TeamName            string          `json:"team_name"`
	SortIndex           *int            `json:"sort_index"`
	PausedAt            *time.Time      `json:"paused_at"`
	CreatedAt           *time.Time      `json:"created_at"`
	UpdatedAt           *time.Time      `json:"updated_at"`
	Status              HeartbeatStatus `json:"status"`
	MaintenanceDays     []string        `json:"maintenance_days"`
	MaintenanceFrom     string          `json:"maintenance_from"`
	MaintenanceTo       string          `json:"maintenance_to"`
	MaintenanceTimezone string          `json:"maintenance_timezone"`
}

// HeartbeatStatus enumerates known heartbeat states.
type HeartbeatStatus string

const (
	HeartbeatStatusPaused  HeartbeatStatus = "paused"
	HeartbeatStatusPending HeartbeatStatus = "pending"
	HeartbeatStatusUp      HeartbeatStatus = "up"
	HeartbeatStatusDown    HeartbeatStatus = "down"
)

// HeartbeatCreateRequest describes fields accepted when creating a heartbeat.
type HeartbeatCreateRequest struct {
	TeamName            *string  `json:"team_name,omitempty"`
	Name                *string  `json:"name,omitempty"`
	Period              *int     `json:"period,omitempty"`
	Grace               *int     `json:"grace,omitempty"`
	Call                *bool    `json:"call,omitempty"`
	SMS                 *bool    `json:"sms,omitempty"`
	Email               *bool    `json:"email,omitempty"`
	Push                *bool    `json:"push,omitempty"`
	CriticalAlert       *bool    `json:"critical_alert,omitempty"`
	TeamWait            *int     `json:"team_wait,omitempty"`
	HeartbeatGroupID    *int     `json:"heartbeat_group_id,omitempty"`
	SortIndex           *int     `json:"sort_index,omitempty"`
	Paused              *bool    `json:"paused,omitempty"`
	MaintenanceDays     []string `json:"maintenance_days,omitempty"`
	MaintenanceFrom     *string  `json:"maintenance_from,omitempty"`
	MaintenanceTo       *string  `json:"maintenance_to,omitempty"`
	MaintenanceTimezone *string  `json:"maintenance_timezone,omitempty"`
	PolicyID            *string  `json:"policy_id,omitempty"`
}

// HeartbeatUpdateRequest describes fields accepted when updating a heartbeat. Partial updates are supported.
type HeartbeatUpdateRequest HeartbeatCreateRequest

type monitorEnvelope struct {
	Data monitorData `json:"data"`
}

type monitorData struct {
	ID         string            `json:"id,omitempty"`
	Type       string            `json:"type"`
	Attributes MonitorAttributes `json:"attributes"`
}

type monitorListEnvelope struct {
	Data       []monitorData  `json:"data"`
	Pagination paginationInfo `json:"pagination"`
}

type heartbeatEnvelope struct {
	Data heartbeatData `json:"data"`
}

type heartbeatData struct {
	ID         string              `json:"id,omitempty"`
	Type       string              `json:"type"`
	Attributes HeartbeatAttributes `json:"attributes"`
}

type heartbeatListEnvelope struct {
	Data       []heartbeatData `json:"data"`
	Pagination paginationInfo  `json:"pagination"`
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
	client.Heartbeats = &HeartbeatService{client: client}
	return client
}

// Create creates a monitor in Better Stack.
func (s *MonitorService) Create(ctx context.Context, req MonitorCreateRequest) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/monitors", req, &respEnvelope); err != nil {
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
func (s *MonitorService) Update(ctx context.Context, id string, req MonitorUpdateRequest) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/monitors/%s", url.PathEscape(id)), req, &respEnvelope); err != nil {
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

// Create creates a heartbeat in Better Stack.
func (s *HeartbeatService) Create(ctx context.Context, req HeartbeatCreateRequest) (Heartbeat, error) {
	var respEnvelope heartbeatEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/heartbeats", req, &respEnvelope); err != nil {
		return Heartbeat{}, err
	}
	return Heartbeat{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// List retrieves heartbeats, optionally paginating the results.
func (s *HeartbeatService) List(ctx context.Context, opts ListHeartbeatsOptions) (HeartbeatList, error) {
	values := url.Values{}
	if opts.Page > 0 {
		values.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		values.Set("per_page", strconv.Itoa(opts.PerPage))
	}

	path := "/heartbeats"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	var resp heartbeatListEnvelope
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return HeartbeatList{}, err
	}

	heartbeats := make([]Heartbeat, 0, len(resp.Data))
	for _, item := range resp.Data {
		heartbeats = append(heartbeats, Heartbeat{ID: item.ID, Attributes: item.Attributes})
	}

	return HeartbeatList{
		Heartbeats: heartbeats,
		Pagination: Pagination(resp.Pagination),
	}, nil
}

// Get retrieves a heartbeat by ID.
func (s *HeartbeatService) Get(ctx context.Context, id string) (Heartbeat, error) {
	var respEnvelope heartbeatEnvelope
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/heartbeats/%s", url.PathEscape(id)), nil, &respEnvelope); err != nil {
		return Heartbeat{}, err
	}
	return Heartbeat{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Update updates an existing heartbeat in Better Stack.
func (s *HeartbeatService) Update(ctx context.Context, id string, req HeartbeatUpdateRequest) (Heartbeat, error) {
	var respEnvelope heartbeatEnvelope
	if err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/heartbeats/%s", url.PathEscape(id)), req, &respEnvelope); err != nil {
		return Heartbeat{}, err
	}
	if respEnvelope.Data.ID == "" {
		respEnvelope.Data.ID = id
	}
	return Heartbeat{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Delete removes a heartbeat. Returns nil if the heartbeat is already absent.
func (s *HeartbeatService) Delete(ctx context.Context, id string) error {
	err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/heartbeats/%s", url.PathEscape(id)), nil, nil)
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

// ListHeartbeatsOptions controls pagination for HeartbeatService.List.
type ListHeartbeatsOptions struct {
	Page    int
	PerPage int
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

// HeartbeatList contains heartbeats with pagination metadata.
type HeartbeatList struct {
	Heartbeats []Heartbeat
	Pagination Pagination
}
