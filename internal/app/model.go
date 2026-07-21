package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivab3/jire/internal/httpclient"
	"github.com/ivab3/jire/internal/project"
	"github.com/ivab3/jire/internal/ui"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusHeaderName
	focusHeaderValue
	focusBody
	focusResponse
)

type sidebarScreen int

const (
	sidebarProjects sidebarScreen = iota
	sidebarRequests
)

type projectMode int

const (
	projectModeNormal projectMode = iota
	projectModeCreate
	projectModeSearch
)

type requestMode int

const (
	requestModeNormal requestMode = iota
	requestModeCreate
)

type requestCreateFocus int

const (
	requestCreateMethod requestCreateFocus = iota
	requestCreateURL
)

type vimMode int

const (
	vimNormal vimMode = iota
	vimInsert
)

var requestMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

type tabState struct {
	Request       project.Request
	Body          textarea.Model
	Response      viewport.Model
	ResponseTitle string
	ResponseError string
	Loading       bool
	VimMode       vimMode
}

type Model struct {
	width  int
	height int

	store  project.Store
	client httpclient.Client
	styles ui.Styles

	projects      []project.Project
	activeProject *project.Project
	projectRow    int
	projectMode   projectMode
	projectError  string

	sidebarScreen sidebarScreen
	requests      []project.Request
	requestRow    int
	requestMode   requestMode
	requestError  string

	projectCreateInput textinput.Model
	projectSearchInput textinput.Model
	requestURLInput    textinput.Model
	requestMethodIndex int
	requestMethodOpen  bool
	requestCreateFocus requestCreateFocus

	headerNameInput  textinput.Model
	headerValueInput textinput.Model
	editorError      string

	tabs      []tabState
	activeTab int
	focus     focusArea
}

type projectsLoadedMsg struct {
	Projects []project.Project
}

type projectCreatedMsg struct {
	Project  project.Project
	Projects []project.Project
}

type projectOpenedMsg struct {
	Project  project.Project
	Projects []project.Project
	Requests []project.Request
}

type requestCreatedMsg struct {
	Request  project.Request
	Requests []project.Request
	Projects []project.Project
}

type requestCompletedMsg struct {
	RequestID string
	Response  httpclient.Response
	Requests  []project.Request
	Projects  []project.Project
}

type requestFailedMsg struct {
	RequestID string
	Err       error
	Requests  []project.Request
	Projects  []project.Project
}

type projectStoreErrorMsg struct {
	Err error
}

func New() Model {
	root, err := project.DefaultRoot()
	m := newModel(project.NewStore(root), httpclient.New())
	if err != nil {
		m.store = project.Store{}
		m.projectError = err.Error()
	}
	return m
}

func newModel(store project.Store, client httpclient.Client) Model {
	return Model{
		store:              store,
		client:             client,
		styles:             ui.DefaultStyles(),
		focus:              focusSidebar,
		sidebarScreen:      sidebarProjects,
		projectCreateInput: newTextInput("Project name", "> "),
		projectSearchInput: newTextInput("Search projects", "> "),
		requestURLInput:    newTextInput("https://api.example.com/resource", ""),
		headerNameInput:    newTextInput("Header name", ""),
		headerValueInput:   newTextInput("Header value", ""),
		requestCreateFocus: requestCreateURL,
	}
}

func newTextInput(placeholder string, prompt string) textinput.Model {
	input := textinput.New()
	input.Prompt = prompt
	input.Placeholder = placeholder
	input.CharLimit = 2048
	return input
}

func newTab(request project.Request) tabState {
	body := textarea.New()
	body.Placeholder = "Raw request body"
	body.ShowLineNumbers = true
	body.Prompt = ""
	body.SetValue(request.Body)
	body.SetWidth(60)
	body.SetHeight(6)

	response := viewport.New()
	response.SoftWrap = true
	response.SetContent("")

	return tabState{
		Request:  request,
		Body:     body,
		Response: response,
		VimMode:  vimNormal,
	}
}

func emptyRequest(projectID string) project.Request {
	return project.Request{
		ID:        "draft_" + projectID,
		ProjectID: projectID,
		Name:      "Untitled request",
		Method:    "GET",
		Headers:   []project.Header{},
	}
}

func (m Model) Init() tea.Cmd {
	if m.store.Root == "" {
		return nil
	}
	return m.loadProjectsCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	case tea.MouseClickMsg:
		if m.handleMouseClick(msg) {
			return m, nil
		}
	case projectsLoadedMsg:
		m.projects = msg.Projects
		m.clampProjectRow()
	case projectCreatedMsg:
		m.projects = msg.Projects
		m.finishProjectMode()
		m.activateProject(msg.Project)
		m.sidebarScreen = sidebarRequests
		m.requests = nil
		m.openEmptyDraft(msg.Project.ID)
		m.setFocus(focusBody)
	case projectOpenedMsg:
		m.projects = msg.Projects
		m.finishProjectMode()
		m.activateProject(msg.Project)
		m.sidebarScreen = sidebarRequests
		m.requests = msg.Requests
		m.requestRow = 0
		m.tabs = nil
		m.activeTab = 0
		if len(msg.Requests) > 0 {
			m.openRequest(msg.Requests[0])
		} else {
			m.openEmptyDraft(msg.Project.ID)
		}
		m.setFocus(focusSidebar)
	case requestCreatedMsg:
		m.projects = msg.Projects
		m.requests = msg.Requests
		m.finishRequestMode()
		m.selectRequestRow(msg.Request.ID)
		m.openRequest(msg.Request)
		m.setFocus(focusHeaderName)
	case requestCompletedMsg:
		m.projects = msg.Projects
		m.requests = msg.Requests
		m.applyCompletedResponse(msg)
	case requestFailedMsg:
		m.projects = msg.Projects
		if msg.Requests != nil {
			m.requests = msg.Requests
		}
		m.applyFailedResponse(msg)
	case projectStoreErrorMsg:
		if m.requestMode == requestModeCreate {
			m.requestError = msg.Err.Error()
		} else {
			m.projectError = msg.Err.Error()
		}
	}

	if m.projectMode != projectModeNormal {
		return m.updateProjectMode(msg)
	}
	if m.requestMode != requestModeNormal {
		return m.updateRequestMode(msg)
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+j":
			if m.canSendActiveRequest() {
				m.prepareActiveRequestForSend()
				m.tabs[m.activeTab].Loading = true
				m.tabs[m.activeTab].ResponseError = ""
				m.tabs[m.activeTab].ResponseTitle = "Sending…"
				m.tabs[m.activeTab].Response.SetContent("")
				return m, m.sendActiveRequestCmd()
			}
		case "tab":
			m.focusNext()
			return m, nil
		case "shift+tab":
			m.focusPrevious()
			return m, nil
		case "esc":
			if m.focus == focusSidebar && m.sidebarScreen == sidebarRequests {
				m.showProjects()
				return m, nil
			}
			if m.focus == focusHeaderName || m.focus == focusHeaderValue {
				m.setFocus(focusBody)
				return m, nil
			}
		case "q":
			if m.focus == focusSidebar || m.focus == focusResponse {
				return m, tea.Quit
			}
		case "c":
			if m.focus == focusSidebar {
				if m.sidebarScreen == sidebarProjects {
					return m, m.enterProjectCreateMode()
				}
				return m, m.enterRequestCreateMode()
			}
		case "s":
			if m.focus == focusSidebar && m.sidebarScreen == sidebarProjects {
				return m, m.enterProjectSearchMode()
			}
		case "up", "k":
			if m.focus == focusSidebar {
				m.moveSidebarSelection(-1)
				return m, nil
			}
		case "down", "j":
			if m.focus == focusSidebar {
				m.moveSidebarSelection(1)
				return m, nil
			}
		case "enter":
			if m.focus == focusSidebar {
				if m.sidebarScreen == sidebarProjects {
					return m, m.openSelectedProjectCmd()
				}
				m.openSelectedRequest()
				return m, nil
			}
			if m.focus == focusHeaderName {
				m.setFocus(focusHeaderValue)
				return m, nil
			}
			if m.focus == focusHeaderValue {
				m.addHeader()
				return m, nil
			}
		case "[":
			if m.focus != focusHeaderName && m.focus != focusHeaderValue {
				m.previousTab()
				return m, nil
			}
		case "]":
			if m.focus != focusHeaderName && m.focus != focusHeaderValue {
				m.nextTab()
				return m, nil
			}
		}
	}

	if len(m.tabs) == 0 || m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return m, cmd
	}

	switch m.focus {
	case focusHeaderName:
		m.headerNameInput, cmd = m.headerNameInput.Update(msg)
		m.editorError = ""
	case focusHeaderValue:
		m.headerValueInput, cmd = m.headerValueInput.Update(msg)
		m.editorError = ""
	case focusBody:
		return m.updateBody(msg)
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
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	sidebarWidth := m.sidebarWidth()
	mainWidth := m.width - sidebarWidth
	tabsHeight := 2
	helpHeight := 1
	contentHeight := m.height - helpHeight
	editorHeight := ui.Clamp((contentHeight-tabsHeight)/2, 10, contentHeight-7)
	responseHeight := contentHeight - tabsHeight - editorHeight

	sidebar := m.styles.Panel(m.focus == focusSidebar).
		Width(sidebarWidth).
		Height(contentHeight).
		Render(m.sidebarView(sidebarWidth, contentHeight))

	tabs := m.styles.Tabs.
		Width(mainWidth).
		Height(tabsHeight).
		Render(m.tabsView(mainWidth))

	editor := m.styles.Panel(m.editorFocused()).
		Width(mainWidth).
		Height(editorHeight).
		Render(m.editorView(mainWidth, editorHeight))

	response := m.styles.Panel(m.focus == focusResponse).
		Width(mainWidth).
		Height(responseHeight).
		Render(m.responseView(mainWidth, responseHeight))

	main := lipgloss.JoinVertical(lipgloss.Left, tabs, editor, response)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	content := lipgloss.JoinVertical(lipgloss.Left, body, m.helpView())

	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "jire"
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *Model) resize() {
	sidebarWidth := m.sidebarWidth()
	mainWidth := m.width - sidebarWidth
	contentHeight := m.height - 1
	tabsHeight := 2
	editorHeight := ui.Clamp((contentHeight-tabsHeight)/2, 10, contentHeight-7)
	responseHeight := contentHeight - tabsHeight - editorHeight

	m.projectCreateInput.SetWidth(ui.Max(10, sidebarWidth-6))
	m.projectSearchInput.SetWidth(ui.Max(10, sidebarWidth-6))
	m.requestURLInput.SetWidth(ui.Max(10, sidebarWidth-14))
	headerInputWidth := ui.Max(10, (mainWidth-5)/2)
	m.headerNameInput.SetWidth(headerInputWidth)
	m.headerValueInput.SetWidth(headerInputWidth)

	if m.width <= 0 || m.height <= 0 {
		return
	}
	for i := range m.tabs {
		headerRows := len(m.tabs[i].Request.Headers)
		maxHeaderRows := editorHeight - 9
		if maxHeaderRows < 0 {
			maxHeaderRows = 0
		}
		if maxHeaderRows > 3 {
			maxHeaderRows = 3
		}
		if headerRows > maxHeaderRows {
			headerRows = maxHeaderRows
		}
		m.tabs[i].Body.SetWidth(ui.Max(20, mainWidth-4))
		m.tabs[i].Body.SetHeight(ui.Max(2, editorHeight-7-headerRows))
		m.tabs[i].Response.SetWidth(ui.Max(20, mainWidth-4))
		m.tabs[i].Response.SetHeight(ui.Max(3, responseHeight-4))
	}
}

func (m Model) sidebarWidth() int {
	if m.width <= 0 {
		return 28
	}
	return ui.Clamp(28, 24, m.width/3)
}

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) bool {
	if msg.Button != tea.MouseLeft || m.sidebarScreen != sidebarRequests {
		return false
	}
	if msg.X <= m.sidebarWidth()+2 && msg.Y >= 1 && msg.Y <= 2 {
		m.showProjects()
		m.setFocus(focusSidebar)
		return true
	}
	return false
}

func (m Model) editorFocused() bool {
	return m.focus == focusHeaderName || m.focus == focusHeaderValue || m.focus == focusBody
}

func (m *Model) focusNext() {
	order := []focusArea{focusSidebar, focusHeaderName, focusHeaderValue, focusBody, focusResponse}
	for i, focus := range order {
		if focus == m.focus {
			m.setFocus(order[(i+1)%len(order)])
			return
		}
	}
	m.setFocus(focusSidebar)
}

func (m *Model) focusPrevious() {
	order := []focusArea{focusSidebar, focusHeaderName, focusHeaderValue, focusBody, focusResponse}
	for i, focus := range order {
		if focus == m.focus {
			m.setFocus(order[(i-1+len(order))%len(order)])
			return
		}
	}
	m.setFocus(focusSidebar)
}

func (m *Model) setFocus(next focusArea) {
	m.blurEditor()
	m.focus = next
	switch next {
	case focusHeaderName:
		m.headerNameInput.Focus()
	case focusHeaderValue:
		m.headerValueInput.Focus()
	case focusBody:
		if m.hasActiveTab() {
			m.tabs[m.activeTab].Body.Focus()
		}
	}
}

func (m *Model) blurEditor() {
	m.projectCreateInput.Blur()
	m.projectSearchInput.Blur()
	m.requestURLInput.Blur()
	m.headerNameInput.Blur()
	m.headerValueInput.Blur()
	if m.hasActiveTab() {
		m.tabs[m.activeTab].Body.Blur()
	}
}

func (m Model) hasActiveTab() bool {
	return len(m.tabs) > 0 && m.activeTab >= 0 && m.activeTab < len(m.tabs)
}

func (m *Model) openEmptyDraft(projectID string) {
	m.blurEditor()
	m.tabs = []tabState{newTab(emptyRequest(projectID))}
	m.activeTab = 0
	m.resize()
}

func (m *Model) openRequest(request project.Request) {
	for i := range m.tabs {
		if m.tabs[i].Request.ID == request.ID {
			m.activeTab = i
			m.tabs[i].Request = request
			m.tabs[i].Body.SetValue(request.Body)
			m.resize()
			return
		}
	}
	m.tabs = append(m.tabs, newTab(request))
	m.activeTab = len(m.tabs) - 1
	m.resize()
}

func (m *Model) previousTab() {
	if !m.hasActiveTab() {
		return
	}
	m.blurEditor()
	if m.activeTab == 0 {
		m.activeTab = len(m.tabs) - 1
	} else {
		m.activeTab--
	}
	m.setFocus(m.focus)
}

func (m *Model) nextTab() {
	if !m.hasActiveTab() {
		return
	}
	m.blurEditor()
	m.activeTab = (m.activeTab + 1) % len(m.tabs)
	m.setFocus(m.focus)
}

func (m Model) tabsView(width int) string {
	if len(m.tabs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		label := requestLabel(tab.Request)
		if len([]rune(label)) > 32 {
			label = string([]rune(label)[:29]) + "…"
		}
		label = fmt.Sprintf(" %s ", label)
		if i == m.activeTab {
			parts = append(parts, m.styles.ActiveTab.Render(label))
		} else {
			parts = append(parts, m.styles.InactiveTab.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func requestLabel(request project.Request) string {
	if strings.TrimSpace(request.URL) != "" {
		return request.URL
	}
	if strings.TrimSpace(request.Name) != "" {
		return request.Name
	}
	return "Untitled request"
}
