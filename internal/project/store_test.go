package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreateListLoadProject(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())

	created, err := store.CreateProject(ctx, "Local API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if created.ID == "" {
		t.Fatal("created project ID is empty")
	}
	if created.Name != "Local API" {
		t.Fatalf("created project name = %q, want %q", created.Name, "Local API")
	}
	if created.Slug != "local-api" {
		t.Fatalf("created project slug = %q, want %q", created.Slug, "local-api")
	}

	projectPath := filepath.Join(store.Root, "projects", "local-api", "project.json")
	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("expected project.json at %s: %v", projectPath, err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "projects", "local-api", "requests")); err != nil {
		t.Fatalf("expected requests directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "projects", "local-api", "history.jsonl")); err != nil {
		t.Fatalf("expected history.jsonl: %v", err)
	}

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("ListProjects returned %d projects, want 1", len(projects))
	}
	if projects[0].ID != created.ID {
		t.Fatalf("listed project ID = %q, want %q", projects[0].ID, created.ID)
	}

	loaded, err := store.LoadProject(ctx, created.ID)
	if err != nil {
		t.Fatalf("LoadProject returned error: %v", err)
	}
	if loaded != created {
		t.Fatalf("loaded project = %#v, want %#v", loaded, created)
	}
}

func TestStoreRejectsBlankProjectName(t *testing.T) {
	store := NewStore(t.TempDir())

	_, err := store.CreateProject(context.Background(), "   ")
	if !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("CreateProject error = %v, want %v", err, ErrProjectNameRequired)
	}

	projects, listErr := store.ListProjects(context.Background())
	if listErr != nil {
		t.Fatalf("ListProjects returned error: %v", listErr)
	}
	if len(projects) != 0 {
		t.Fatalf("ListProjects returned %d projects, want 0", len(projects))
	}
}

func TestStoreAddsSlugSuffixOnCollision(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())

	first, err := store.CreateProject(ctx, "Local API")
	if err != nil {
		t.Fatalf("CreateProject first returned error: %v", err)
	}
	second, err := store.CreateProject(ctx, "Local API")
	if err != nil {
		t.Fatalf("CreateProject second returned error: %v", err)
	}

	if first.Slug != "local-api" {
		t.Fatalf("first slug = %q, want %q", first.Slug, "local-api")
	}
	if second.Slug != "local-api-2" {
		t.Fatalf("second slug = %q, want %q", second.Slug, "local-api-2")
	}
}

func TestStoreListProjectsSortsByUpdatedAtThenName(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())
	oldTime := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(time.Hour)

	writeFixtureProject(t, store, Project{
		ID:        "prj_a",
		Name:      "Alpha",
		Slug:      "alpha",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	})
	writeFixtureProject(t, store, Project{
		ID:        "prj_c",
		Name:      "Charlie",
		Slug:      "charlie",
		CreatedAt: oldTime,
		UpdatedAt: newTime,
	})
	writeFixtureProject(t, store, Project{
		ID:        "prj_b",
		Name:      "Bravo",
		Slug:      "bravo",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	})

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	got := []string{projects[0].Name, projects[1].Name, projects[2].Name}
	want := []string{"Charlie", "Alpha", "Bravo"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("project order = %v, want %v", got, want)
		}
	}
}

func TestStoreTouchProjectUpdatesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())

	created, err := store.CreateProject(ctx, "Local API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	before := created.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := store.TouchProject(ctx, created.ID); err != nil {
		t.Fatalf("TouchProject returned error: %v", err)
	}

	loaded, err := store.LoadProject(ctx, created.ID)
	if err != nil {
		t.Fatalf("LoadProject returned error: %v", err)
	}
	if !loaded.UpdatedAt.After(before) {
		t.Fatalf("UpdatedAt = %v, want after %v", loaded.UpdatedAt, before)
	}
}

func writeFixtureProject(t *testing.T, store Store, project Project) {
	t.Helper()

	dir := filepath.Join(store.Root, "projects", project.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
