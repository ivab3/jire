# AGENTS.md

This file is for future Codex sessions working on `jire`.

## Project Intent

`jire` is a terminal HTTP client. It should be a calm alternative to large API
clients that push accounts, cloud workspaces, popups, and broad protocol
support before the basic request/response workflow feels good.

The name comes from Haitian Creole and maps to the product metaphor of a
"witness": the app sends a request, observes the server response, and keeps a
clear local record of what happened.

## Product Boundaries

Keep the initial product focused on HTTP:

- Projects
- Saved requests
- Recent requests
- Tabs for opened requests
- Method, URL, headers, and raw body editing
- Request execution
- Response status, duration, size, headers, and body inspection
- Local persistence

Do not expand the MVP into GraphQL, gRPC, WebSockets, cloud sync, accounts,
team collaboration, plugins, generated SDKs, or a GUI unless the user explicitly
asks for that direction.

## Preferred Stack

- Go for the application and HTTP engine
- Bubble Tea for the TUI runtime
- Bubbles for common input/view components
- Lipgloss for styling
- Go standard library `net/http` for HTTP requests
- JSON/JSONL files for MVP persistence

Use SQLite only when project requirements justify it. Local, inspectable files
are preferred for the first implementation.

## Repository Map

The intended implementation layout is:

```text
cmd/jire/              CLI entrypoint
internal/app/          Bubble Tea root model, update loop, commands
internal/httpclient/   HTTP request execution and response capture
internal/project/      project files, request models, persistence
internal/ui/           panes, styles, reusable TUI components
docs/                  product and technical notes
```

This layout is a target, not a mandate. Follow it unless the actual codebase
evolves into a clearer local pattern.

## Working Rules

- Read `README.md` and `docs/` before making product or architecture changes.
- Preserve the small-tool feel.
- Prefer simple, explicit state over clever abstractions.
- Keep request execution separate from TUI rendering.
- Keep persistence separate from TUI state.
- Use `go fmt ./...` before finishing Go edits.
- Use `go test ./...` when code exists and tests can run locally.
- Avoid storing secrets in logs or test fixtures.
- Be careful with generated files and user data under `~/.config/jire`.

## UX Notes

The main screen should remain recognizable:

- Left side: projects and recent/saved requests.
- Top of main area: open request tabs.
- Middle: focused request editor.
- Bottom or secondary pane: latest response.

Favor keyboard workflows. Keep visible help short and contextual. Avoid large
welcome screens, marketing copy, or explanatory panels inside the app.

## Current Status

The repository contains planning docs plus an initial Go module and Bubble Tea
application shell.

Current code anchors:

- `cmd/jire/main.go`: executable entrypoint.
- `internal/app/model.go`: root Bubble Tea model, focus handling, tabs, static
  request data, body textarea, and response viewport.
- `internal/ui/styles.go`: shared Lipgloss styles and small layout helpers.

HTTP execution and local persistence are not implemented yet.
