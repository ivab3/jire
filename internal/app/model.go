package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivab3/jire/internal/ui"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusEditor
	focusResponse
)

type requestDraft struct {
	ID      string
	Name    string
	Method  string
	URL     string
	Headers []headerDraft
	Body    string
}

type headerDraft struct {
	Enabled bool
	Name    string
	Value   string
}

type tabState struct {
	Request  requestDraft
	Body     textarea.Model
	Response viewport.Model
}

type Model struct {
	width       int
	height      int
	projectName string
	requests    []requestDraft
	tabs        []tabState
	activeTab   int
	sidebarRow  int
	focus       focusArea
	styles      ui.Styles
}

func New() Model {
	requests := []requestDraft{
		{
			ID:     "req_local_health",
			Name:   "Local health",
			Method: "GET",
			URL:    "http://localhost:8080/health",
			Headers: []headerDraft{
				{Enabled: true, Name: "Accept", Value: "application/json"},
			},
		},
		{
			ID:     "req_create_user",
			Name:   "Create user",
			Method: "POST",
			URL:    "http://localhost:8080/users",
			Headers: []headerDraft{
				{Enabled: true, Name: "Content-Type", Value: "application/json"},
			},
			Body: "{\n  \"name\": \"Ada\"\n}",
		},
	}

	m := Model{
		projectName: "scratch",
		requests:    requests,
		focus:       focusEditor,
		styles:      ui.DefaultStyles(),
	}
	m.tabs = []tabState{newTab(requests[0]), newTab(requests[1])}
	m.tabs[0].Body.Focus()
	return m
}

func newTab(req requestDraft) tabState {
	body := textarea.New()
	body.Placeholder = "Raw request body"
	body.ShowLineNumbers = false
	body.Prompt = "  "
	body.SetValue(req.Body)
	body.SetWidth(60)
	body.SetHeight(6)

	resp := viewport.New()
	resp.SoftWrap = true
	resp.SetContent(responsePlaceholder(req))

	return tabState{
		Request:  req,
		Body:     body,
		Response: resp,
	}
}

func responsePlaceholder(req requestDraft) string {
	return fmt.Sprintf(
		"Not sent yet.\n\n%s %s\n\nResponse metadata, headers, and body will land here after HTTP execution is wired.",
		req.Method,
		req.URL,
	)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	handledGlobalKey := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focusNext()
			handledGlobalKey = true
		case "shift+tab":
			m.focusPrevious()
			handledGlobalKey = true
		case "up", "k":
			if m.focus == focusSidebar && m.sidebarRow > 0 {
				m.sidebarRow--
				handledGlobalKey = true
			}
		case "down", "j":
			if m.focus == focusSidebar && m.sidebarRow < len(m.requests)-1 {
				m.sidebarRow++
				handledGlobalKey = true
			}
		case "enter":
			if m.focus == focusSidebar {
				m.openSidebarRequest()
				handledGlobalKey = true
			}
		case "[":
			m.previousTab()
			handledGlobalKey = true
		case "]":
			m.nextTab()
			handledGlobalKey = true
		}
	}

	if handledGlobalKey {
		return m, cmd
	}

	if len(m.tabs) == 0 {
		return m, cmd
	}

	switch m.focus {
	case focusEditor:
		m.tabs[m.activeTab].Body, cmd = m.tabs[m.activeTab].Body.Update(msg)
		m.tabs[m.activeTab].Request.Body = m.tabs[m.activeTab].Body.Value()
	case focusResponse:
		m.tabs[m.activeTab].Response, cmd = m.tabs[m.activeTab].Response.Update(msg)
	}

	return m, cmd
}

func (m Model) View() tea.View {
	if m.width < 72 || m.height < 18 {
		v := tea.NewView(m.styles.SmallScreen.Render("jire needs a little more terminal space."))
		v.AltScreen = true
		v.WindowTitle = "jire"
		return v
	}

	sidebarWidth := ui.Clamp(24, 22, m.width/3)
	mainWidth := m.width - sidebarWidth - 1
	tabsHeight := 3
	helpHeight := 1
	contentHeight := m.height - helpHeight
	editorHeight := ui.Clamp((contentHeight-tabsHeight)/2, 9, contentHeight-8)
	responseHeight := contentHeight - tabsHeight - editorHeight

	sidebar := m.styles.Panel(focusSidebar == m.focus).
		Width(sidebarWidth).
		Height(contentHeight).
		Render(m.sidebarView(sidebarWidth, contentHeight))

	tabs := m.styles.Tabs.
		Width(mainWidth).
		Height(tabsHeight).
		Render(m.tabsView(mainWidth))

	editor := m.styles.Panel(focusEditor == m.focus).
		Width(mainWidth).
		Height(editorHeight).
		Render(m.editorView(mainWidth, editorHeight))

	response := m.styles.Panel(focusResponse == m.focus).
		Width(mainWidth).
		Height(responseHeight).
		Render(m.responseView(mainWidth, responseHeight))

	main := lipgloss.JoinVertical(lipgloss.Left, tabs, editor, response)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	content := lipgloss.JoinVertical(lipgloss.Left, body, m.helpView())

	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "jire"
	return v
}

func (m *Model) resize() {
	if m.width <= 0 || m.height <= 0 || len(m.tabs) == 0 {
		return
	}

	sidebarWidth := ui.Clamp(24, 22, m.width/3)
	mainWidth := m.width - sidebarWidth - 5
	contentHeight := m.height - 4
	editorHeight := ui.Clamp(contentHeight/2, 7, contentHeight-6)
	responseHeight := ui.Clamp(contentHeight-editorHeight, 5, contentHeight)

	for i := range m.tabs {
		m.tabs[i].Body.SetWidth(ui.Max(20, mainWidth-4))
		m.tabs[i].Body.SetHeight(ui.Max(4, editorHeight-8))
		m.tabs[i].Response.SetWidth(ui.Max(20, mainWidth-4))
		m.tabs[i].Response.SetHeight(ui.Max(3, responseHeight-4))
	}
}

func (m *Model) focusNext() {
	m.setFocus((m.focus + 1) % 3)
}

func (m *Model) focusPrevious() {
	if m.focus == focusSidebar {
		m.setFocus(focusResponse)
		return
	}
	m.setFocus(m.focus - 1)
}

func (m *Model) setFocus(next focusArea) {
	if len(m.tabs) > 0 {
		m.tabs[m.activeTab].Body.Blur()
	}
	m.focus = next
	if len(m.tabs) > 0 && m.focus == focusEditor {
		m.tabs[m.activeTab].Body.Focus()
	}
}

func (m *Model) openSidebarRequest() {
	if m.sidebarRow < 0 || m.sidebarRow >= len(m.requests) {
		return
	}

	req := m.requests[m.sidebarRow]
	for i := range m.tabs {
		if m.tabs[i].Request.ID == req.ID {
			m.activeTab = i
			m.setFocus(focusEditor)
			return
		}
	}

	m.tabs = append(m.tabs, newTab(req))
	m.activeTab = len(m.tabs) - 1
	m.setFocus(focusEditor)
	m.resize()
}

func (m *Model) previousTab() {
	if len(m.tabs) == 0 {
		return
	}
	m.tabs[m.activeTab].Body.Blur()
	if m.activeTab == 0 {
		m.activeTab = len(m.tabs) - 1
	} else {
		m.activeTab--
	}
	if m.focus == focusEditor {
		m.tabs[m.activeTab].Body.Focus()
	}
}

func (m *Model) nextTab() {
	if len(m.tabs) == 0 {
		return
	}
	m.tabs[m.activeTab].Body.Blur()
	m.activeTab = (m.activeTab + 1) % len(m.tabs)
	if m.focus == focusEditor {
		m.tabs[m.activeTab].Body.Focus()
	}
}

func (m Model) sidebarView(width int, height int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n\n", m.styles.Title.Render("jire"))
	fmt.Fprintf(&b, "%s %s\n\n", m.styles.Muted.Render("project"), m.projectName)
	fmt.Fprintln(&b, m.styles.Section.Render("Recent"))

	for i, req := range m.requests {
		prefix := "  "
		style := m.styles.ListItem
		if i == m.sidebarRow {
			prefix = "> "
			style = m.styles.SelectedListItem
		}
		line := fmt.Sprintf("%s%s %s", prefix, req.Method, req.Name)
		fmt.Fprintln(&b, style.Width(width-4).Render(line))
	}

	remaining := height - lipgloss.Height(b.String()) - 2
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	fmt.Fprintln(&b, m.styles.Muted.Render("enter open  tab focus"))
	return b.String()
}

func (m Model) tabsView(width int) string {
	if len(m.tabs) == 0 {
		return m.styles.Muted.Render("No open requests")
	}

	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		label := fmt.Sprintf(" %s ", tab.Request.Name)
		if i == m.activeTab {
			parts = append(parts, m.styles.ActiveTab.Render(label))
		} else {
			parts = append(parts, m.styles.InactiveTab.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) editorView(width int, height int) string {
	if len(m.tabs) == 0 {
		return m.styles.Muted.Render("Open a request to start editing.")
	}

	tab := m.tabs[m.activeTab]
	headerLines := []string{
		m.styles.Section.Render("Request"),
		fmt.Sprintf("%s  %s", m.styles.Method.Render(tab.Request.Method), tab.Request.URL),
		"",
		m.styles.Section.Render("Headers"),
	}

	for _, h := range tab.Request.Headers {
		marker := " "
		if h.Enabled {
			marker = "x"
		}
		headerLines = append(headerLines, fmt.Sprintf("[%s] %s: %s", marker, h.Name, h.Value))
	}

	headerLines = append(headerLines, "", m.styles.Section.Render("Body"))
	body := tab.Body.View()

	content := strings.Join(headerLines, "\n") + "\n" + body
	return lipgloss.NewStyle().Width(width - 4).Height(height - 2).Render(content)
}

func (m Model) responseView(width int, height int) string {
	if len(m.tabs) == 0 {
		return m.styles.Muted.Render("No response yet.")
	}

	title := m.styles.Section.Render("Response")
	body := m.tabs[m.activeTab].Response.View()
	content := title + "\n\n" + body
	return lipgloss.NewStyle().Width(width - 4).Height(height - 2).Render(content)
}

func (m Model) helpView() string {
	focus := map[focusArea]string{
		focusSidebar:  "sidebar",
		focusEditor:   "editor",
		focusResponse: "response",
	}[m.focus]

	return m.styles.Help.Width(m.width).Render(
		fmt.Sprintf("focus: %s   tab/shift+tab focus   [/]/ switch tabs   j/k move   q quit", focus),
	)
}
