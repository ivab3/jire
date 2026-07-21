package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivab3/jire/internal/httpclient"
	"github.com/ivab3/jire/internal/project"
	"github.com/ivab3/jire/internal/ui"
)

func (m Model) editorView(width int, height int) string {
	if !m.hasActiveTab() {
		return ""
	}

	tab := m.tabs[m.activeTab]
	contentWidth := ui.Max(20, width-4)
	keyWidth := ui.Max(10, (contentWidth-3)/2)
	valueWidth := ui.Max(10, contentWidth-keyWidth-3)

	var b strings.Builder
	fmt.Fprintln(&b, m.styles.Section.Render("Headers"))
	heading := m.styles.Muted.Render(
		lipgloss.NewStyle().Width(keyWidth).Render("KEY") + " │ " +
			lipgloss.NewStyle().Width(valueWidth).Render("VALUE"),
	)
	fmt.Fprintln(&b, heading)

	visibleHeaders := tab.Request.Headers
	maxHeaderRows := height - 9
	if maxHeaderRows < 0 {
		maxHeaderRows = 0
	}
	if maxHeaderRows > 3 {
		maxHeaderRows = 3
	}
	if len(visibleHeaders) > maxHeaderRows {
		visibleHeaders = visibleHeaders[len(visibleHeaders)-maxHeaderRows:]
	}
	for _, header := range visibleHeaders {
		key := truncateText(header.Name, keyWidth)
		value := truncateText(header.Value, valueWidth)
		row := lipgloss.NewStyle().Width(keyWidth).Render(key) + " │ " +
			lipgloss.NewStyle().Width(valueWidth).Render(value)
		fmt.Fprintln(&b, row)
	}
	if len(tab.Request.Headers) > len(visibleHeaders) {
		fmt.Fprintln(&b, m.styles.Muted.Render(fmt.Sprintf("… %d more", len(tab.Request.Headers)-len(visibleHeaders))))
	}

	inputRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(keyWidth).Render(m.headerNameInput.View()),
		" │ ",
		lipgloss.NewStyle().Width(valueWidth).Render(m.headerValueInput.View()),
	)
	fmt.Fprintln(&b, inputRow)
	if m.editorError != "" {
		fmt.Fprintln(&b, m.styles.Muted.Render(m.editorError))
	}

	fmt.Fprintln(&b, m.styles.Section.Render("Body"))
	b.WriteString(tab.Body.View())

	mode := "NORMAL"
	if tab.VimMode == vimInsert {
		mode = "INSERT"
	}
	modeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	if tab.VimMode == vimInsert {
		modeStyle = modeStyle.Background(lipgloss.Color("31")).Padding(0, 1)
	} else {
		modeStyle = modeStyle.Background(lipgloss.Color("236")).Padding(0, 1)
	}
	status := modeStyle.Render(mode)
	remaining := height - 2 - lipgloss.Height(b.String()) - 1
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}
	b.WriteString(lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Right).Render(status))

	return lipgloss.NewStyle().Width(contentWidth).Height(height - 2).Render(b.String())
}

func (m Model) responseView(width int, height int) string {
	title := m.styles.Section.Render("Response")
	if !m.hasActiveTab() {
		return title
	}
	tab := m.tabs[m.activeTab]
	if tab.ResponseTitle != "" {
		title += "  " + m.styles.Muted.Render(tab.ResponseTitle)
	}
	if strings.TrimSpace(tab.Response.View()) == "" {
		return title
	}
	content := title + "\n\n" + tab.Response.View()
	return lipgloss.NewStyle().Width(width - 4).Height(height - 2).Render(content)
}

func (m Model) helpView() string {
	help := "^J send   tab focus   [/] tabs   ^C quit"
	switch {
	case m.projectMode == projectModeCreate:
		help = "enter create project   esc cancel"
	case m.projectMode == projectModeSearch:
		help = "j/k move   enter open   esc cancel"
	case m.requestMode == requestModeCreate:
		help = "tab method/url   j/k choose method   enter confirm   esc cancel"
	case m.focus == focusSidebar && m.sidebarScreen == sidebarProjects:
		help = "c create project   s search   enter open   q quit"
	case m.focus == focusSidebar:
		help = "c new request   enter open   esc projects   ^J send"
	case m.focus == focusHeaderName:
		help = "header key   enter value   tab next   ^J send"
	case m.focus == focusHeaderValue:
		help = "header value   enter add   tab body   ^J send"
	case m.focus == focusBody && m.hasActiveTab() && m.tabs[m.activeTab].VimMode == vimNormal:
		help = "NORMAL   i/a insert   hjkl move   esc normal   ^J send"
	case m.focus == focusBody:
		help = "INSERT   esc normal   ^J send"
	case m.focus == focusResponse:
		help = "response   j/k scroll   tab focus   ^J send"
	}
	return m.styles.Help.Width(ui.Max(1, m.width)).Render(help)
}

func (m *Model) addHeader() {
	if !m.hasActiveTab() {
		return
	}
	name := strings.TrimSpace(m.headerNameInput.Value())
	if name == "" {
		m.editorError = "header key is required"
		m.setFocus(focusHeaderName)
		return
	}
	m.tabs[m.activeTab].Request.Headers = append(m.tabs[m.activeTab].Request.Headers, project.Header{
		Enabled: true,
		Name:    name,
		Value:   m.headerValueInput.Value(),
	})
	m.headerNameInput.SetValue("")
	m.headerValueInput.SetValue("")
	m.editorError = ""
	m.setFocus(focusHeaderName)
}

func (m Model) updateBody(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.hasActiveTab() {
		return m, nil
	}
	tab := &m.tabs[m.activeTab]
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if tab.VimMode == vimInsert {
			if key.String() == "esc" {
				tab.VimMode = vimNormal
				return m, nil
			}
			var cmd tea.Cmd
			tab.Body, cmd = tab.Body.Update(msg)
			tab.Request.Body = tab.Body.Value()
			return m, cmd
		}

		switch key.String() {
		case "i":
			tab.VimMode = vimInsert
			return m, tab.Body.Focus()
		case "a":
			line := tab.Body.LineInfo()
			tab.Body.SetCursorColumn(line.CharOffset + 1)
			tab.VimMode = vimInsert
			return m, tab.Body.Focus()
		case "I":
			tab.Body.CursorStart()
			tab.VimMode = vimInsert
			return m, tab.Body.Focus()
		case "A":
			tab.Body.CursorEnd()
			tab.VimMode = vimInsert
			return m, tab.Body.Focus()
		case "h", "left":
			line := tab.Body.LineInfo()
			tab.Body.SetCursorColumn(line.CharOffset - 1)
			return m, nil
		case "l", "right":
			line := tab.Body.LineInfo()
			tab.Body.SetCursorColumn(line.CharOffset + 1)
			return m, nil
		case "j", "down":
			tab.Body.CursorDown()
			return m, nil
		case "k", "up":
			tab.Body.CursorUp()
			return m, nil
		case "0", "home":
			tab.Body.CursorStart()
			return m, nil
		case "$", "end":
			tab.Body.CursorEnd()
			return m, nil
		case "x", "delete":
			deleteMsg := tea.KeyPressMsg(tea.Key{Code: tea.KeyDelete})
			var cmd tea.Cmd
			tab.Body, cmd = tab.Body.Update(deleteMsg)
			tab.Request.Body = tab.Body.Value()
			return m, cmd
		case "esc":
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	tab.Body, cmd = tab.Body.Update(msg)
	tab.Request.Body = tab.Body.Value()
	return m, cmd
}

func (m Model) canSendActiveRequest() bool {
	return m.hasActiveTab() && !m.tabs[m.activeTab].Loading
}

func (m *Model) prepareActiveRequestForSend() {
	if !m.hasActiveTab() {
		return
	}
	m.tabs[m.activeTab].Request.Body = m.tabs[m.activeTab].Body.Value()
}

func (m *Model) applyCompletedResponse(msg requestCompletedMsg) {
	for i := range m.tabs {
		if m.tabs[i].Request.ID != msg.RequestID {
			continue
		}
		m.tabs[i].Loading = false
		m.tabs[i].ResponseError = ""
		m.tabs[i].ResponseTitle = fmt.Sprintf("%s  %s  %s", msg.Response.Status, formatDuration(msg.Response.Duration), formatBytes(msg.Response.Size))
		m.tabs[i].Response.SetContent(formatResponse(msg.Response))
		m.tabs[i].Response.GotoTop()
		m.refreshTabRequest(i, msg.Requests)
		return
	}
}

func (m *Model) applyFailedResponse(msg requestFailedMsg) {
	for i := range m.tabs {
		if m.tabs[i].Request.ID != msg.RequestID {
			continue
		}
		m.tabs[i].Loading = false
		m.tabs[i].ResponseError = msg.Err.Error()
		m.tabs[i].ResponseTitle = "Request failed"
		m.tabs[i].Response.SetContent(msg.Err.Error())
		m.tabs[i].Response.GotoTop()
		m.refreshTabRequest(i, msg.Requests)
		return
	}
}

func (m *Model) refreshTabRequest(tabIndex int, requests []project.Request) {
	for _, request := range requests {
		if request.ID == m.tabs[tabIndex].Request.ID {
			m.tabs[tabIndex].Request = request
			return
		}
	}
}

func formatResponse(response httpclient.Response) string {
	var b strings.Builder
	keys := make([]string, 0, len(response.Headers))
	for key := range response.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range response.Headers.Values(key) {
			fmt.Fprintf(&b, "%s: %s\n", key, value)
		}
	}
	if len(keys) > 0 && len(response.Body) > 0 {
		b.WriteByte('\n')
	}

	body := response.Body
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		body = []byte(pretty.String())
	}
	b.Write(body)
	if response.Truncated {
		b.WriteString("\n\n[response truncated at 5 MiB]")
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatDuration(duration interface{ String() string }) string {
	value := duration.String()
	if strings.Contains(value, ".") {
		parts := strings.SplitN(value, ".", 2)
		if len(parts[0]) > 0 && strings.Contains(parts[1], "ms") {
			return parts[0] + "ms"
		}
	}
	return value
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(size)/(1024*1024))
}
