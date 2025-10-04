package betterstack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"loks0n/betterstack-operator/internal/testutil/assert"
	"loks0n/betterstack-operator/internal/testutil/httpmock"
)

func TestMonitorServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPost)
		assert.String(t, "path", req.URL.Path, "/monitors")

		var payload map[string]any
		err := json.NewDecoder(req.Body).Decode(&payload)
		assert.NoError(t, err, "decode payload")
		monitorType, ok := payload["monitor_type"].(string)
		assert.Bool(t, "monitor_type type", ok, true)
		assert.String(t, "monitor_type", monitorType, "status")
		url, ok := payload["url"].(string)
		assert.Bool(t, "url type", ok, true)
		assert.String(t, "url", url, "https://example.com")
		name, ok := payload["pronounceable_name"].(string)
		assert.Bool(t, "pronounceable_name type", ok, true)
		assert.String(t, "pronounceable_name", name, "Example")
		email, ok := payload["email"].(bool)
		assert.Bool(t, "email type", ok, true)
		assert.Bool(t, "email", email, true)
		cf, ok := payload["check_frequency"].(float64)
		assert.Bool(t, "check_frequency type", ok, true)
		assert.Int(t, "check_frequency", int(cf), 180)
		headers, ok := payload["request_headers"].([]any)
		assert.Bool(t, "request_headers type", ok, true)
		assert.Int(t, "request_headers length", len(headers), 1)
		custom, ok := payload["custom"].(string)
		assert.Bool(t, "custom type", ok, true)
		assert.String(t, "custom attribute", custom, "value")

		return httpmock.JSONResponse(http.StatusCreated, `{"data":{"id":"monitor-1","type":"monitor","attributes":{}}}`), nil
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
	assert.NoError(t, err, "CreateMonitor")
	assert.String(t, "id", monitor.ID, "monitor-1")
}

func TestMonitorServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPatch)
		assert.String(t, "path", req.URL.EscapedPath(), "/monitors/abc%2F123")

		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err, "read body")
		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		assert.NoError(t, err, "decode payload")
		assert.Equal(t, "paused", payload["paused"], true)

		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"monitor","attributes":{}}}`), nil
	})})

	paused := true
	req := MonitorUpdateRequest{Paused: &paused}

	monitor, err := client.Monitors.Update(context.Background(), "abc/123", req)
	assert.NoError(t, err, "UpdateMonitor")
	assert.String(t, "id", monitor.ID, "abc/123")
}

func TestMonitorServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodDelete)
		assert.String(t, "path", req.URL.EscapedPath(), "/monitors/abc%2F123")
		deleted = true
		return httpmock.JSONResponse(http.StatusNoContent, "{}"), nil
	})})

	err := client.Monitors.Delete(context.Background(), "abc/123")
	assert.NoError(t, err, "DeleteMonitor")
	assert.Bool(t, "delete invoked", deleted, true)
}

func TestMonitorServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, "{}"), nil
	})})

	err := client.Monitors.Delete(context.Background(), "missing")
	assert.NoError(t, err, "DeleteMonitor missing")
}

func TestMonitorServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "path", req.URL.EscapedPath(), "/monitors/abc%2F123")
		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"monitor","attributes":{"url":"https://example.com"}}}`), nil
	})})

	monitor, err := client.Monitors.Get(context.Background(), "abc/123")
	assert.NoError(t, err, "GetMonitor")
	assert.String(t, "id", monitor.ID, "abc/123")
	assert.String(t, "url", monitor.Attributes.URL, "https://example.com")
}

func TestMonitorServiceGetNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, `{"errors":"Resource with provided ID was not found"}`), nil
	})})

	_, err := client.Monitors.Get(context.Background(), "missing")
	assert.Error(t, err, "expected missing monitor")
	assert.Bool(t, "IsNotFound", IsNotFound(err), true)
}

func TestMonitorServiceList(t *testing.T) {
	var calls int
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.RequestURI() {
		case "/monitors":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"1","type":"monitor","attributes":{"pronounceable_name":"First","url":"https://first.example.com"}}],"pagination":{"next":"https://api.test/monitors?page=2"}}`), nil
		case "/monitors?page=2":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"2","type":"monitor","attributes":{"pronounceable_name":"Second","url":"https://second.example.com"}}],"pagination":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.RequestURI())
		}
		return nil, nil
	})})

	monitors, err := client.Monitors.List(context.Background())
	assert.NoError(t, err, "List monitors")
	assert.Int(t, "call count", calls, 2)
	assert.Int(t, "monitor count", len(monitors), 2)
	assert.String(t, "first name", monitors[0].Attributes.PronounceableName, "First")
	assert.String(t, "second url", monitors[1].Attributes.URL, "https://second.example.com")
}
