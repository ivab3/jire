package app

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/ivab3/jire/internal/httpclient"
	"github.com/ivab3/jire/internal/project"
)

func (m Model) loadProjectsCmd() tea.Cmd {
	store := m.store
	return func() tea.Msg {
		projects, err := store.ListProjects(context.Background())
		if err != nil {
			return projectStoreErrorMsg{Err: err}
		}
		return projectsLoadedMsg{Projects: projects}
	}
}

func (m Model) createProjectCmd(name string) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		ctx := context.Background()
		created, err := store.CreateProject(ctx, name)
		if err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		projects, err := store.ListProjects(ctx)
		if err != nil {
			return projectStoreErrorMsg{Err: err}
		}
		return projectCreatedMsg{Project: created, Projects: projects}
	}
}

func (m Model) openSelectedProjectCmd() tea.Cmd {
	projects := m.visibleProjects()
	if len(projects) == 0 || m.projectRow < 0 || m.projectRow >= len(projects) {
		return nil
	}
	selected := projects[m.projectRow]
	store := m.store
	return func() tea.Msg {
		ctx := context.Background()
		if err := store.TouchProject(ctx, selected.ID); err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		opened, err := store.LoadProject(ctx, selected.ID)
		if err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		requests, err := store.ListRequests(ctx, opened.ID)
		if err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		projects, err := store.ListProjects(ctx)
		if err != nil {
			return projectStoreErrorMsg{Err: err}
		}
		return projectOpenedMsg{Project: opened, Projects: projects, Requests: requests}
	}
}

func (m Model) createRequestCmd(method string, url string) tea.Cmd {
	if m.activeProject == nil {
		return nil
	}
	store := m.store
	projectID := m.activeProject.ID
	return func() tea.Msg {
		ctx := context.Background()
		created, err := store.CreateRequest(ctx, projectID, method, url)
		if err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		requests, err := store.ListRequests(ctx, projectID)
		if err != nil {
			return projectStoreErrorMsg{Err: userFacingProjectError(err)}
		}
		projects, err := store.ListProjects(ctx)
		if err != nil {
			return projectStoreErrorMsg{Err: err}
		}
		return requestCreatedMsg{Request: created, Requests: requests, Projects: projects}
	}
}

func (m Model) sendActiveRequestCmd() tea.Cmd {
	if !m.hasActiveTab() {
		return nil
	}
	request := m.tabs[m.activeTab].Request
	store := m.store
	client := m.client
	return func() tea.Msg {
		ctx := context.Background()
		if err := store.SaveRequest(ctx, request); err != nil {
			return requestFailedMsg{RequestID: request.ID, Err: userFacingProjectError(err)}
		}

		response, err := client.Do(ctx, toHTTPRequest(request))
		requests, listErr := store.ListRequests(ctx, request.ProjectID)
		if listErr != nil && err == nil {
			err = listErr
		}
		projects, projectsErr := store.ListProjects(ctx)
		if projectsErr != nil && err == nil {
			err = projectsErr
		}
		if err != nil {
			return requestFailedMsg{RequestID: request.ID, Err: err, Requests: requests, Projects: projects}
		}
		return requestCompletedMsg{RequestID: request.ID, Response: response, Requests: requests, Projects: projects}
	}
}

func toHTTPRequest(request project.Request) httpclient.Request {
	headers := make([]httpclient.Header, 0, len(request.Headers))
	for _, header := range request.Headers {
		headers = append(headers, httpclient.Header{
			Enabled: header.Enabled,
			Name:    header.Name,
			Value:   header.Value,
		})
	}
	return httpclient.Request{
		Method:  request.Method,
		URL:     request.URL,
		Headers: headers,
		Body:    request.Body,
	}
}

func userFacingProjectError(err error) error {
	switch {
	case errors.Is(err, project.ErrProjectNameRequired):
		return errors.New("project name is required")
	case errors.Is(err, project.ErrProjectNotFound):
		return errors.New("project was not found")
	case errors.Is(err, project.ErrRequestURLRequired), errors.Is(err, httpclient.ErrURLRequired):
		return errors.New("request URL is required")
	case errors.Is(err, project.ErrRequestNotFound):
		return errors.New("request was not found")
	default:
		return fmt.Errorf("%w", err)
	}
}
