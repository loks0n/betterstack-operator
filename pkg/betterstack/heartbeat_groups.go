package betterstack

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HeartbeatGroupClient defines heartbeat group operations provided by Better Stack.
type HeartbeatGroupClient interface {
	Create(ctx context.Context, req HeartbeatGroupCreateRequest) (HeartbeatGroup, error)
	Get(ctx context.Context, id string) (HeartbeatGroup, error)
	Update(ctx context.Context, id string, req HeartbeatGroupUpdateRequest) (HeartbeatGroup, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]HeartbeatGroup, error)
	ListHeartbeats(ctx context.Context, groupID string) ([]Heartbeat, error)
}

// HeartbeatGroupService provides heartbeat group operations for Better Stack.
type HeartbeatGroupService struct {
	client *Client
}

// HeartbeatGroup represents a Better Stack heartbeat group.
type HeartbeatGroup struct {
	ID         string                   `json:"id"`
	Attributes HeartbeatGroupAttributes `json:"attributes"`
}

// HeartbeatGroupAttributes describe the configuration of a heartbeat group.
type HeartbeatGroupAttributes struct {
	Name      string     `json:"name"`
	SortIndex *int       `json:"sort_index"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	TeamName  string     `json:"team_name"`
	Paused    bool       `json:"paused"`
}

// HeartbeatGroupRequest captures writable heartbeat group attributes for create and update operations.
type HeartbeatGroupRequest struct {
	TeamName  *string `json:"team_name,omitempty"`
	Paused    *bool   `json:"paused,omitempty"`
	Name      *string `json:"name,omitempty"`
	SortIndex *int    `json:"sort_index,omitempty"`
}

// HeartbeatGroupCreateRequest describes fields accepted when creating a heartbeat group.
type HeartbeatGroupCreateRequest = HeartbeatGroupRequest

// HeartbeatGroupUpdateRequest describes fields accepted when updating a heartbeat group.
type HeartbeatGroupUpdateRequest = HeartbeatGroupRequest

type heartbeatGroupEnvelope struct {
	Data heartbeatGroupData `json:"data"`
}

type heartbeatGroupData struct {
	ID         string                   `json:"id,omitempty"`
	Type       string                   `json:"type"`
	Attributes HeartbeatGroupAttributes `json:"attributes"`
}

type heartbeatGroupListEnvelope struct {
	Data       []heartbeatGroupData `json:"data"`
	Pagination struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"pagination"`
}

// Create creates a heartbeat group in Better Stack.
func (s *HeartbeatGroupService) Create(ctx context.Context, req HeartbeatGroupCreateRequest) (HeartbeatGroup, error) {
	var respEnvelope heartbeatGroupEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/heartbeat-groups", req, &respEnvelope); err != nil {
		return HeartbeatGroup{}, err
	}
	return HeartbeatGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Get retrieves a heartbeat group by ID.
func (s *HeartbeatGroupService) Get(ctx context.Context, id string) (HeartbeatGroup, error) {
	var respEnvelope heartbeatGroupEnvelope
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/heartbeat-groups/%s", url.PathEscape(id)), nil, &respEnvelope); err != nil {
		return HeartbeatGroup{}, err
	}
	return HeartbeatGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Update updates an existing heartbeat group in Better Stack.
func (s *HeartbeatGroupService) Update(ctx context.Context, id string, req HeartbeatGroupUpdateRequest) (HeartbeatGroup, error) {
	var respEnvelope heartbeatGroupEnvelope
	if err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/heartbeat-groups/%s", url.PathEscape(id)), req, &respEnvelope); err != nil {
		return HeartbeatGroup{}, err
	}
	if respEnvelope.Data.ID == "" {
		respEnvelope.Data.ID = id
	}
	return HeartbeatGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Delete removes a heartbeat group. Returns nil if the group is already absent.
func (s *HeartbeatGroupService) Delete(ctx context.Context, id string) error {
	err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/heartbeat-groups/%s", url.PathEscape(id)), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}

// List returns all heartbeat groups, following pagination automatically.
func (s *HeartbeatGroupService) List(ctx context.Context) ([]HeartbeatGroup, error) {
	path := "/heartbeat-groups"
	var groups []HeartbeatGroup

	for path != "" {
		var envelope heartbeatGroupListEnvelope
		if err := s.client.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
			return nil, err
		}

		for _, item := range envelope.Data {
			groups = append(groups, HeartbeatGroup{ID: item.ID, Attributes: item.Attributes})
		}

		next := strings.TrimSpace(envelope.Pagination.Next)
		if next == "" {
			break
		}
		next, _ = strings.CutPrefix(next, s.client.baseURL)
		path = next
	}

	return groups, nil
}

// ListHeartbeats returns all heartbeats belonging to a heartbeat group.
func (s *HeartbeatGroupService) ListHeartbeats(ctx context.Context, groupID string) ([]Heartbeat, error) {
	path := fmt.Sprintf("/heartbeat-groups/%s/heartbeats", url.PathEscape(groupID))
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
		next, _ = strings.CutPrefix(next, s.client.baseURL)
		path = next
	}

	return heartbeats, nil
}

var _ HeartbeatGroupClient = (*HeartbeatGroupService)(nil)
