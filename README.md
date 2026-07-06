# jire

`jire` is a small terminal HTTP client for people who want to inspect APIs
without account prompts, workspace clutter, or a wall of product features.

The name comes from Haitian Creole and carries the "witness" idea: the tool
looks at what a server does and records what it saw.

This project is intentionally scoped to HTTP first. No GraphQL, gRPC,
WebSockets, cloud sync, team workspaces, or marketplace features in the initial
product.

## Product Shape

The first version should feel like a quiet, keyboard-first workbench:

```text
+ Projects / Recent ----+ GET /users  POST /login ----------------+
| my-api                | Method  URL                              |
|   GET /users          | Headers                                  |
|   POST /login         | Body                                     |
|                       +------------------------------------------+
|                       | Response: status, time, size, headers    |
|                       | body viewer                              |
+-----------------------+------------------------------------------+
```

Core workflow:

1. Create or open a project.
2. Pick a recent or saved request.
3. Open it as a tab.
4. Edit method, URL, headers, and body.
5. Send the request.
6. Inspect status, timing, headers, and body.

## Planned Stack

- Language: Go 1.25+
- TUI: Bubble Tea
- UI components: Bubbles
- Styling: Lipgloss
- HTTP: Go standard library `net/http`
- Storage: local JSON/JSONL files for the MVP

Go keeps distribution simple: one small binary, no runtime service, no browser,
no account system. Bubble Tea gives the app a state-driven TUI model that fits
tabs, panes, focused controls, and async HTTP requests well.

## MVP Scope

In scope:

- Local projects
- Saved requests per project
- Recent request history
- Request tabs
- Basic HTTP methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- URL editor
- Header editor
- Raw body editor
- Send/cancel request
- Response status, duration, size, headers, and body
- JSON pretty printing when safe to do so
- Plain text fallback for all response bodies
- Local persistence without accounts or sync

Out of scope for the first version:

- GraphQL
- gRPC
- WebSockets
- Cloud sync
- User accounts
- Team collaboration
- Plugin marketplace
- Pre-request scripting
- Visual GUI

## Proposed Local Data Layout

```text
~/.config/jire/
  projects/
    my-api/
      project.json
      requests/
        get-users.json
        create-user.json
      history.jsonl
```

The MVP should prefer boring, inspectable files over a database. SQLite can be
added later if search, tags, migrations, or large histories start to matter.

## Development

The Go module and first Bubble Tea application shell are scaffolded.

Useful commands:

```sh
go run ./cmd/jire
go test ./...
go fmt ./...
```

In sandboxed environments, point Go caches at writable directories:

```sh
GOCACHE=/private/tmp/jire-go-build GOMODCACHE=/private/tmp/jire-gomodcache go test ./...
```

## Design Principles

- Local-first by default.
- No account prompts.
- No telemetry unless explicitly added and documented later.
- HTTP-only until the HTTP workflow is genuinely good.
- Keyboard-first, but mouse support can be added if it does not complicate the
  core model.
- Keep the interface calm: projects, requests, tabs, editor, response.
- Make saved data easy to read, diff, back up, and delete.
