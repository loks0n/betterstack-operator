package betterstack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHeartbeatServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/heartbeats" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if name := payload["name"]; name != "Example" {
			t.Fatalf("expected name Example, got %v", name)
		}

		return jsonResponse(http.StatusCreated, `{"data":{"id":"67890","type":"heartbeat","attributes":{"status":"pending"}}}`), nil
	})})

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

func TestHeartbeatServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", req.URL.EscapedPath())
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if name := payload["name"]; name != "Updated" {
			t.Fatalf("expected name Updated, got %v", name)
		}

		return jsonResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"down","name":"Updated"}}}`), nil
	})})

	name := "Updated"
	heartbeat, err := client.Heartbeats.Update(context.Background(), "abc/123", HeartbeatUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateHeartbeat error: %v", err)
	}
	if heartbeat.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", heartbeat.ID)
	}
	if heartbeat.Attributes.Status != HeartbeatStatusDown {
		t.Fatalf("unexpected status: %s", heartbeat.Attributes.Status)
	}
	if heartbeat.Attributes.Name != "Updated" {
		t.Fatalf("unexpected name: %s", heartbeat.Attributes.Name)
	}
}

func TestHeartbeatServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", req.URL.EscapedPath())
		}
		deleted = true
		return jsonResponse(http.StatusNoContent, "{}"), nil
	})})

	if err := client.Heartbeats.Delete(context.Background(), "abc/123"); err != nil {
		t.Fatalf("DeleteHeartbeat error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete to be called")
	}
}

func TestHeartbeatServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, "{}"), nil
	})})

	if err := client.Heartbeats.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("expected not found delete to be ignored, got %v", err)
	}
}

func TestHeartbeatServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/heartbeats/abc%2F123" {
			t.Fatalf("unexpected path: %s", req.URL.EscapedPath())
		}
		return jsonResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"up"}}}`), nil
	})})

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

func TestHeartbeatServiceGetNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{"errors":"Resource with provided ID was not found"}`), nil
	})})

	if _, err := client.Heartbeats.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error")
	} else if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
