# Technical Design

## Stack

`jire` should start as a Go application using Bubble Tea.

Recommended dependencies:

- `net/http` from the Go standard library for request execution.
- Bubble Tea for the TUI application model.
- Bubbles for text inputs, text areas, viewports, lists, and similar controls.
- Lipgloss for styling.

The main reason for this stack is operational simplicity: a small native binary
with a direct HTTP implementation and no browser, Node runtime, or background
service.

## Architecture

Target package layout:

```text
cmd/jire/
  main.go

internal/app/
  model.go
  update.go
  commands.go
  keys.go

internal/httpclient/
  client.go
  request.go
  response.go

internal/project/
  project.go
  request.go
  history.go
  store.go

internal/ui/
  layout.go
  styles.go
  sidebar.go
  tabs.go
  request_editor.go
  response_viewer.go
```

The package boundaries should stay boring:

- `internal/httpclient` knows how to execute HTTP requests.
- `internal/project` knows how to load and save project data.
- `internal/app` owns Bubble Tea state and wires async commands.
- `internal/ui` renders focused pieces of the TUI.

## TUI State Model

The root Bubble Tea model should hold:

- Current project.
- Saved request list.
- Recent request list.
- Open tabs.
- Active tab index.
- Focused pane/control.
- Current terminal size.
- In-flight request state.
- Latest response or latest error per tab.

HTTP execution should run as a Bubble Tea command and return a message such as
`requestCompletedMsg` or `requestFailedMsg`.

## HTTP Execution

The HTTP layer should:

- Reuse an `http.Client`.
- Set a default timeout.
- Validate method, URL, and headers before sending.
- Build `http.Request` values from internal request models.
- Read response bodies with a defensive size limit.
- Return response metadata separately from response body text.
- Avoid printing secrets or request bodies to logs.

Possible response model:

```go
type Response struct {
    StatusCode int
    Status     string
    Headers    map[string][]string
    Body       []byte
    Duration   time.Duration
    Size       int64
    ReceivedAt time.Time
}
```

The TUI can decide how to format the body. The HTTP package should not know
about terminal rendering.

## Persistence

MVP storage should be file-based:

```text
~/.config/jire/
  projects/
    <project-slug>/
      project.json
      requests/
        <request-id>.json
      history.jsonl
```

Possible request model:

```json
{
  "id": "req_01",
  "name": "Get users",
  "method": "GET",
  "url": "https://api.example.test/users",
  "headers": [
    {
      "enabled": true,
      "name": "Accept",
      "value": "application/json"
    }
  ],
  "body": {
    "mode": "raw",
    "text": ""
  },
  "updatedAt": "2026-07-06T00:00:00Z"
}
```

Use stable IDs for requests so renaming a request does not rename the file by
accident. Slugs can be used for display or optional export later.

## Rendering Guidelines

- Keep the main layout split into sidebar, tabs, editor, and response viewer.
- Avoid large onboarding screens.
- Keep help text short.
- Preserve content when terminal size changes.
- Ensure text inputs and text areas do not resize the whole layout unexpectedly.
- Show errors in place, close to the request or response pane.

## Testing Strategy

Start with tests around logic that is easy to isolate:

- Request model validation.
- Header validation.
- URL validation.
- HTTP request construction.
- Response body formatting.
- Project store read/write behavior.

Use `httptest` for HTTP execution tests. Avoid testing Bubble Tea rendering too
deeply until the UI structure stabilizes.

## Milestones

1. Go module and empty TUI shell.
2. Static layout with sidebar, tabs, editor, and response pane.
3. Request model and in-memory tab editing.
4. HTTP execution through `net/http`.
5. Response viewer with status, headers, and body.
6. Local project storage.
7. Recent request history.
8. Polish keyboard navigation and error states.

