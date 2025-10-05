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

func TestMonitorGroupServiceCreate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPost)
		assert.String(t, "path", req.URL.Path, "/monitor-groups")

		var payload map[string]any
		err := json.NewDecoder(req.Body).Decode(&payload)
		assert.NoError(t, err, "decode payload")
		name, ok := payload["name"].(string)
		assert.Bool(t, "name type", ok, true)
		assert.String(t, "name", name, "Backend services")
		paused, ok := payload["paused"].(bool)
		assert.Bool(t, "paused type", ok, true)
		assert.Bool(t, "paused", paused, true)
		sortIndex, ok := payload["sort_index"].(float64)
		assert.Bool(t, "sort_index type", ok, true)
		assert.Int(t, "sort_index", int(sortIndex), 10)

		return httpmock.JSONResponse(http.StatusCreated, `{"data":{"id":"group-1","type":"monitor_group","attributes":{}}}`), nil
	})})

	name := "Backend services"
	paused := true
	sortIndex := 10
	req := MonitorGroupCreateRequest{
		Name:      &name,
		Paused:    &paused,
		SortIndex: &sortIndex,
	}

	group, err := client.MonitorGroups.Create(context.Background(), req)
	assert.NoError(t, err, "CreateMonitorGroup")
	assert.String(t, "id", group.ID, "group-1")
}

func TestMonitorGroupServiceUpdate(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodPatch)
		assert.String(t, "path", req.URL.EscapedPath(), "/monitor-groups/team%2Fgroup")

		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err, "read body")
		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		assert.NoError(t, err, "decode payload")
		assert.Equal(t, "name", payload["name"], "Platform team")

		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"team/group","type":"monitor_group","attributes":{}}}`), nil
	})})

	name := "Platform team"
	req := MonitorGroupUpdateRequest{Name: &name}

	group, err := client.MonitorGroups.Update(context.Background(), "team/group", req)
	assert.NoError(t, err, "UpdateMonitorGroup")
	assert.String(t, "id", group.ID, "team/group")
}

func TestMonitorGroupServiceDelete(t *testing.T) {
	deleted := false
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "method", req.Method, http.MethodDelete)
		assert.String(t, "path", req.URL.EscapedPath(), "/monitor-groups/group-123")
		deleted = true
		return httpmock.JSONResponse(http.StatusNoContent, ""), nil
	})})

	err := client.MonitorGroups.Delete(context.Background(), "group-123")
	assert.NoError(t, err, "DeleteMonitorGroup")
	assert.Bool(t, "delete invoked", deleted, true)
}

func TestMonitorGroupServiceDeleteNotFound(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return httpmock.JSONResponse(http.StatusNotFound, "{}"), nil
	})})

	err := client.MonitorGroups.Delete(context.Background(), "missing")
	assert.NoError(t, err, "Delete monitor group missing")
}

func TestMonitorGroupServiceGet(t *testing.T) {
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.String(t, "path", req.URL.EscapedPath(), "/monitor-groups/group-1")
		return httpmock.JSONResponse(http.StatusOK, `{"data":{"id":"group-1","type":"monitor_group","attributes":{"name":"Backend"}}}`), nil
	})})

	group, err := client.MonitorGroups.Get(context.Background(), "group-1")
	assert.NoError(t, err, "GetMonitorGroup")
	assert.String(t, "id", group.ID, "group-1")
	assert.String(t, "name", group.Attributes.Name, "Backend")
}

func TestMonitorGroupServiceList(t *testing.T) {
	var calls int
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.RequestURI() {
		case "/monitor-groups":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"1","type":"monitor_group","attributes":{"name":"Backend"}}],"pagination":{"next":"https://api.test/monitor-groups?page=2"}}`), nil
		case "/monitor-groups?page=2":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"2","type":"monitor_group","attributes":{"name":"Frontend"}}],"pagination":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.RequestURI())
		}
		return nil, nil
	})})

	groups, err := client.MonitorGroups.List(context.Background())
	assert.NoError(t, err, "List monitor groups")
	assert.Int(t, "call count", calls, 2)
	assert.Int(t, "group count", len(groups), 2)
	assert.String(t, "first name", groups[0].Attributes.Name, "Backend")
	assert.String(t, "second name", groups[1].Attributes.Name, "Frontend")
}

func TestMonitorGroupServiceListMonitors(t *testing.T) {
	var calls int
	client := NewClient("https://api.test", "token", &http.Client{Transport: httpmock.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.RequestURI() {
		case "/monitor-groups/group-1/monitors":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"m1","type":"monitor","attributes":{"pronounceable_name":"First"}}],"pagination":{"next":"https://api.test/monitor-groups/group-1/monitors?page=2"}}`), nil
		case "/monitor-groups/group-1/monitors?page=2":
			return httpmock.JSONResponse(http.StatusOK, `{"data":[{"id":"m2","type":"monitor","attributes":{"pronounceable_name":"Second"}}],"pagination":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected path: %s", req.URL.RequestURI())
		}
		return nil, nil
	})})

	monitors, err := client.MonitorGroups.ListMonitors(context.Background(), "group-1")
	assert.NoError(t, err, "List monitors in group")
	assert.Int(t, "call count", calls, 2)
	assert.Int(t, "monitor count", len(monitors), 2)
	assert.String(t, "first name", monitors[0].Attributes.PronounceableName, "First")
	assert.String(t, "second name", monitors[1].Attributes.PronounceableName, "Second")
}
