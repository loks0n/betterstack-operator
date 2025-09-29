package httpmock

import (
	"io"
	"net/http"
	"strings"
)

// RoundTripFunc allows mocking the http.RoundTripper interface.
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip executes the mocked transport.
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// JSONResponse builds an *http.Response with a JSON payload.
func JSONResponse(status int, body string) *http.Response {
	if body == "" {
		body = "{}"
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
