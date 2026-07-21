package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultMaxBodyBytes int64 = 5 << 20

var (
	ErrURLRequired       = errors.New("request URL is required")
	ErrUnsupportedScheme = errors.New("request URL must use http or https")
)

type Header struct {
	Enabled bool
	Name    string
	Value   string
}

type Request struct {
	Method  string
	URL     string
	Headers []Header
	Body    string
}

type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	Size       int64
	Truncated  bool
	ReceivedAt time.Time
}

type Client struct {
	HTTPClient   *http.Client
	MaxBodyBytes int64
}

func New() Client {
	return Client{
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		MaxBodyBytes: DefaultMaxBodyBytes,
	}
}

func (c Client) Do(ctx context.Context, input Request) (Response, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodGet
	}
	target := strings.TrimSpace(input.URL)
	if target == "" {
		return Response{}, ErrURLRequired
	}
	parsed, err := url.ParseRequestURI(target)
	if err != nil {
		return Response{}, fmt.Errorf("invalid request URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Response{}, ErrUnsupportedScheme
	}

	req, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(input.Body))
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}
	for _, header := range input.Headers {
		name := strings.TrimSpace(header.Name)
		if !header.Enabled || name == "" {
			continue
		}
		req.Header.Add(name, header.Value)
	}

	client := c.HTTPClient
	if client == nil {
		client = New().HTTPClient
	}
	limit := c.MaxBodyBytes
	if limit <= 0 {
		limit = DefaultMaxBodyBytes
	}

	started := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(started)
	if err != nil {
		return Response{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}
	truncated := int64(len(body)) > limit
	if truncated {
		body = bytes.Clone(body[:limit])
	}

	size := int64(len(body))
	if resp.ContentLength >= 0 {
		size = resp.ContentLength
	}
	return Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header.Clone(),
		Body:       body,
		Duration:   duration,
		Size:       size,
		Truncated:  truncated,
		ReceivedAt: time.Now().UTC(),
	}, nil
}
