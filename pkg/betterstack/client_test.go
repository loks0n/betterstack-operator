package betterstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMonitors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("url"); got != "https://example.com" {
			t.Fatalf("expected url filter, got %q", got)
		}
		if got := q.Get("pronounceable_name"); got != "Example" {
			t.Fatalf("expected pronounceable_name filter, got %q", got)
		}
		if got := q.Get("page"); got != "2" {
			t.Fatalf("expected page=2, got %q", got)
		}
		if got := q.Get("per_page"); got != "5" {
			t.Fatalf("expected per_page=5, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"monitor","attributes":{"url":"https://example.com","monitor_type":"status"}}],"pagination":{"first":"f","last":"l","prev":"p","next":"n"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	list, err := client.Monitors.List(context.Background(), ListMonitorsOptions{
		URL:               "https://example.com",
		PronounceableName: "Example",
		Page:              2,
		PerPage:           5,
	})
	if err != nil {
		t.Fatalf("ListMonitors error: %v", err)
	}

	if len(list.Monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(list.Monitors))
	}
	monitor := list.Monitors[0]
	if monitor.ID != "1" {
		t.Fatalf("unexpected monitor id: %s", monitor.ID)
	}
	if monitor.Attributes.URL != "https://example.com" {
		t.Fatalf("unexpected url: %s", monitor.Attributes.URL)
	}
	if monitor.Attributes.MonitorType != "status" {
		t.Fatalf("unexpected monitor type: %s", monitor.Attributes.MonitorType)
	}
	if list.Pagination.Next != "n" {
		t.Fatalf("unexpected pagination: %+v", list.Pagination)
	}
}

func TestCreateMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/monitors" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["monitor_type"] != "status" {
			t.Fatalf("unexpected monitor_type: %v", payload["monitor_type"])
		}
		if payload["url"] != "https://example.com" {
			t.Fatalf("unexpected url: %v", payload["url"])
		}
		if payload["pronounceable_name"] != "Example" {
			t.Fatalf("unexpected pronounceable_name: %v", payload["pronounceable_name"])
		}
		if payload["email"] != true {
			t.Fatalf("expected email true, got %v", payload["email"])
		}
		if cf, ok := payload["check_frequency"].(float64); !ok || cf != 180 {
			t.Fatalf("unexpected check_frequency: %v", payload["check_frequency"])
		}
		headers, ok := payload["request_headers"].([]any)
		if !ok || len(headers) != 1 {
			t.Fatalf("unexpected request_headers: %v", payload["request_headers"])
		}
		if val := payload["custom"]; val != "value" {
			t.Fatalf("expected additional attribute, got %v", val)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"monitor-1","type":"monitor","attributes":{}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	monitorType := "status"
	url := "https://example.com"
	name := "Example"
	email := true
	checkFrequency := 180
	req := MonitorCreateRequest{
		MonitorType:       &monitorType,
		URL:               &url,
		PronounceableName: &name,
		Email:             &email,
		CheckFrequency:    &checkFrequency,
		RequestHeaders:    []MonitorRequestHeader{{Name: "X-Test", Value: "true"}},
		AdditionalAttributes: map[string]any{
			"custom": "value",
		},
	}

	monitor, err := client.Monitors.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateMonitor error: %v", err)
	}
	if monitor.ID != "monitor-1" {
		t.Fatalf("unexpected id: %s", monitor.ID)
	}
}

func TestUpdateMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.EscapedPath() != "/monitors/abc%2F123" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["paused"] != true {
			t.Fatalf("expected paused true, got %v", payload["paused"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc/123","type":"monitor","attributes":{}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	paused := true
	req := MonitorUpdateRequest{
		Paused: &paused,
	}

	monitor, err := client.Monitors.Update(context.Background(), "abc/123", req)
	if err != nil {
		t.Fatalf("UpdateMonitor error: %v", err)
	}
	if monitor.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", monitor.ID)
	}
}

func TestCreateHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/heartbeats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if name := payload["name"]; name != "Example" {
			t.Fatalf("expected name Example, got %v", name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"67890","type":"heartbeat","attributes":{"status":"pending"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	name := "Example"
	heartbeat, err := client.Heartbeats.Create(context.Background(), HeartbeatCreateRequest{Name: &name})
	if err != nil {
		t.Fatalf("CreateHeartbeat error: %v", err)
	}
	if heartbeat.ID != "67890" {
		t.Fatalf("unexpected id: %s", heartbeat.ID)
	}
	if heartbeat.Attributes.Status != HeartbeatStatusPending {
		t.Fatalf("unexpected attributes: %+v", heartbeat.Attributes)
	}
}

func TestUpdateHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if name := payload["name"]; name != "Updated" {
			t.Fatalf("expected name Updated, got %v", name)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"down","name":"Updated"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	name := "Updated"
	heartbeat, err := client.Heartbeats.Update(context.Background(), "abc/123", HeartbeatUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateHeartbeat error: %v", err)
	}
	if heartbeat.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", heartbeat.ID)
	}
	if heartbeat.Attributes.Status != HeartbeatStatusDown {
		t.Fatalf("unexpected attributes: %+v", heartbeat.Attributes)
	}
	if heartbeat.Attributes.Name != "Updated" {
		t.Fatalf("unexpected name: %s", heartbeat.Attributes.Name)
	}
}

func TestDeleteHeartbeat(t *testing.T) {
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	if err := client.Heartbeats.Delete(context.Background(), "abc/123"); err != nil {
		t.Fatalf("DeleteHeartbeat error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete to be called")
	}
}

func TestDeleteHeartbeatNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	if err := client.Heartbeats.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("expected no error for not found, got %v", err)
	}
}

func TestListHeartbeats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/heartbeats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if got := q.Get("page"); got != "2" {
			t.Fatalf("expected page=2, got %q", got)
		}
		if got := q.Get("per_page"); got != "5" {
			t.Fatalf("expected per_page=5, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"12345","type":"heartbeat","attributes":{"status":"up","name":"Testing heartbeat"}}],"pagination":{"first":"f","last":"l","prev":null,"next":null}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	list, err := client.Heartbeats.List(context.Background(), ListHeartbeatsOptions{
		Page:    2,
		PerPage: 5,
	})
	if err != nil {
		t.Fatalf("ListHeartbeats error: %v", err)
	}

	if len(list.Heartbeats) != 1 {
		t.Fatalf("expected 1 heartbeat, got %d", len(list.Heartbeats))
	}
	if list.Heartbeats[0].ID != "12345" {
		t.Fatalf("unexpected heartbeat id: %s", list.Heartbeats[0].ID)
	}
	if list.Heartbeats[0].Attributes.Status != HeartbeatStatusUp {
		t.Fatalf("unexpected attributes: %+v", list.Heartbeats[0].Attributes)
	}
	if list.Pagination.First != "f" {
		t.Fatalf("unexpected pagination: %+v", list.Pagination)
	}
}

func TestGetHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"up"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	heartbeat, err := client.Heartbeats.Get(context.Background(), "abc/123")
	if err != nil {
		t.Fatalf("GetHeartbeat error: %v", err)
	}
	if heartbeat.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", heartbeat.ID)
	}
	if heartbeat.Attributes.Status != HeartbeatStatusUp {
		t.Fatalf("unexpected attributes: %+v", heartbeat.Attributes)
	}
}

func TestGetHeartbeatNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":"Resource with provided ID was not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	_, err := client.Heartbeats.Get(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/monitors/abc%2F123" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc/123","type":"monitor","attributes":{"url":"https://example.com"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	monitor, err := client.Monitors.Get(context.Background(), "abc/123")
	if err != nil {
		t.Fatalf("GetMonitor error: %v", err)
	}
	if monitor.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", monitor.ID)
	}
	if monitor.Attributes.URL != "https://example.com" {
		t.Fatalf("unexpected attributes: %+v", monitor.Attributes)
	}
}

func TestGetMonitorNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":"Resource with provided ID was not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	_, err := client.Monitors.Get(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
