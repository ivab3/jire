package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ivab3/jire/internal/httpclient"
	"github.com/ivab3/jire/internal/project"
)

func TestSidebarShowsProjectEmptyState(t *testing.T) {
	m := newTestModel(t)

	view := m.sidebarView(28, 12)
	if !strings.Contains(view, "Projects") {
		t.Fatalf("sidebar view does not contain Projects header: %q", view)
	}
	if !strings.Contains(view, "Press c to create a project.") {
		t.Fatalf("sidebar view does not contain empty state: %q", view)
	}
	if strings.Contains(view, "jire") {
		t.Fatalf("sidebar view should not render the jire title: %q", view)
	}
}

func TestCreateProjectFlowPersistsProjectAndOpensDraft(t *testing.T) {
	m := newTestModel(t)

	next, cmd := m.Update(keyPress("c"))
	m = mustModel(t, next)
	if m.projectMode != projectModeCreate {
		t.Fatalf("project mode = %v, want create", m.projectMode)
	}
	if cmd == nil {
		t.Fatal("expected create input focus command")
	}

	m.projectCreateInput.SetValue("Local API")
	next, cmd = m.Update(keyPress("enter"))
	m = mustModel(t, next)
	if cmd == nil {
		t.Fatal("expected create project command")
	}

	next, _ = m.Update(cmd())
	m = mustModel(t, next)

	if m.activeProject == nil {
		t.Fatal("active project is nil")
	}
	if m.activeProject.Name != "Local API" {
		t.Fatalf("active project name = %q, want %q", m.activeProject.Name, "Local API")
	}
	if m.focus != focusBody {
		t.Fatalf("focus = %v, want body", m.focus)
	}
	if len(m.tabs) != 1 {
		t.Fatalf("tabs = %d, want 1", len(m.tabs))
	}
	if got := m.tabs[0].Request; got.Name != "Untitled request" || got.Method != "GET" || got.URL != "" || got.ProjectID != m.activeProject.ID {
		t.Fatalf("draft request = %#v, active project ID = %q", got, m.activeProject.ID)
	}

	projectPath := filepath.Join(m.store.Root, "projects", "local-api", "project.json")
	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("expected persisted project at %s: %v", projectPath, err)
	}
}

func TestProjectSearchFiltersByCaseInsensitiveName(t *testing.T) {
	m := newTestModel(t)
	ctx := t.Context()

	localAPI, err := m.store.CreateProject(ctx, "Local API")
	if err != nil {
		t.Fatalf("CreateProject Local API returned error: %v", err)
	}
	if _, err := m.store.CreateProject(ctx, "Billing Sandbox"); err != nil {
		t.Fatalf("CreateProject Billing Sandbox returned error: %v", err)
	}
	projects, err := m.store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	m.projects = projects
	m.projectMode = projectModeSearch
	m.projectSearchInput.SetValue("api")

	visible := m.visibleProjects()
	if len(visible) != 1 {
		t.Fatalf("visible projects = %#v, want exactly one match", visible)
	}
	if visible[0].ID != localAPI.ID {
		t.Fatalf("visible project ID = %q, want %q", visible[0].ID, localAPI.ID)
	}
}

func TestProjectCommandKeysAreScopedAwayFromEditorInput(t *testing.T) {
	m := newTestModel(t)
	created, err := m.store.CreateProject(t.Context(), "Local API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	m.activateProject(created)
	m.openEmptyDraft(created.ID)
	m.setFocus(focusBody)

	next, _ := m.Update(keyPress("i"))
	m = mustModel(t, next)
	next, _ = m.Update(keyPress("c"))
	m = mustModel(t, next)
	next, _ = m.Update(keyPress("s"))
	m = mustModel(t, next)

	if m.projectMode != projectModeNormal {
		t.Fatalf("project mode = %v, want normal", m.projectMode)
	}
	if got := m.tabs[0].Body.Value(); got != "cs" {
		t.Fatalf("editor body = %q, want %q", got, "cs")
	}
}

func newTestModel(t *testing.T) Model {
	t.Helper()

	return newModel(project.NewStore(t.TempDir()), httpclient.New())
}

func mustModel(t *testing.T, model tea.Model) Model {
	t.Helper()

	m, ok := model.(Model)
	if !ok {
		t.Fatalf("model has type %T, want app.Model", model)
	}
	return m
}

func keyPress(key string) tea.KeyPressMsg {
	switch key {
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "up":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	case "down":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	}

	runes := []rune(key)
	if len(runes) == 0 {
		return tea.KeyPressMsg(tea.Key{})
	}
	return tea.KeyPressMsg(tea.Key{Code: runes[0], Text: key})
}
