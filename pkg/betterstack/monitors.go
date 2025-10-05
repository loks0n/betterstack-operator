package betterstack

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MonitorClient defines the monitor operations provided by Better Stack.
type MonitorClient interface {
	Create(ctx context.Context, req MonitorCreateRequest) (Monitor, error)
	Get(ctx context.Context, id string) (Monitor, error)
	Update(ctx context.Context, id string, req MonitorUpdateRequest) (Monitor, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]Monitor, error)
}

// MonitorService provides monitor-specific Better Stack operations.
type MonitorService struct {
	client *Client
}

// Monitor represents a Better Stack monitor.
type Monitor struct {
	ID         string            `json:"id"`
	Attributes MonitorAttributes `json:"attributes"`
}

// MonitorAttributes describe the configuration and runtime state of a monitor.
type MonitorAttributes struct {
	URL               string `json:"url"`
	PronounceableName string `json:"pronounceable_name"`
	MonitorType       string `json:"monitor_type"`
	// MonitorGroupID is documented as a string, but the Better Stack API currently returns
	// either a quoted string or a bare number depending on the endpoint.
	// TODO(loks0n): drop the custom unmarshal once the API consistently emits strings.
	MonitorGroupID       *string           `json:"-"`
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

type monitorAttributesAlias MonitorAttributes

func (a *MonitorAttributes) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var groupID *string
	if v, ok := raw["monitor_group_id"]; ok {
		id, err := parseMonitorGroupID(v)
		if err != nil {
			return err
		}
		groupID = id
		delete(raw, "monitor_group_id")
	}

	processed, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	var alias monitorAttributesAlias
	if err := json.Unmarshal(processed, &alias); err != nil {
		return err
	}

	*a = MonitorAttributes(alias)
	a.MonitorGroupID = groupID
	return nil
}

func parseMonitorGroupID(raw json.RawMessage) (*string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 || trimmed == "null" {
		return nil, nil
	}

	if trimmed[0] == '"' {
		var str string
		if err := json.Unmarshal(raw, &str); err != nil {
			return nil, err
		}
		if str == "" {
			return nil, nil
		}
		return &str, nil
	}

	// Some monitor endpoints emit numeric IDs even though the docs specify strings,
	// so accept numbers and normalise them to canonical string form.
	// TODO(loks0n): tighten this once Better Stack fixes their API.
	var num json.Number
	if err := json.Unmarshal(raw, &num); err != nil {
		return nil, err
	}
	str := num.String()
	if str == "" {
		return nil, nil
	}
	return &str, nil
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
	maps.Copy(payload, r.AdditionalAttributes)
	return json.Marshal(payload)
}

// MonitorCreateRequest describes fields accepted when creating a monitor.
type MonitorCreateRequest = MonitorRequest

// MonitorUpdateRequest describes fields accepted when updating a monitor.
type MonitorUpdateRequest = MonitorRequest

type monitorEnvelope struct {
	Data monitorData `json:"data"`
}

type monitorData struct {
	ID         string            `json:"id,omitempty"`
	Type       string            `json:"type"`
	Attributes MonitorAttributes `json:"attributes"`
}

type monitorListEnvelope struct {
	Data       []monitorData `json:"data"`
	Pagination struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"pagination"`
}

// Create creates a monitor in Better Stack.
func (s *MonitorService) Create(ctx context.Context, req MonitorCreateRequest) (Monitor, error) {
	var respEnvelope monitorEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/monitors", req, &respEnvelope); err != nil {
		return Monitor{}, err
	}
	return Monitor{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
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

// List returns all monitors, following pagination automatically.
func (s *MonitorService) List(ctx context.Context) ([]Monitor, error) {
	path := "/monitors"
	var monitors []Monitor

	for path != "" {
		var envelope monitorListEnvelope
		if err := s.client.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
			return nil, err
		}

		for _, item := range envelope.Data {
			monitors = append(monitors, Monitor{ID: item.ID, Attributes: item.Attributes})
		}

		next := strings.TrimSpace(envelope.Pagination.Next)
		if next == "" {
			break
		}
		next, _ = strings.CutPrefix(next, s.client.baseURL)
		path = next
	}

	return monitors, nil
}

var _ MonitorClient = (*MonitorService)(nil)
