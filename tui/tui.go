package tui

import (
	"fmt"
	"strings"

	"github.com/Ville-Eurometropole-Strasbourg/grist-ctl/gristapi"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current screen
type View int

const (
	ViewOrgs View = iota
	ViewWorkspaces
	ViewDocs
	ViewDocActions
	ViewTables
	ViewTableData
)

// DocAction represents an action that can be performed on a document
type DocAction int

const (
	ActionViewTables DocAction = iota
	ActionExportCSV
	ActionExportExcel
	ActionExportGrist
	ActionViewAccess
	ActionDelete
)

var docActionLabels = []string{
	"View Tables",
	"Export as CSV",
	"Export as Excel (.xlsx)",
	"Export as Grist (.grist)",
	"View Access",
	"Delete Document",
}

// Model is the main application state
type Model struct {
	// Navigation
	view       View
	breadcrumb []string

	// Data
	orgs       []gristapi.Org
	workspaces []gristapi.Workspace
	docs       []gristapi.Doc
	tables     []gristapi.Table

	// Selection context
	selectedOrg       *gristapi.Org
	selectedWorkspace *gristapi.Workspace
	selectedDoc       *gristapi.Doc
	selectedTable     *gristapi.Table

	// List state
	cursor  int
	items   []string
	itemIDs []interface{} // stores the actual items for selection

	// UI state
	loading bool
	spinner spinner.Model
	err     error
	message string // success/info message

	// Keybindings
	keys KeyMap

	// Dimensions
	width, height int
}

// Messages
type orgsLoadedMsg []gristapi.Org
type workspacesLoadedMsg []gristapi.Workspace
type docsLoadedMsg struct {
	docs      []gristapi.Doc
	workspace gristapi.Workspace
}
type tablesLoadedMsg []gristapi.Table
type errMsg error
type successMsg string

// Commands
func loadOrgs() tea.Msg {
	orgs := gristapi.GetOrgs()
	return orgsLoadedMsg(orgs)
}

func loadWorkspaces(orgID int) tea.Cmd {
	return func() tea.Msg {
		workspaces := gristapi.GetOrgWorkspaces(orgID)
		return workspacesLoadedMsg(workspaces)
	}
}

func loadDocs(workspaceID int) tea.Cmd {
	return func() tea.Msg {
		workspace := gristapi.GetWorkspace(workspaceID)
		return docsLoadedMsg{docs: workspace.Docs, workspace: workspace}
	}
}

func loadTables(docID string) tea.Cmd {
	return func() tea.Msg {
		tables := gristapi.GetDocTables(docID)
		return tablesLoadedMsg(tables.Tables)
	}
}

func exportExcel(docID, filename string) tea.Cmd {
	return func() tea.Msg {
		gristapi.ExportDocExcel(docID, filename)
		return successMsg(fmt.Sprintf("Exported to %s", filename))
	}
}

func exportGrist(docID, filename string) tea.Cmd {
	return func() tea.Msg {
		gristapi.ExportDocGrist(docID, filename)
		return successMsg(fmt.Sprintf("Exported to %s", filename))
	}
}

// New creates a new TUI model
func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		view:    ViewOrgs,
		keys:    DefaultKeyMap(),
		spinner: s,
		loading: true,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadOrgs,
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear any message on keypress
		m.message = ""
		m.err = nil

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case key.Matches(msg, m.keys.Select):
			return m.handleSelect()

		case key.Matches(msg, m.keys.Back):
			return m.handleBack()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case orgsLoadedMsg:
		m.loading = false
		m.orgs = msg
		m.updateOrgsList()

	case workspacesLoadedMsg:
		m.loading = false
		m.workspaces = msg
		m.updateWorkspacesList()

	case docsLoadedMsg:
		m.loading = false
		m.docs = msg.docs
		// Update workspace info if we got more detail
		if m.selectedWorkspace != nil {
			ws := msg.workspace
			m.selectedWorkspace = &ws
		}
		m.updateDocsList()

	case tablesLoadedMsg:
		m.loading = false
		m.tables = msg
		m.updateTablesList()

	case successMsg:
		m.loading = false
		m.message = string(msg)

	case errMsg:
		m.loading = false
		m.err = msg
	}

	return m, nil
}

// handleSelect processes enter/select action
func (m Model) handleSelect() (tea.Model, tea.Cmd) {
	if len(m.items) == 0 || m.loading {
		return m, nil
	}

	switch m.view {
	case ViewOrgs:
		org := m.orgs[m.cursor]
		m.selectedOrg = &org
		m.breadcrumb = []string{org.Name}
		m.view = ViewWorkspaces
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadWorkspaces(org.Id))

	case ViewWorkspaces:
		ws := m.workspaces[m.cursor]
		m.selectedWorkspace = &ws
		m.breadcrumb = append(m.breadcrumb, ws.Name)
		m.view = ViewDocs
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadDocs(ws.Id))

	case ViewDocs:
		if len(m.docs) == 0 {
			return m, nil
		}
		doc := m.docs[m.cursor]
		m.selectedDoc = &doc
		m.breadcrumb = append(m.breadcrumb, doc.Name)
		m.view = ViewDocActions
		m.cursor = 0
		m.updateActionsList()

	case ViewDocActions:
		return m.handleDocAction(DocAction(m.cursor))

	case ViewTables:
		if len(m.tables) == 0 {
			return m, nil
		}
		table := m.tables[m.cursor]
		m.selectedTable = &table
		m.breadcrumb = append(m.breadcrumb, table.Id)
		m.view = ViewTableData
		// TODO: Load table data
		m.message = fmt.Sprintf("Table: %s (data view coming soon)", table.Id)
	}

	return m, nil
}

// handleDocAction executes the selected document action
func (m Model) handleDocAction(action DocAction) (tea.Model, tea.Cmd) {
	if m.selectedDoc == nil {
		return m, nil
	}

	docID := m.selectedDoc.Id
	docName := m.selectedDoc.Name

	switch action {
	case ActionViewTables:
		m.view = ViewTables
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadTables(docID))

	case ActionExportCSV:
		m.message = "CSV export: select a table first"
		m.view = ViewTables
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadTables(docID))

	case ActionExportExcel:
		filename := sanitizeFilename(docName) + ".xlsx"
		m.loading = true
		m.message = "Exporting..."
		return m, tea.Batch(m.spinner.Tick, exportExcel(docID, filename))

	case ActionExportGrist:
		filename := sanitizeFilename(docName) + ".grist"
		m.loading = true
		m.message = "Exporting..."
		return m, tea.Batch(m.spinner.Tick, exportGrist(docID, filename))

	case ActionViewAccess:
		// TODO: Show access list
		m.message = "Access view coming soon"

	case ActionDelete:
		// TODO: Confirm and delete
		m.message = "Delete requires confirmation (coming soon)"
	}

	return m, nil
}

// handleBack goes back one level
func (m Model) handleBack() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewOrgs:
		return m, tea.Quit

	case ViewWorkspaces:
		m.view = ViewOrgs
		m.selectedOrg = nil
		m.breadcrumb = nil
		m.cursor = 0
		m.updateOrgsList()

	case ViewDocs:
		m.view = ViewWorkspaces
		m.selectedWorkspace = nil
		m.breadcrumb = m.breadcrumb[:1]
		m.cursor = 0
		m.updateWorkspacesList()

	case ViewDocActions:
		m.view = ViewDocs
		m.selectedDoc = nil
		m.breadcrumb = m.breadcrumb[:2]
		m.cursor = 0
		m.updateDocsList()

	case ViewTables:
		m.view = ViewDocActions
		m.breadcrumb = m.breadcrumb[:3]
		m.cursor = 0
		m.updateActionsList()

	case ViewTableData:
		m.view = ViewTables
		m.selectedTable = nil
		m.breadcrumb = m.breadcrumb[:3]
		m.cursor = 0
		m.updateTablesList()
	}

	return m, nil
}

// Update item lists for each view
func (m *Model) updateOrgsList() {
	m.items = make([]string, len(m.orgs))
	for i, org := range m.orgs {
		m.items[i] = org.Name
	}
}

func (m *Model) updateWorkspacesList() {
	m.items = make([]string, len(m.workspaces))
	for i, ws := range m.workspaces {
		docCount := len(ws.Docs)
		m.items[i] = fmt.Sprintf("%s (%d docs)", ws.Name, docCount)
	}
}

func (m *Model) updateDocsList() {
	m.items = make([]string, len(m.docs))
	for i, doc := range m.docs {
		name := doc.Name
		if doc.IsPinned {
			name += " [pinned]"
		}
		m.items[i] = name
	}
}

func (m *Model) updateActionsList() {
	m.items = make([]string, len(docActionLabels))
	copy(m.items, docActionLabels)
}

func (m *Model) updateTablesList() {
	m.items = make([]string, len(m.tables))
	for i, t := range m.tables {
		m.items[i] = t.Id
	}
}

// View implements tea.Model
func (m Model) View() string {
	var b strings.Builder

	// Header with breadcrumb
	b.WriteString(RenderBreadcrumb(m.breadcrumb))
	b.WriteString("\n\n")

	// View title
	var title string
	switch m.view {
	case ViewOrgs:
		title = "Organizations"
	case ViewWorkspaces:
		title = "Workspaces"
	case ViewDocs:
		title = "Documents"
	case ViewDocActions:
		title = "Actions"
	case ViewTables:
		title = "Tables"
	case ViewTableData:
		title = "Table Data"
	}
	b.WriteString(TitleStyle.Render(title))
	b.WriteString("\n")

	// Loading state
	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...\n")
	} else if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	} else if len(m.items) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("(empty)"))
		b.WriteString("\n")
	} else {
		// Render list items
		for i, item := range m.items {
			cursor := "  "
			style := ItemStyle
			if i == m.cursor {
				cursor = CursorStyle.Render()
				style = SelectedItemStyle
			}
			b.WriteString(cursor + style.Render(item) + "\n")
		}
	}

	// Success/info message
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(SuccessStyle.Render(m.message))
		b.WriteString("\n")
	}

	// Footer with help
	b.WriteString("\n")
	help := []string{}
	help = append(help, HelpKeyStyle.Render("enter")+" select")
	if m.view != ViewOrgs {
		help = append(help, HelpKeyStyle.Render("esc")+" back")
	}
	help = append(help, HelpKeyStyle.Render("q")+" quit")
	b.WriteString(HelpStyle.Render(strings.Join(help, "  ")))

	return AppStyle.Render(b.String())
}

// sanitizeFilename makes a string safe for use as a filename
func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(s)
}

// Run starts the TUI
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
