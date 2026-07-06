# Roadmap

This roadmap is intentionally small. `jire` should earn complexity only after
the base HTTP workflow feels good.

## Phase 0: Planning

- Write README.
- Write product requirements.
- Write technical design.
- Decide the first implementation stack.
- Initialize git and publish the repository.

## Phase 1: Application Shell

- Create Go module.
- Add Bubble Tea, Bubbles, and Lipgloss.
- Start a full-screen TUI.
- Render static sidebar, tabs, request editor, and response pane.
- Add basic key bindings for quitting, focus movement, and tab switching.

## Phase 2: In-Memory Requests

- Define request model.
- Add request list.
- Open requests as tabs.
- Edit method and URL.
- Edit headers.
- Edit raw body.
- Keep all state in memory.

## Phase 3: HTTP Execution

- Build requests with `net/http`.
- Send requests asynchronously from the TUI.
- Show loading state.
- Support cancellation.
- Display network errors.
- Show status, duration, size, headers, and body.

## Phase 4: Local Projects

- Define project file format.
- Save requests as JSON.
- Load project on startup.
- Add project creation.
- Add request duplication and deletion.

## Phase 5: History

- Store recent request entries in JSONL.
- Show recent requests in the sidebar.
- Reopen a recent request.
- Store response summary without response body by default.

## Phase 6: Usability Pass

- Add JSON pretty printing.
- Improve keyboard navigation.
- Add empty and error states.
- Add body size limits.
- Add tests for request construction and project storage.
- Prepare first tagged release.

## Later

- Environments and variables.
- Import/export curl.
- Request folders.
- Search.
- Secret masking.
- Cookie jar.
- Proxy and TLS controls.

