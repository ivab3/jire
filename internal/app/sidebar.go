package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivab3/jire/internal/project"
	"github.com/ivab3/jire/internal/ui"
)

func (m Model) sidebarView(width int, height int) string {
	if m.sidebarScreen == sidebarRequests && m.activeProject != nil {
		return m.requestSidebarView(width, height)
	}
	return m.projectSidebarView(width, height)
}

func (m Model) projectSidebarView(width int, height int) string {
	var b strings.Builder
	fmt.Fprintln(&b, m.styles.Title.Render("Projects"))

	if m.projectMode == projectModeCreate {
		fmt.Fprintf(&b, "\n%s\n", m.styles.Section.Render("Create project"))
		fmt.Fprintln(&b, m.projectCreateInput.View())
		if m.projectError != "" {
			fmt.Fprintln(&b, m.styles.Muted.Render(m.projectError))
		}
		return m.sidebarWithHelp(b.String(), width, height, "enter create   esc cancel")
	}

	if m.projectMode == projectModeSearch {
		fmt.Fprintf(&b, "\n%s\n", m.styles.Section.Render("Search"))
		fmt.Fprintln(&b, m.projectSearchInput.View())
	}

	projects := m.visibleProjects()
	if len(projects) == 0 {
		if m.projectMode == projectModeSearch {
			fmt.Fprintln(&b, "\n"+m.styles.Muted.Render("No matching projects."))
		} else {
			fmt.Fprintln(&b, m.styles.Muted.Render("Press c to create a project."))
		}
	} else {
		fmt.Fprintln(&b)
		for i, item := range projects {
			prefix := "  "
			style := m.styles.ListItem
			if i == m.projectRow {
				prefix = "> "
				style = m.styles.SelectedListItem
			}
			line := truncateText(prefix+item.Name, width-4)
			fmt.Fprintln(&b, style.Width(width-4).Render(line))
		}
	}

	if m.projectError != "" {
		fmt.Fprintln(&b, "\n"+m.styles.Muted.Render(m.projectError))
	}
	return m.sidebarWithHelp(b.String(), width, height, "c create   s search   enter open")
}

func (m Model) requestSidebarView(width int, height int) string {
	var b strings.Builder
	fmt.Fprintln(&b, m.styles.Title.Render("← Projects"))
	fmt.Fprintln(&b, m.activeProject.Name)

	if m.requestMode == requestModeCreate {
		fmt.Fprintf(&b, "\n%s\n", m.styles.Section.Render("New request"))
		method := requestMethods[m.requestMethodIndex] + " ▾"
		methodStyle := ui.MethodStyle(requestMethods[m.requestMethodIndex]).Padding(0, 1)
		if m.requestCreateFocus == requestCreateMethod {
			methodStyle = methodStyle.Background(lipgloss.Color("236"))
		}
		line := lipgloss.JoinHorizontal(lipgloss.Top, methodStyle.Render(method), " ", m.requestURLInput.View())
		fmt.Fprintln(&b, line)

		if m.requestMethodOpen {
			for i, candidate := range requestMethods {
				marker := "  "
				style := ui.MethodStyle(candidate)
				if i == m.requestMethodIndex {
					marker = "> "
					style = style.Background(lipgloss.Color("236"))
				}
				fmt.Fprintln(&b, style.Width(11).Render(marker+candidate))
			}
		}
		if m.requestError != "" {
			fmt.Fprintln(&b, m.styles.Muted.Render(m.requestError))
		}
		return m.sidebarWithHelp(b.String(), width, height, "tab field   enter confirm   esc cancel")
	}

	fmt.Fprintf(&b, "\n%s\n", m.styles.Section.Render("Recent"))
	if len(m.requests) == 0 {
		fmt.Fprintln(&b, m.styles.Muted.Render("No recent requests."))
	} else {
		for i, request := range m.requests {
			prefix := "  "
			rowStyle := m.styles.ListItem
			if i == m.requestRow {
				prefix = "> "
				rowStyle = m.styles.SelectedListItem
			}
			method := ui.MethodStyle(request.Method).Render(request.Method)
			available := width - lipgloss.Width(prefix) - lipgloss.Width(request.Method) - 5
			url := truncateText(request.URL, ui.Max(1, available))
			line := prefix + method + " " + url
			fmt.Fprintln(&b, rowStyle.Width(width-4).Render(line))
		}
	}
	if m.requestError != "" {
		fmt.Fprintln(&b, "\n"+m.styles.Muted.Render(m.requestError))
	}
	return m.sidebarWithHelp(b.String(), width, height, "c create   esc projects   enter open")
}

func (m Model) sidebarWithHelp(content string, width int, height int, help string) string {
	renderedHelp := m.styles.Muted.Width(width - 4).Render(help)
	remaining := height - 2 - lipgloss.Height(content) - lipgloss.Height(renderedHelp)
	if remaining > 0 {
		content += strings.Repeat("\n", remaining)
	}
	return content + renderedHelp
}

func (m *Model) enterProjectCreateMode() tea.Cmd {
	m.blurEditor()
	m.focus = focusSidebar
	m.projectMode = projectModeCreate
	m.projectError = ""
	m.projectSearchInput.Blur()
	return m.projectCreateInput.Focus()
}

func (m *Model) enterProjectSearchMode() tea.Cmd {
	m.blurEditor()
	m.focus = focusSidebar
	m.projectMode = projectModeSearch
	m.projectError = ""
	m.projectCreateInput.Blur()
	m.clampProjectRow()
	return m.projectSearchInput.Focus()
}

func (m *Model) finishProjectMode() {
	m.projectMode = projectModeNormal
	m.projectError = ""
	m.projectCreateInput.SetValue("")
	m.projectSearchInput.SetValue("")
	m.projectCreateInput.Blur()
	m.projectSearchInput.Blur()
	m.clampProjectRow()
}

func (m Model) updateProjectMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.finishProjectMode()
			return m, nil
		case "enter":
			if m.projectMode == projectModeCreate {
				name := strings.TrimSpace(m.projectCreateInput.Value())
				if name == "" {
					m.projectError = project.ErrProjectNameRequired.Error()
					return m, nil
				}
				return m, m.createProjectCmd(name)
			}
			return m, m.openSelectedProjectCmd()
		case "up", "k":
			if m.projectMode == projectModeSearch {
				m.moveProjectSelection(-1)
				return m, nil
			}
		case "down", "j":
			if m.projectMode == projectModeSearch {
				m.moveProjectSelection(1)
				return m, nil
			}
		}
	}

	if m.projectMode == projectModeCreate {
		m.projectCreateInput, cmd = m.projectCreateInput.Update(msg)
		m.projectError = ""
	} else {
		m.projectSearchInput, cmd = m.projectSearchInput.Update(msg)
		m.clampProjectRow()
	}
	return m, cmd
}

func (m *Model) enterRequestCreateMode() tea.Cmd {
	if m.activeProject == nil {
		return nil
	}
	m.blurEditor()
	m.focus = focusSidebar
	m.requestMode = requestModeCreate
	m.requestError = ""
	m.requestMethodIndex = 0
	m.requestMethodOpen = false
	m.requestCreateFocus = requestCreateURL
	m.requestURLInput.SetValue("")
	return m.requestURLInput.Focus()
}

func (m *Model) finishRequestMode() {
	m.requestMode = requestModeNormal
	m.requestError = ""
	m.requestMethodOpen = false
	m.requestURLInput.SetValue("")
	m.requestURLInput.Blur()
}

func (m Model) updateRequestMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.requestMethodOpen {
				m.requestMethodOpen = false
				return m, nil
			}
			m.finishRequestMode()
			return m, nil
		case "tab", "shift+tab":
			if m.requestCreateFocus == requestCreateURL {
				m.requestURLInput.Blur()
				m.requestCreateFocus = requestCreateMethod
			} else {
				m.requestCreateFocus = requestCreateURL
				m.requestMethodOpen = false
				cmd = m.requestURLInput.Focus()
			}
			return m, cmd
		case "enter":
			if m.requestCreateFocus == requestCreateMethod {
				if m.requestMethodOpen {
					m.requestMethodOpen = false
					m.requestCreateFocus = requestCreateURL
					return m, m.requestURLInput.Focus()
				}
				m.requestMethodOpen = true
				return m, nil
			}
			url := strings.TrimSpace(m.requestURLInput.Value())
			if url == "" {
				m.requestError = project.ErrRequestURLRequired.Error()
				return m, nil
			}
			return m, m.createRequestCmd(requestMethods[m.requestMethodIndex], url)
		case "up", "k", "left", "h":
			if m.requestCreateFocus == requestCreateMethod {
				m.requestMethodIndex = (m.requestMethodIndex - 1 + len(requestMethods)) % len(requestMethods)
				return m, nil
			}
		case "down", "j", "right", "l":
			if m.requestCreateFocus == requestCreateMethod {
				m.requestMethodIndex = (m.requestMethodIndex + 1) % len(requestMethods)
				return m, nil
			}
		}
	}
	if m.requestCreateFocus == requestCreateURL {
		m.requestURLInput, cmd = m.requestURLInput.Update(msg)
		m.requestError = ""
	}
	return m, cmd
}

func (m *Model) showProjects() {
	m.finishRequestMode()
	m.sidebarScreen = sidebarProjects
	if m.activeProject != nil {
		m.selectProjectRow(m.activeProject.ID)
	}
}

func (m *Model) activateProject(item project.Project) {
	m.activeProject = &item
	m.selectProjectRow(item.ID)
}

func (m Model) visibleProjects() []project.Project {
	if m.projectMode != projectModeSearch {
		return m.projects
	}
	query := strings.ToLower(strings.TrimSpace(m.projectSearchInput.Value()))
	if query == "" {
		return m.projects
	}
	filtered := make([]project.Project, 0, len(m.projects))
	for _, item := range m.projects {
		if strings.Contains(strings.ToLower(item.Name), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m *Model) moveSidebarSelection(delta int) {
	if m.sidebarScreen == sidebarProjects {
		m.moveProjectSelection(delta)
		return
	}
	if len(m.requests) == 0 {
		m.requestRow = 0
		return
	}
	m.requestRow += delta
	if m.requestRow < 0 {
		m.requestRow = 0
	}
	if m.requestRow >= len(m.requests) {
		m.requestRow = len(m.requests) - 1
	}
}

func (m *Model) moveProjectSelection(delta int) {
	projects := m.visibleProjects()
	if len(projects) == 0 {
		m.projectRow = 0
		return
	}
	m.projectRow += delta
	if m.projectRow < 0 {
		m.projectRow = 0
	}
	if m.projectRow >= len(projects) {
		m.projectRow = len(projects) - 1
	}
}

func (m *Model) clampProjectRow() {
	projects := m.visibleProjects()
	if len(projects) == 0 {
		m.projectRow = 0
		return
	}
	if m.projectRow < 0 {
		m.projectRow = 0
	}
	if m.projectRow >= len(projects) {
		m.projectRow = len(projects) - 1
	}
}

func (m *Model) selectProjectRow(id string) {
	for i, item := range m.projects {
		if item.ID == id {
			m.projectRow = i
			return
		}
	}
	m.clampProjectRow()
}

func (m *Model) selectRequestRow(id string) {
	for i, item := range m.requests {
		if item.ID == id {
			m.requestRow = i
			return
		}
	}
	if len(m.requests) == 0 {
		m.requestRow = 0
	}
}

func (m *Model) openSelectedRequest() {
	if m.requestRow < 0 || m.requestRow >= len(m.requests) {
		return
	}
	m.openRequest(m.requests[m.requestRow])
	m.setFocus(focusBody)
}

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}
