package betterstack

import (
	"context"
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
		_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"monitor","attributes":{"url":"https://example.com"}}],"pagination":{"first":"f","last":"l","prev":"p","next":"n"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	list, err := client.ListMonitors(context.Background(), ListMonitorsOptions{
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
	if list.Monitors[0].ID != "1" {
		t.Fatalf("unexpected monitor id: %s", list.Monitors[0].ID)
	}
	if list.Pagination.Next != "n" {
		t.Fatalf("unexpected pagination: %+v", list.Pagination)
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

	monitor, err := client.GetMonitor(context.Background(), "abc/123")
	if err != nil {
		t.Fatalf("GetMonitor error: %v", err)
	}
	if monitor.ID != "abc/123" {
		t.Fatalf("unexpected id: %s", monitor.ID)
	}
	if url := monitor.Attributes["url"]; url != "https://example.com" {
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

	_, err := client.GetMonitor(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
