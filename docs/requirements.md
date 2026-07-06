# Product Requirements

## Vision

`jire` is a local terminal HTTP client for fast, low-friction API inspection.
It should let a developer create a project, save a few requests, open them in
tabs, send them, and inspect the response without leaving the terminal.

The product should feel small on purpose. A good version of `jire` does less
than Postman, but makes the common HTTP workflow calmer and faster.

## Target User

Primary user:

- A developer working with local services, staging APIs, or small backend
  projects.
- Comfortable in a terminal.
- Wants saved requests and history, but does not want accounts, sync prompts,
  workspace onboarding, or a heavy GUI.

## Core Principles

- Local-first.
- HTTP-first.
- Keyboard-first.
- No account requirement.
- No cloud dependency.
- Files should be inspectable and portable.
- The UI should optimize for repeat work, not first-run marketing.

## MVP Functional Requirements

### Projects

- Create a project with a name.
- List local projects.
- Open a project.
- Store requests and history under the selected project.

### Requests

- Create a request.
- Edit request name.
- Edit HTTP method.
- Edit URL.
- Add, edit, disable, and remove headers.
- Edit a raw request body.
- Duplicate a request.
- Delete a request after confirmation.
- Open a request in a tab.
- Switch between open request tabs.

### Request Execution

- Support GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS.
- Build requests with custom headers.
- Send a raw body when the method and user input allow it.
- Use a configurable timeout.
- Show in-flight state.
- Allow cancellation of an in-flight request.
- Capture network errors and display them in the response pane.

### Response Inspection

- Show status code and reason.
- Show request duration.
- Show response size.
- Show response headers.
- Show response body.
- Pretty-print JSON responses when possible.
- Fall back to raw text for unknown content types.
- Make large responses viewable without freezing the TUI.

### History

- Save recent requests per project.
- Keep enough metadata to reopen a recent request.
- Record the latest response summary: status, duration, size, and timestamp.
- Do not store response bodies in history by default until the user explicitly
  chooses that behavior.

### Persistence

- Store project data locally.
- Prefer JSON for request definitions.
- Prefer JSONL for append-only history.
- Keep the file format simple enough to edit manually.

## MVP Non-Functional Requirements

- Run as a single local binary.
- Start quickly.
- Work offline except for user-triggered HTTP requests.
- Do not require login.
- Do not send telemetry.
- Handle narrow terminals gracefully.
- Avoid panics on malformed URLs, invalid headers, bad encodings, timeouts, and
  network failures.
- Keep request execution testable without running the TUI.

## Out of Scope

- GraphQL
- gRPC
- WebSockets
- Cloud sync
- Team workspaces
- User accounts
- OAuth helper flows
- Code generation
- Plugin marketplace
- Pre-request or post-response scripting
- Full browser-like cookie jar behavior
- Visual desktop GUI

## Later Ideas

These are explicitly not MVP commitments:

- Environments and variables
- Secret masking
- Cookie jar
- Import from curl
- Export to curl
- Import Postman collections
- Request folders
- Search
- Tags
- Response body persistence
- Diff previous responses
- TLS certificate controls
- Proxy settings

## Open Questions

- Should the first version support environments, or just plain URLs?
- Should history keep response bodies behind an opt-in setting?
- What is the desired default request timeout?
- Should Windows terminal support be a first-class target from day one?
- Should project files be designed for easy git commits, or optimized for local
  use only?

