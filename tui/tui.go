package tui

import (
	"fmt"
	"strings"

	"github.com/bdmorin/gristle/gristapi"
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
	ViewTableActions
	ViewDocAccess
	ViewConfirmDelete
)

// DocAction represents an action that can be performed on a document
type DocAction int

const (
	ActionViewTables DocAction = iota
	ActionExportExcel
	ActionExportGrist
	ActionViewAccess
	ActionDelete
)

var docActionLabels = []string{
	"View Tables",
	"Export as Excel (.xlsx)",
	"Export as Grist (.grist)",
	"View Access",
	"Delete Document",
}

// TableAction represents an action that can be performed on a table
type TableAction int

const (
	TableActionViewData TableAction = iota
	TableActionExportCSV
)

var tableActionLabels = []string{
	"View Data",
	"Export as CSV",
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

	// Table data
	tableColumns []gristapi.TableColumn
	tableData    map[string][]interface{} // column ID -> values
	tableRowIDs  []uint

	// Access data
	docAccess gristapi.EntityAccess

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

	// Scroll state for table data
	scrollX int
	scrollY int

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
type tableDataLoadedMsg struct {
	columns []gristapi.TableColumn
	data    map[string][]interface{}
	rowIDs  []uint
}
type docAccessLoadedMsg gristapi.EntityAccess
type docDeletedMsg struct{}
type csvExportedMsg string
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

func loadTableData(docID, tableID string) tea.Cmd {
	return func() tea.Msg {
		columns := gristapi.GetTableColumns(docID, tableID)
		rows := gristapi.GetTableRows(docID, tableID)

		// Fetch actual data using the records endpoint
		data := make(map[string][]interface{})
		// For now, we'll use the row IDs and column info
		// The actual data would need a GetTableRecords function
		return tableDataLoadedMsg{
			columns: columns.Columns,
			data:    data,
			rowIDs:  rows.Id,
		}
	}
}

func loadDocAccess(docID string) tea.Cmd {
	return func() tea.Msg {
		access := gristapi.GetDocAccess(docID)
		return docAccessLoadedMsg(access)
	}
}

func deleteDoc(docID string) tea.Cmd {
	return func() tea.Msg {
		gristapi.DeleteDoc(docID)
		return docDeletedMsg{}
	}
}

func exportTableCSV(docID, tableID, filename string) tea.Cmd {
	return func() tea.Msg {
		gristapi.GetTableContent(docID, tableID)
		return csvExportedMsg(fmt.Sprintf("Exported %s to CSV", tableID))
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

	case tableDataLoadedMsg:
		m.loading = false
		m.tableColumns = msg.columns
		m.tableData = msg.data
		m.tableRowIDs = msg.rowIDs
		m.scrollX = 0
		m.scrollY = 0

	case docAccessLoadedMsg:
		m.loading = false
		m.docAccess = gristapi.EntityAccess(msg)
		m.updateAccessList()

	case docDeletedMsg:
		m.loading = false
		m.message = "Document deleted successfully"
		// Go back to docs list and refresh
		m.view = ViewDocs
		m.selectedDoc = nil
		m.breadcrumb = m.breadcrumb[:2]
		m.cursor = 0
		if m.selectedWorkspace != nil {
			return m, tea.Batch(m.spinner.Tick, loadDocs(m.selectedWorkspace.Id))
		}

	case csvExportedMsg:
		m.loading = false
		m.message = string(msg)

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
		m.view = ViewTableActions
		m.cursor = 0
		m.updateTableActionsList()

	case ViewTableActions:
		return m.handleTableAction(TableAction(m.cursor))

	case ViewConfirmDelete:
		// Yes/No confirmation - cursor 0 = Yes, cursor 1 = No
		if m.cursor == 0 && m.selectedDoc != nil {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, deleteDoc(m.selectedDoc.Id))
		}
		// Cancel - go back to doc actions
		m.view = ViewDocActions
		m.cursor = 0
		m.updateActionsList()
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
		m.view = ViewDocAccess
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadDocAccess(docID))

	case ActionDelete:
		m.view = ViewConfirmDelete
		m.cursor = 1 // Default to "No" for safety
		m.items = []string{"Yes, delete this document", "No, cancel"}
	}

	return m, nil
}

// handleTableAction executes the selected table action
func (m Model) handleTableAction(action TableAction) (tea.Model, tea.Cmd) {
	if m.selectedDoc == nil || m.selectedTable == nil {
		return m, nil
	}

	docID := m.selectedDoc.Id
	tableID := m.selectedTable.Id

	switch action {
	case TableActionViewData:
		m.view = ViewTableData
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadTableData(docID, tableID))

	case TableActionExportCSV:
		filename := sanitizeFilename(tableID) + ".csv"
		m.loading = true
		m.message = "Exporting CSV..."
		return m, tea.Batch(m.spinner.Tick, exportTableCSV(docID, tableID, filename))
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

	case ViewTableActions:
		m.view = ViewTables
		m.selectedTable = nil
		m.breadcrumb = m.breadcrumb[:3]
		m.cursor = 0
		m.updateTablesList()

	case ViewTableData:
		m.view = ViewTableActions
		m.breadcrumb = m.breadcrumb[:4]
		m.cursor = 0
		m.updateTableActionsList()

	case ViewDocAccess:
		m.view = ViewDocActions
		m.breadcrumb = m.breadcrumb[:3]
		m.cursor = 0
		m.updateActionsList()

	case ViewConfirmDelete:
		m.view = ViewDocActions
		m.cursor = 0
		m.updateActionsList()
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

func (m *Model) updateTableActionsList() {
	m.items = make([]string, len(tableActionLabels))
	copy(m.items, tableActionLabels)
}

func (m *Model) updateAccessList() {
	m.items = make([]string, len(m.docAccess.Users))
	for i, user := range m.docAccess.Users {
		access := user.Access
		if access == "" {
			access = user.ParentAccess
			if access != "" {
				access += " (inherited)"
			}
		}
		m.items[i] = fmt.Sprintf("%s <%s> - %s", user.Name, user.Email, access)
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
		title = "Document Actions"
	case ViewTables:
		title = "Tables"
	case ViewTableActions:
		title = "Table Actions"
	case ViewTableData:
		title = "Table Data"
	case ViewDocAccess:
		title = "Document Access"
	case ViewConfirmDelete:
		title = "Confirm Delete"
	}
	b.WriteString(TitleStyle.Render(title))
	b.WriteString("\n")

	// Special view for table data
	if m.view == ViewTableData && !m.loading {
		b.WriteString(m.renderTableData())
	} else if m.view == ViewConfirmDelete && !m.loading {
		// Show warning for delete confirmation
		if m.selectedDoc != nil {
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("Are you sure you want to delete '%s'?", m.selectedDoc.Name)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("This action cannot be undone."))
			b.WriteString("\n\n")
		}
		// Render Yes/No options
		for i, item := range m.items {
			cursor := "  "
			style := ItemStyle
			if i == m.cursor {
				cursor = CursorStyle.Render()
				style = SelectedItemStyle
			}
			b.WriteString(cursor + style.Render(item) + "\n")
		}
	} else if m.loading {
		// Loading state
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

// renderTableData renders the table data view
func (m Model) renderTableData() string {
	var b strings.Builder

	if len(m.tableColumns) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("No columns found"))
		b.WriteString("\n")
		return b.String()
	}

	// Show table info
	if m.selectedTable != nil {
		b.WriteString(fmt.Sprintf("Table: %s\n", m.selectedTable.Id))
	}
	b.WriteString(fmt.Sprintf("Columns: %d | Rows: %d\n\n", len(m.tableColumns), len(m.tableRowIDs)))

	// Render column headers
	headers := make([]string, len(m.tableColumns))
	for i, col := range m.tableColumns {
		headers[i] = TableHeaderStyle.Render(fmt.Sprintf(" %-15s ", col.Id))
	}
	b.WriteString(strings.Join(headers, "|"))
	b.WriteString("\n")

	// Separator line
	sep := strings.Repeat("-", len(m.tableColumns)*18)
	b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(sep))
	b.WriteString("\n")

	// Show row IDs (we don't have full data yet, but we can show row count)
	maxRows := 10
	if len(m.tableRowIDs) < maxRows {
		maxRows = len(m.tableRowIDs)
	}

	for i := 0; i < maxRows; i++ {
		rowID := m.tableRowIDs[i]
		// Show row ID in first "column" position
		b.WriteString(TableCellStyle.Render(fmt.Sprintf(" Row %-10d ", rowID)))
		for j := 1; j < len(m.tableColumns); j++ {
			b.WriteString(TableCellStyle.Render(fmt.Sprintf(" %-15s ", "-")))
		}
		b.WriteString("\n")
	}

	if len(m.tableRowIDs) > maxRows {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(
			fmt.Sprintf("\n... and %d more rows", len(m.tableRowIDs)-maxRows)))
		b.WriteString("\n")
	}

	return b.String()
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
