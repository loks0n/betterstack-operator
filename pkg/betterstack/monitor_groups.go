package betterstack

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MonitorGroupClient defines the monitor group operations provided by Better Stack.
type MonitorGroupClient interface {
	Create(ctx context.Context, req MonitorGroupCreateRequest) (MonitorGroup, error)
	Get(ctx context.Context, id string) (MonitorGroup, error)
	Update(ctx context.Context, id string, req MonitorGroupUpdateRequest) (MonitorGroup, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]MonitorGroup, error)
	ListMonitors(ctx context.Context, groupID string) ([]Monitor, error)
}

// MonitorGroupService provides monitor group operations for Better Stack.
type MonitorGroupService struct {
	client *Client
}

// MonitorGroup represents a Better Stack monitor group.
type MonitorGroup struct {
	ID         string                 `json:"id"`
	Attributes MonitorGroupAttributes `json:"attributes"`
}

// MonitorGroupAttributes describe the configuration of a monitor group.
type MonitorGroupAttributes struct {
	Name      string     `json:"name"`
	SortIndex *int       `json:"sort_index"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	TeamName  string     `json:"team_name"`
	Paused    bool       `json:"paused"`
}

// MonitorGroupRequest captures writable monitor group attributes for create and update operations.
type MonitorGroupRequest struct {
	TeamName  *string `json:"team_name,omitempty"`
	Paused    *bool   `json:"paused,omitempty"`
	Name      *string `json:"name,omitempty"`
	SortIndex *int    `json:"sort_index,omitempty"`
}

// MonitorGroupCreateRequest describes fields accepted when creating a monitor group.
type MonitorGroupCreateRequest = MonitorGroupRequest

// MonitorGroupUpdateRequest describes fields accepted when updating a monitor group.
type MonitorGroupUpdateRequest = MonitorGroupRequest

type monitorGroupEnvelope struct {
	Data monitorGroupData `json:"data"`
}

type monitorGroupData struct {
	ID         string                 `json:"id,omitempty"`
	Type       string                 `json:"type"`
	Attributes MonitorGroupAttributes `json:"attributes"`
}

type monitorGroupListEnvelope struct {
	Data       []monitorGroupData `json:"data"`
	Pagination struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"pagination"`
}

// Create creates a monitor group in Better Stack.
func (s *MonitorGroupService) Create(ctx context.Context, req MonitorGroupCreateRequest) (MonitorGroup, error) {
	var respEnvelope monitorGroupEnvelope
	if err := s.client.do(ctx, http.MethodPost, "/monitor-groups", req, &respEnvelope); err != nil {
		return MonitorGroup{}, err
	}
	return MonitorGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Get retrieves a monitor group by ID.
func (s *MonitorGroupService) Get(ctx context.Context, id string) (MonitorGroup, error) {
	var respEnvelope monitorGroupEnvelope
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/monitor-groups/%s", url.PathEscape(id)), nil, &respEnvelope); err != nil {
		return MonitorGroup{}, err
	}
	return MonitorGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Update updates an existing monitor group in Better Stack.
func (s *MonitorGroupService) Update(ctx context.Context, id string, req MonitorGroupUpdateRequest) (MonitorGroup, error) {
	var respEnvelope monitorGroupEnvelope
	if err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/monitor-groups/%s", url.PathEscape(id)), req, &respEnvelope); err != nil {
		return MonitorGroup{}, err
	}
	if respEnvelope.Data.ID == "" {
		respEnvelope.Data.ID = id
	}
	return MonitorGroup{ID: respEnvelope.Data.ID, Attributes: respEnvelope.Data.Attributes}, nil
}

// Delete removes a monitor group. Returns nil if the group is already absent.
func (s *MonitorGroupService) Delete(ctx context.Context, id string) error {
	err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/monitor-groups/%s", url.PathEscape(id)), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}

// List returns all monitor groups, following pagination automatically.
func (s *MonitorGroupService) List(ctx context.Context) ([]MonitorGroup, error) {
	path := "/monitor-groups"
	var groups []MonitorGroup

	for path != "" {
		var envelope monitorGroupListEnvelope
		if err := s.client.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
			return nil, err
		}

		for _, item := range envelope.Data {
			groups = append(groups, MonitorGroup{ID: item.ID, Attributes: item.Attributes})
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

// ListMonitors returns all monitors belonging to a monitor group.
func (s *MonitorGroupService) ListMonitors(ctx context.Context, groupID string) ([]Monitor, error) {
	path := fmt.Sprintf("/monitor-groups/%s/monitors", url.PathEscape(groupID))
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

var _ MonitorGroupClient = (*MonitorGroupService)(nil)
