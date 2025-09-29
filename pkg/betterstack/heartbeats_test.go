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

func TestHeartbeatServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPost)
		assert.String(t, "path", req.URL.Path, "/heartbeats")

		var payload map[string]any
		err := json.NewDecoder(req.Body).Decode(&payload)
		assert.NoError(t, err, "decode payload")
		name, ok := payload["name"].(string)
		assert.Bool(t, "payload name type", ok, true)
		assert.String(t, "name", name, "Example")

		return httpmock.JSONResponse(http.StatusCreated, `{"data":{"id":"67890","type":"heartbeat","attributes":{"status":"pending"}}}`), nil
	})})

	name := "Example"
	heartbeat, err := client.Heartbeats.Create(context.Background(), HeartbeatCreateRequest{Name: &name})
	assert.NoError(t, err, "CreateHeartbeat")
	assert.String(t, "id", heartbeat.ID, "67890")
	assert.Equal(t, "status", heartbeat.Attributes.Status, HeartbeatStatusPending)
}

func TestHeartbeatServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPatch)
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeats/abc%2F123")

		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err, "read body")
		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		assert.NoError(t, err, "decode payload")
		name, ok := payload["name"].(string)
		assert.Bool(t, "payload name type", ok, true)
		assert.String(t, "name", name, "Updated")

		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"down","name":"Updated"}}}`), nil
	})})

	name := "Updated"
	heartbeat, err := client.Heartbeats.Update(context.Background(), "abc/123", HeartbeatUpdateRequest{Name: &name})
	assert.NoError(t, err, "UpdateHeartbeat")
	assert.String(t, "id", heartbeat.ID, "abc/123")
	assert.Equal(t, "status", heartbeat.Attributes.Status, HeartbeatStatusDown)
	assert.String(t, "name", heartbeat.Attributes.Name, "Updated")
}

func TestHeartbeatServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodDelete)
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeats/abc%2F123")
		deleted = true
		return httpmock.JSONResponse(http.StatusNoContent, "{}"), nil
	})})

	err := client.Heartbeats.Delete(context.Background(), "abc/123")
	assert.NoError(t, err, "DeleteHeartbeat")
	assert.Bool(t, "delete invoked", deleted, true)
}

func TestHeartbeatServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, "{}"), nil
	})})

	err := client.Heartbeats.Delete(context.Background(), "missing")
	assert.NoError(t, err, "DeleteHeartbeat missing")
}

func TestHeartbeatServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeats/abc%2F123")
		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"abc/123","type":"heartbeat","attributes":{"status":"up"}}}`), nil
	})})

	heartbeat, err := client.Heartbeats.Get(context.Background(), "abc/123")
	assert.NoError(t, err, "GetHeartbeat")
	assert.String(t, "id", heartbeat.ID, "abc/123")
	assert.Equal(t, "status", heartbeat.Attributes.Status, HeartbeatStatusUp)
}

func TestHeartbeatServiceGetNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, `{"errors":"Resource with provided ID was not found"}`), nil
	})})

	_, err := client.Heartbeats.Get(context.Background(), "missing")
	assert.Error(t, err, "expected missing heartbeat")
	assert.Bool(t, "IsNotFound", IsNotFound(err), true)
}
