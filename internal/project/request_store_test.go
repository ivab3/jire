package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreCreateSaveAndListRequests(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())
	owner, err := store.CreateProject(ctx, "Public API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	created, err := store.CreateRequest(ctx, owner.ID, "post", "https://example.com/posts")
	if err != nil {
		t.Fatalf("CreateRequest returned error: %v", err)
	}
	if created.Method != "POST" || created.URL != "https://example.com/posts" {
		t.Fatalf("created request = %#v", created)
	}

	created.Headers = []Header{{Enabled: true, Name: "Accept", Value: "application/json"}}
	created.Body = `{"hello":"world"}`
	if err := store.SaveRequest(ctx, created); err != nil {
		t.Fatalf("SaveRequest returned error: %v", err)
	}

	requests, err := store.ListRequests(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListRequests returned error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("ListRequests returned %d requests, want 1", len(requests))
	}
	if requests[0].Headers[0].Name != "Accept" || requests[0].Body != created.Body {
		t.Fatalf("listed request = %#v", requests[0])
	}

	path := filepath.Join(store.Root, "projects", owner.Slug, "requests", created.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected request JSON at %s: %v", path, err)
	}
}

func TestStoreRejectsBlankRequestURL(t *testing.T) {
	store := NewStore(t.TempDir())
	owner, err := store.CreateProject(context.Background(), "Public API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	_, err = store.CreateRequest(context.Background(), owner.ID, "GET", "  ")
	if !errors.Is(err, ErrRequestURLRequired) {
		t.Fatalf("CreateRequest error = %v, want %v", err, ErrRequestURLRequired)
	}
}
