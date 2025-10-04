package betterstack

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HeartbeatClient defines the heartbeat operations provided by Better Stack.
type HeartbeatClient interface {
	Create(ctx context.Context, req HeartbeatCreateRequest) (Heartbeat, error)
	Get(ctx context.Context, id string) (Heartbeat, error)
	Update(ctx context.Context, id string, req HeartbeatUpdateRequest) (Heartbeat, error)
	Delete(ctx context.Context, id string) error
}

// HeartbeatService provides heartbeat-specific Better Stack operations.
type HeartbeatService struct {
	client *Client
}

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
	Pagination struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"pagination"`
}

// Create creates a heartbeat in Better Stack.
func (s *HeartbeatService) Create(ctx context.Context, req HeartbeatCreateRequest) (Heartbeat, error) {
	var respEnvelope heartbeatEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/heartbeats", req, &respEnvelope); err != nil {
		return Heartbeat{}, err
	}
	return Heartbeat{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
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

// List returns the collection of heartbeats. Pagination is followed automatically.
func (s *HeartbeatService) List(ctx context.Context) ([]Heartbeat, error) {
	path := "/heartbeats"
	var heartbeats []Heartbeat

	for path != "" {
		var envelope heartbeatListEnvelope
		if err := s.client.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
			return nil, err
		}

		for _, item := range envelope.Data {
			heartbeats = append(heartbeats, Heartbeat{ID: item.ID, Attributes: item.Attributes})
		}

		next := strings.TrimSpace(envelope.Pagination.Next)
		if next == "" {
			break
		}

		// normalise next path relative to base URL when required
		if strings.HasPrefix(next, s.client.baseURL) {
			next = strings.TrimPrefix(next, s.client.baseURL)
		}
		path = next
	}

	return heartbeats, nil
}

var _ HeartbeatClient = (*HeartbeatService)(nil)
