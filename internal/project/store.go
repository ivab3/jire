package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectNotFound     = errors.New("project not found")
	ErrRequestURLRequired  = errors.New("request URL is required")
	ErrRequestNotFound     = errors.New("request not found")
)

type Store struct {
	Root string
}

func DefaultRoot() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "jire"), nil
	}

	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".config", "jire"), nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jire"), nil
}

func NewStore(root string) Store {
	return Store{Root: root}
}

func (s Store) ListProjects(ctx context.Context) ([]Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.projectsRoot())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(s.projectsRoot(), entry.Name(), "project.json")
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}

		var project Project
		if err := json.Unmarshal(data, &project); err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		projects = append(projects, project)
	}

	sortProjects(projects)
	return projects, nil
}

func (s Store) CreateProject(ctx context.Context, name string) (Project, error) {
	if err := ctx.Err(); err != nil {
		return Project{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrProjectNameRequired
	}

	id, err := newID("prj")
	if err != nil {
		return Project{}, err
	}

	slug, err := s.availableSlug(slugify(name))
	if err != nil {
		return Project{}, err
	}

	now := time.Now().UTC()
	project := Project{
		ID:        id,
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.writeProject(project); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Join(s.projectDir(project), "requests"), 0o755); err != nil {
		return Project{}, err
	}
	history, err := os.OpenFile(filepath.Join(s.projectDir(project), "history.jsonl"), os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return Project{}, err
	}
	if err := history.Close(); err != nil {
		return Project{}, err
	}

	return project, nil
}

func (s Store) LoadProject(ctx context.Context, id string) (Project, error) {
	if err := ctx.Err(); err != nil {
		return Project{}, err
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		return Project{}, err
	}

	for _, project := range projects {
		if project.ID == id {
			return project, nil
		}
	}
	return Project{}, ErrProjectNotFound
}

func (s Store) TouchProject(ctx context.Context, id string) error {
	project, err := s.LoadProject(ctx, id)
	if err != nil {
		return err
	}

	project.UpdatedAt = time.Now().UTC()
	return s.writeProject(project)
}

func (s Store) ListRequests(ctx context.Context, projectID string) ([]Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	owner, err := s.LoadProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(filepath.Join(s.projectDir(owner), "requests"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	requests := make([]Request, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.projectDir(owner), "requests", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var request Request
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if request.ProjectID != owner.ID {
			continue
		}
		requests = append(requests, request)
	}

	sortRequests(requests)
	return requests, nil
}

func (s Store) CreateRequest(ctx context.Context, projectID string, method string, url string) (Request, error) {
	if err := ctx.Err(); err != nil {
		return Request{}, err
	}

	owner, err := s.LoadProject(ctx, projectID)
	if err != nil {
		return Request{}, err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return Request{}, ErrRequestURLRequired
	}

	id, err := newID("req")
	if err != nil {
		return Request{}, err
	}
	now := time.Now().UTC()
	request := Request{
		ID:        id,
		ProjectID: owner.ID,
		Name:      url,
		Method:    strings.ToUpper(strings.TrimSpace(method)),
		URL:       url,
		Headers:   []Header{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if request.Method == "" {
		request.Method = "GET"
	}

	if err := s.SaveRequest(ctx, request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (s Store) SaveRequest(ctx context.Context, request Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(request.ID) == "" {
		return ErrRequestNotFound
	}

	owner, err := s.LoadProject(ctx, request.ProjectID)
	if err != nil {
		return err
	}
	request.URL = strings.TrimSpace(request.URL)
	if request.URL == "" {
		return ErrRequestURLRequired
	}
	request.Method = strings.ToUpper(strings.TrimSpace(request.Method))
	if request.Method == "" {
		request.Method = "GET"
	}
	if strings.TrimSpace(request.Name) == "" {
		request.Name = request.URL
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	request.UpdatedAt = time.Now().UTC()
	if request.Headers == nil {
		request.Headers = []Header{}
	}

	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	requestsDir := filepath.Join(s.projectDir(owner), "requests")
	if err := os.MkdirAll(requestsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(requestsDir, request.ID+".json"), data, 0o644); err != nil {
		return err
	}
	return s.TouchProject(ctx, owner.ID)
}

func (s Store) projectsRoot() string {
	return filepath.Join(s.Root, "projects")
}

func (s Store) projectDir(project Project) string {
	return filepath.Join(s.projectsRoot(), project.Slug)
}

func (s Store) availableSlug(base string) (string, error) {
	for i := 1; ; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}

		_, err := os.Stat(filepath.Join(s.projectsRoot(), candidate))
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
}

func (s Store) writeProject(project Project) error {
	dir := s.projectDir(project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(filepath.Join(dir, "project.json"), data, 0o644)
}

func slugify(name string) string {
	var b strings.Builder
	inSeparator := false

	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			inSeparator = false
			continue
		}
		if b.Len() > 0 && !inSeparator {
			b.WriteByte('-')
			inSeparator = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "project"
	}
	return slug
}

func newID(prefix string) (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}

func sortProjects(projects []Project) {
	sort.SliceStable(projects, func(i int, j int) bool {
		if !projects[i].UpdatedAt.Equal(projects[j].UpdatedAt) {
			return projects[i].UpdatedAt.After(projects[j].UpdatedAt)
		}
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
}

func sortRequests(requests []Request) {
	sort.SliceStable(requests, func(i int, j int) bool {
		if !requests[i].UpdatedAt.Equal(requests[j].UpdatedAt) {
			return requests[i].UpdatedAt.After(requests[j].UpdatedAt)
		}
		return requests[i].ID < requests[j].ID
	})
}
