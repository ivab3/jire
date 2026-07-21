package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientSendsRequestAndCapturesResponse(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("X-Test") != "jire" {
			t.Errorf("X-Test = %q, want jire", r.Header.Get("X-Test"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"hello":"world"}` {
			t.Errorf("body = %q", body)
		}
		return &http.Response{
			StatusCode:    http.StatusCreated,
			Status:        "201 Created",
			Header:        http.Header{"Content-Type": []string{"application/json"}},
			Body:          io.NopCloser(strings.NewReader(`{"ok":true}`)),
			ContentLength: 11,
		}, nil
	})

	client := New()
	client.HTTPClient.Transport = transport
	response, err := client.Do(context.Background(), Request{
		Method: "POST",
		URL:    "https://example.test/posts",
		Headers: []Header{
			{Enabled: true, Name: "X-Test", Value: "jire"},
			{Enabled: false, Name: "X-Skipped", Value: "no"},
		},
		Body: `{"hello":"world"}`,
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if response.StatusCode != http.StatusCreated || string(response.Body) != `{"ok":true}` {
		t.Fatalf("response = %#v", response)
	}
	if response.Duration <= 0 || response.ReceivedAt.IsZero() {
		t.Fatalf("response timing was not captured: %#v", response)
	}
}

func TestClientTruncatesLargeResponse(t *testing.T) {
	client := New()
	client.HTTPClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{},
			Body:          io.NopCloser(strings.NewReader(strings.Repeat("x", 12))),
			ContentLength: 12,
		}, nil
	})
	client.MaxBodyBytes = 5
	response, err := client.Do(context.Background(), Request{URL: "https://example.test/large"})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if !response.Truncated || string(response.Body) != "xxxxx" || response.Size != 12 {
		t.Fatalf("response = %#v", response)
	}
}

func TestClientRejectsNonHTTPURL(t *testing.T) {
	_, err := New().Do(context.Background(), Request{URL: "file:///tmp/example"})
	if err == nil {
		t.Fatal("Do returned nil error")
	}
}
