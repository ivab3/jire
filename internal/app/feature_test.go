package app

import (
	"io"
	"net/http"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ivab3/jire/internal/httpclient"
	"github.com/ivab3/jire/internal/project"
)

type appRoundTripFunc func(*http.Request) (*http.Response, error)

func (f appRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestProjectSidebarShowsRecentRequestsWithoutLocalHeading(t *testing.T) {
	m := newTestModel(t)
	owner, err := m.store.CreateProject(t.Context(), "Public API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	request, err := m.store.CreateRequest(t.Context(), owner.ID, "DELETE", "https://example.test/posts/1")
	if err != nil {
		t.Fatalf("CreateRequest returned error: %v", err)
	}
	m.activateProject(owner)
	m.sidebarScreen = sidebarRequests
	m.requests = []project.Request{request}

	view := m.sidebarView(34, 18)
	for _, want := range []string{"← Projects", "Public API", "Recent", "DELETE", "https://example.test"} {
		if !strings.Contains(view, want) {
			t.Fatalf("sidebar does not contain %q: %q", want, view)
		}
	}
	if strings.Contains(view, "Local") {
		t.Fatalf("sidebar should not contain Local heading: %q", view)
	}
}

func TestCreateRequestFlowPersistsAndOpensRequest(t *testing.T) {
	m := newTestModel(t)
	owner, err := m.store.CreateProject(t.Context(), "Public API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	m.activateProject(owner)
	m.sidebarScreen = sidebarRequests
	m.setFocus(focusSidebar)

	next, cmd := m.Update(keyPress("c"))
	m = mustModel(t, next)
	if m.requestMode != requestModeCreate || cmd == nil {
		t.Fatalf("request create mode = %v, cmd nil = %v", m.requestMode, cmd == nil)
	}
	m.requestURLInput.SetValue("https://example.test/posts")

	next, cmd = m.Update(keyPress("enter"))
	m = mustModel(t, next)
	if cmd == nil {
		t.Fatal("expected create request command")
	}
	next, _ = m.Update(cmd())
	m = mustModel(t, next)

	if len(m.requests) != 1 || m.requests[0].Method != "GET" {
		t.Fatalf("requests = %#v", m.requests)
	}
	if !m.hasActiveTab() || m.tabs[m.activeTab].Request.ID != m.requests[0].ID {
		t.Fatalf("active tab = %#v", m.tabs)
	}
	if m.focus != focusHeaderName {
		t.Fatalf("focus = %v, want header name", m.focus)
	}
}

func TestClickingProjectsTitleReturnsToProjectList(t *testing.T) {
	m := modelWithRequest(t)
	m.width = 100
	m.sidebarScreen = sidebarRequests

	next, _ := m.Update(tea.MouseClickMsg{X: 2, Y: 1, Button: tea.MouseLeft})
	m = mustModel(t, next)
	if m.sidebarScreen != sidebarProjects || m.focus != focusSidebar {
		t.Fatalf("sidebar screen = %v, focus = %v", m.sidebarScreen, m.focus)
	}
}

func TestHeadersAreAddedAsKeyValuePairs(t *testing.T) {
	m := modelWithRequest(t)
	m.headerNameInput.SetValue("Accept")
	m.headerValueInput.SetValue("application/json")
	m.setFocus(focusHeaderValue)

	next, _ := m.Update(keyPress("enter"))
	m = mustModel(t, next)
	if len(m.tabs[m.activeTab].Request.Headers) != 1 {
		t.Fatalf("headers = %#v", m.tabs[m.activeTab].Request.Headers)
	}
	header := m.tabs[m.activeTab].Request.Headers[0]
	if !header.Enabled || header.Name != "Accept" || header.Value != "application/json" {
		t.Fatalf("header = %#v", header)
	}
	view := m.editorView(80, 24)
	if !strings.Contains(view, "KEY") || !strings.Contains(view, "VALUE") {
		t.Fatalf("editor should label key and value columns: %q", view)
	}
}

func TestBodyUsesNormalAndInsertModes(t *testing.T) {
	m := modelWithRequest(t)
	m.setFocus(focusBody)
	if m.tabs[m.activeTab].VimMode != vimNormal {
		t.Fatalf("default Vim mode = %v, want normal", m.tabs[m.activeTab].VimMode)
	}

	next, _ := m.Update(keyPress("i"))
	m = mustModel(t, next)
	next, _ = m.Update(keyPress("x"))
	m = mustModel(t, next)
	if m.tabs[m.activeTab].Body.Value() != "x" || m.tabs[m.activeTab].VimMode != vimInsert {
		t.Fatalf("body = %q, mode = %v", m.tabs[m.activeTab].Body.Value(), m.tabs[m.activeTab].VimMode)
	}
	next, _ = m.Update(keyPress("esc"))
	m = mustModel(t, next)
	if m.tabs[m.activeTab].VimMode != vimNormal {
		t.Fatalf("mode after escape = %v, want normal", m.tabs[m.activeTab].VimMode)
	}
}

func TestControlJSendsActiveRequestAndRendersResponse(t *testing.T) {
	m := modelWithRequest(t)
	m.client = httpclient.New()
	m.client.HTTPClient.Transport = appRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.String() != "https://example.test/posts/1" {
			t.Fatalf("sent request = %s %s", request.Method, request.URL)
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"application/json"}},
			Body:          io.NopCloser(strings.NewReader(`{"id":1}`)),
			ContentLength: 8,
		}, nil
	})

	next, cmd := m.Update(controlKey('j'))
	m = mustModel(t, next)
	if cmd == nil || !m.tabs[m.activeTab].Loading {
		t.Fatalf("send cmd nil = %v, loading = %v", cmd == nil, m.tabs[m.activeTab].Loading)
	}
	next, _ = m.Update(cmd())
	m = mustModel(t, next)

	tab := m.tabs[m.activeTab]
	if tab.Loading || !strings.Contains(tab.ResponseTitle, "200 OK") {
		t.Fatalf("response title = %q, loading = %v", tab.ResponseTitle, tab.Loading)
	}
	if !strings.Contains(tab.Response.GetContent(), "\"id\": 1") {
		t.Fatalf("response body was not pretty printed: %q", tab.Response.GetContent())
	}
}

func modelWithRequest(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t)
	owner, err := m.store.CreateProject(t.Context(), "Public API")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	request, err := m.store.CreateRequest(t.Context(), owner.ID, "GET", "https://example.test/posts/1")
	if err != nil {
		t.Fatalf("CreateRequest returned error: %v", err)
	}
	m.activateProject(owner)
	m.sidebarScreen = sidebarRequests
	m.requests = append(m.requests, request)
	m.openRequest(request)
	return m
}

func controlKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: tea.ModCtrl})
}
