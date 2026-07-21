# Projects Feature Plan

This document describes the first real implementation pass for projects in
`jire`. It is a planning document for future agents. Do not treat it as an
already implemented feature.

## Goal

A project is the parent entity for saved requests and request history. Projects
let a developer keep requests for different APIs, services, or experiments
separate while staying inside the same calm terminal workflow.

The initial version should make projects usable without expanding the product
past the HTTP-client MVP:

- list local projects
- create a project
- open/switch a project
- search projects by name
- open a new empty request draft after project creation
- keep project data in local inspectable files

## Current Context

The application currently has a Bubble Tea shell with a left sidebar, request
tabs, a request editor, and a response pane. Project and request data are still
hardcoded in `internal/app/model.go`.

Existing docs already say that projects are MVP scope and local file storage is
preferred. Search is listed as a later idea in `docs/requirements.md`, but for
this feature pass project search is considered necessary because navigating
multiple projects without filtering will degrade quickly.

## Product Behavior

### Sidebar

The left sidebar becomes the project navigator.

At the top of the sidebar:

```text
Projects
<active project name or empty-state hint>
```

Do not show the `jire` title in this location. Do not show a separate
`project` label before the project name.

When at least one project exists, the second line shows the active project name.
Below that, the sidebar lists projects. The selected row is the project that
will be opened by `enter`.

When no projects exist, show a short empty state:

```text
Projects
Press c to create a project.
```

Keep the help text short and contextual. For the sidebar focus state, include:

```text
c create   s search   enter open
```

### Key Bindings

Use these keys for the project feature:

- `c`: create project
- `s`: search projects
- `enter`: confirm the highlighted project, create form, or search result
- `esc`: cancel create/search mode and return to the normal sidebar
- `j` / `down`: move selection down
- `k` / `up`: move selection up

`c` and `s` should be command keys only when the sidebar is focused, or when the
app is in an explicit project command mode. They must not be global keys while
the request editor is accepting text, otherwise typing a request body can
unexpectedly create or search projects.

### Create Project Flow

1. User focuses the sidebar.
2. User presses `c`.
3. Sidebar switches into a compact create mode with a single project-name input.
4. User types a non-empty name.
5. User presses `enter`.
6. The app creates the project on disk.
7. The new project becomes active.
8. The main area opens one empty request draft tab.
9. Focus moves to the request editor.

The empty request draft should use boring defaults:

- name: `Untitled request`
- method: `GET`
- URL: empty
- headers: empty
- body: empty

The project should be persisted immediately. The empty request draft can remain
an in-memory draft until request persistence is implemented, but it should carry
the active project ID so later save behavior has a clear owner.

Invalid project names should not panic. Show an inline error in the sidebar
create mode. At minimum, reject blank names after trimming whitespace.

### Open Project Flow

1. User focuses the sidebar.
2. User moves over the project list.
3. User presses `enter`.
4. The app loads that project's saved requests and recent request history.
5. The selected project becomes active.
6. Open tabs should be scoped to the active project.

For the first implementation, switching projects can close existing tabs from
the previous project. Later we can preserve per-project tab sessions if it feels
important.

### Search Flow

Project search starts when the user presses `s` while the sidebar is focused.

Search behavior:

- show a compact search input in the sidebar
- filter local projects by case-insensitive substring match on project name
- keep selection inside the filtered result set
- `enter` opens the highlighted result
- `esc` clears search mode and restores the full project list
- empty query shows all projects

This feature pass should search projects only. It should not search request
names, URLs, headers, bodies, response bodies, or history yet.

## Data Model

Add project models under `internal/project`.

Suggested model:

```go
type Project struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Slug      string    `json:"slug"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

Use a stable ID as the primary identity. Use the slug for directory names and
human-readable paths. Renaming a project later should not require changing its
ID.

Suggested request ownership field for future request persistence:

```go
type Request struct {
    ID        string `json:"id"`
    ProjectID string `json:"projectId"`
    // existing request fields...
}
```

## Persistence

Use the existing intended local layout:

```text
~/.config/jire/
  projects/
    <project-slug>/
      project.json
      requests/
      history.jsonl
```

For the first pass, `project.json` is enough to make project creation, listing,
and opening real. Create the `requests/` directory and empty `history.jsonl`
only if doing so simplifies later implementation; otherwise defer until those
features need them.

`project.json` example:

```json
{
  "id": "prj_01JZ...",
  "name": "Local API",
  "slug": "local-api",
  "createdAt": "2026-07-07T00:00:00Z",
  "updatedAt": "2026-07-07T00:00:00Z"
}
```

Slug rules:

- trim whitespace
- lowercase ASCII letters where possible
- replace runs of non-alphanumeric characters with `-`
- trim leading/trailing `-`
- fall back to `project` if the slug would be empty
- append a short suffix on collision

Project list ordering should default to `updatedAt` descending, then name
ascending for stable ties. Opening or creating a project should update
`updatedAt`.

## Store API

Keep persistence separate from Bubble Tea state. The TUI should call a small
store API instead of reading files directly.

Suggested API shape:

```go
type Store struct {
    Root string
}

func DefaultRoot() (string, error)
func NewStore(root string) Store

func (s Store) ListProjects(ctx context.Context) ([]Project, error)
func (s Store) CreateProject(ctx context.Context, name string) (Project, error)
func (s Store) LoadProject(ctx context.Context, id string) (Project, error)
func (s Store) TouchProject(ctx context.Context, id string) error
```

`Root` should usually point at the `jire` config directory, not directly at the
`projects` directory.

The store should create missing directories on write, but should not create fake
projects on read.

## TUI State

The root app model should move away from a single hardcoded `projectName`.

Suggested state additions:

```go
type projectMode int

const (
    projectModeNormal projectMode = iota
    projectModeCreate
    projectModeSearch
)

type Model struct {
    // existing fields...
    store project.Store

    projects []project.Project
    activeProject *project.Project
    projectRow int
    projectMode projectMode
    projectSearchQuery string
    projectCreateInput textinput.Model
    projectSearchInput textinput.Model
    projectError string
}
```

Exact field names can follow the implementation style of the codebase, but the
separation matters:

- store and project models live in `internal/project`
- Bubble Tea focus, key routing, and text inputs live in `internal/app`
- rendering helpers can move into `internal/ui` as the file grows

## Bubble Tea Messages

Project file work should run through commands and messages so the TUI remains
responsive and testable.

Suggested messages:

```go
type projectsLoadedMsg struct {
    Projects []project.Project
}

type projectCreatedMsg struct {
    Project project.Project
}

type projectOpenedMsg struct {
    Project project.Project
    Requests []project.Request
}

type projectStoreErrorMsg struct {
    Err error
}
```

If request persistence does not exist yet, `projectOpenedMsg` can omit
`Requests` or use the current in-memory request draft type. Do not force request
storage into this feature if that makes the change too broad.

## Implementation Sequence

1. Add `internal/project` models and store tests for create/list/load.
2. Add project loading to app startup.
3. Replace hardcoded `projectName` with active project state.
4. Render the sidebar as `Projects` plus active-project name or empty-state hint.
5. Add sidebar create mode bound to `c`.
6. After project creation, open a new empty request draft tab in the main area.
7. Add sidebar search mode bound to `s`.
8. Keep key routing scoped so text editor input still behaves normally.
9. Run `go fmt ./...` and `go test ./...`.

## Acceptance Criteria

- Fresh config directory: app starts without panic and shows `Projects` plus
  `Press c to create a project.`
- Pressing `c` in the sidebar opens a project-name input.
- Creating `Local API` writes a real `project.json` under the local config
  project directory.
- The created project becomes active immediately.
- The main area opens an empty request draft tab after project creation.
- Restarting the app lists the previously created project.
- Pressing `s` in the sidebar opens project search.
- Search filters project names case-insensitively.
- Pressing `enter` on a search result opens that project.
- Pressing `c` or `s` while typing in the request body inserts text normally
  instead of triggering project commands.
- Project storage errors appear in the UI instead of crashing the process.

## Non-Goals

Do not add these in the project feature pass:

- cloud sync
- accounts
- team workspaces
- project folders inside projects
- request folders
- global search across request bodies or response bodies
- project import/export
- project deletion
- project rename
- per-project environment variables

## Open Questions

- Should the first project be auto-created on first launch, or should the user
  explicitly press `c`? Current plan favors explicit creation.
- Should project search eventually include saved request names and URLs? Current
  plan keeps search project-only.
- Should switching projects preserve open tabs per project? Current plan closes
  previous project tabs for simplicity.
