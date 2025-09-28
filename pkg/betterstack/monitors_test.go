package betterstack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMonitorServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/monitors" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		var payload map[string]any
		withBody := json.NewDecoder(req.Body)
		if err := withBody.Decode(&payload); err != nil {
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

		return jsonResponse(http.StatusCreated, `{"data":{"id":"monitor-1","type":"monitor","attributes":{}}}`), nil
	})})

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

func TestMonitorServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.EscapedPath() != "/monitors/abc%2F123" {
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
		if payload["paused"] != true {
			t.Fatalf("expected paused true, got %v", payload["paused"])
		}

		return jsonResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"monitor","attributes":{}}}`), nil
	})})

	paused := true
	req := MonitorUpdateRequest{Paused: &paused}

	monitor, err := client.Monitors.Update(context.Background(), "abc/123", req)
	if err != nil {
		t.Fatalf("UpdateMonitor error: %v", err)
	}
	if monitor.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", monitor.ID)
	}
}

func TestMonitorServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.EscapedPath() != "/monitors/abc%2F123" {
			t.Fatalf("unexpected path: %s", req.URL.EscapedPath())
		}
		deleted = true
		return jsonResponse(http.StatusNoContent, "{}"), nil
	})})

	if err := client.Monitors.Delete(context.Background(), "abc/123"); err != nil {
		t.Fatalf("DeleteMonitor error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete call")
	}
}

func TestMonitorServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, "{}"), nil
	})})

	if err := client.Monitors.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("expected no error for not found delete, got %v", err)
	}
}

func TestMonitorServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/monitors/abc%2F123" {
			t.Fatalf("unexpected path: %s", req.URL.EscapedPath())
		}
		return jsonResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"monitor","attributes":{"url":"https://example.com"}}}`), nil
	})})

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

func TestMonitorServiceGetNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{"errors":"Resource with provided ID was not found"}`), nil
	})})

	if _, err := client.Monitors.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error")
	} else if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
