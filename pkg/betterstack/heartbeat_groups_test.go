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

func TestHeartbeatGroupServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPost)
		assert.String(t, "path", req.URL.Path, "/heartbeat-groups")

		var payload map[string]any
		err := json.NewDecoder(req.Body).Decode(&payload)
		assert.NoError(t, err, "decode payload")
		name, ok := payload["name"].(string)
		assert.Bool(t, "name type", ok, true)
		assert.String(t, "name", name, "Backend services")
		paused, ok := payload["paused"].(bool)
		assert.Bool(t, "paused type", ok, true)
		assert.Bool(t, "paused", paused, true)

		return httpmock.JSONResponse(http.StatusCreated, `{"data":{"id":"group-1","type":"heartbeat_group","attributes":{}}}`), nil
	})})

	name := "Backend services"
	paused := true
	group, err := client.HeartbeatGroups.Create(context.Background(), HeartbeatGroupCreateRequest{Name: &name, Paused: &paused})
	assert.NoError(t, err, "CreateHeartbeatGroup")
	assert.String(t, "id", group.ID, "group-1")
}

func TestHeartbeatGroupServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPatch)
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeat-groups/team%2Fgroup")

		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err, "read body")
		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		assert.NoError(t, err, "decode payload")
		assert.Equal(t, "name", payload["name"], "Platform team")

		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"team/group","type":"heartbeat_group","attributes":{}}}`), nil
	})})

	name := "Platform team"
	group, err := client.HeartbeatGroups.Update(context.Background(), "team/group", HeartbeatGroupUpdateRequest{Name: &name})
	assert.NoError(t, err, "UpdateHeartbeatGroup")
	assert.String(t, "id", group.ID, "team/group")
}

func TestHeartbeatGroupServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodDelete)
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeat-groups/group-123")
		deleted = true
		return httpmock.JSONResponse(http.StatusNoContent, ""), nil
	})})

	err := client.HeartbeatGroups.Delete(context.Background(), "group-123")
	assert.NoError(t, err, "DeleteHeartbeatGroup")
	assert.Bool(t, "delete invoked", deleted, true)
}

func TestHeartbeatGroupServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, "{}"), nil
	})})

	err := client.HeartbeatGroups.Delete(context.Background(), "missing")
	assert.NoError(t, err, "Delete heartbeat group missing")
}

func TestHeartbeatGroupServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "path", req.URL.EscapedPath(), "/heartbeat-groups/group-1")
		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"group-1","type":"heartbeat_group","attributes":{"name":"Backend"}}}`), nil
	})})

	group, err := client.HeartbeatGroups.Get(context.Background(), "group-1")
	assert.NoError(t, err, "GetHeartbeatGroup")
	assert.String(t, "id", group.ID, "group-1")
	assert.String(t, "name", group.Attributes.Name, "Backend")
}

func TestHeartbeatGroupServiceList(t *testing.T) {
	var calls int
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.RequestURI() {
		case "/heartbeat-groups":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"1","type":"heartbeat_group","attributes":{"name":"Backend"}}],"pagination":{"next":"https://api.test/heartbeat-groups?page=2"}}`), nil
		case "/heartbeat-groups?page=2":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"2","type":"heartbeat_group","attributes":{"name":"Frontend"}}],"pagination":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.RequestURI())
		}
		return nil, nil
	})})

	groups, err := client.HeartbeatGroups.List(context.Background())
	assert.NoError(t, err, "List heartbeat groups")
	assert.Int(t, "call count", calls, 2)
	assert.Int(t, "group count", len(groups), 2)
	assert.String(t, "first name", groups[0].Attributes.Name, "Backend")
	assert.String(t, "second name", groups[1].Attributes.Name, "Frontend")
}

func TestHeartbeatGroupServiceListHeartbeats(t *testing.T) {
	var calls int
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.RequestURI() {
		case "/heartbeat-groups/group-1/heartbeats":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"h1","type":"heartbeat","attributes":{"name":"Primary"}}],"pagination":{"next":"https://api.test/heartbeat-groups/group-1/heartbeats?page=2"}}`), nil
		case "/heartbeat-groups/group-1/heartbeats?page=2":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"h2","type":"heartbeat","attributes":{"name":"Backup"}}],"pagination":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.RequestURI())
		}
		return nil, nil
	})})

	heartbeats, err := client.HeartbeatGroups.ListHeartbeats(context.Background(), "group-1")
	assert.NoError(t, err, "List heartbeats in group")
	assert.Int(t, "call count", calls, 2)
	assert.Int(t, "heartbeat count", len(heartbeats), 2)
	assert.String(t, "first name", heartbeats[0].Attributes.Name, "Primary")
	assert.String(t, "second name", heartbeats[1].Attributes.Name, "Backup")
}
