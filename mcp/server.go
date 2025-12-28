package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bdmorin/gristle/gristapi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new MCP server for Grist operations
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"gristle",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	registerListOrgs(s)
	registerListWorkspaces(s)
	registerListDocs(s)
	registerGetDoc(s)
	registerExportDoc(s)
	registerGetDocTables(s)
	registerDeleteRecords(s)
	registerGetDocWebhooks(s)

	return s
}

// Run starts the MCP server on stdio
func Run() error {
	s := NewServer()
	return server.ServeStdio(s)
}

// registerListOrgs adds the list_orgs tool
func registerListOrgs(s *server.MCPServer) {
	tool := mcp.NewTool("list_orgs",
		mcp.WithDescription("List all organizations accessible to the user"),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgs := gristapi.GetOrgs()

		type orgInfo struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Domain string `json:"domain,omitempty"`
		}

		result := make([]orgInfo, len(orgs))
		for i, org := range orgs {
			result[i] = orgInfo{
				ID:     org.Id,
				Name:   org.Name,
				Domain: org.Domain,
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerListWorkspaces adds the list_workspaces tool
func registerListWorkspaces(s *server.MCPServer) {
	tool := mcp.NewTool("list_workspaces",
		mcp.WithDescription("List all workspaces in an organization"),
		mcp.WithNumber("org_id",
			mcp.Required(),
			mcp.Description("The organization ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("org_id")
		if err != nil {
			return mcp.NewToolResultError("org_id is required"), nil
		}

		workspaces := gristapi.GetOrgWorkspaces(orgID)

		type wsInfo struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			DocCount int    `json:"doc_count"`
		}

		result := make([]wsInfo, len(workspaces))
		for i, ws := range workspaces {
			result[i] = wsInfo{
				ID:       ws.Id,
				Name:     ws.Name,
				DocCount: len(ws.Docs),
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerListDocs adds the list_docs tool
func registerListDocs(s *server.MCPServer) {
	tool := mcp.NewTool("list_docs",
		mcp.WithDescription("List all documents in a workspace"),
		mcp.WithNumber("workspace_id",
			mcp.Required(),
			mcp.Description("The workspace ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wsID, err := req.RequireInt("workspace_id")
		if err != nil {
			return mcp.NewToolResultError("workspace_id is required"), nil
		}

		workspace := gristapi.GetWorkspace(wsID)

		type docInfo struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			IsPinned bool   `json:"is_pinned"`
		}

		result := make([]docInfo, len(workspace.Docs))
		for i, doc := range workspace.Docs {
			result[i] = docInfo{
				ID:       doc.Id,
				Name:     doc.Name,
				IsPinned: doc.IsPinned,
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerGetDoc adds the get_doc tool
func registerGetDoc(s *server.MCPServer) {
	tool := mcp.NewTool("get_doc",
		mcp.WithDescription("Get detailed information about a document"),
		mcp.WithString("doc_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := req.RequireString("doc_id")
		if err != nil {
			return mcp.NewToolResultError("doc_id is required"), nil
		}

		doc := gristapi.GetDoc(docID)
		tables := gristapi.GetDocTables(docID)

		type tableInfo struct {
			ID string `json:"id"`
		}

		type docDetail struct {
			ID        string      `json:"id"`
			Name      string      `json:"name"`
			IsPinned  bool        `json:"is_pinned"`
			Workspace string      `json:"workspace"`
			Org       string      `json:"org"`
			Tables    []tableInfo `json:"tables"`
		}

		tableList := make([]tableInfo, len(tables.Tables))
		for i, t := range tables.Tables {
			tableList[i] = tableInfo{ID: t.Id}
		}

		result := docDetail{
			ID:        doc.Id,
			Name:      doc.Name,
			IsPinned:  doc.IsPinned,
			Workspace: doc.Workspace.Name,
			Org:       doc.Workspace.Org.Name,
			Tables:    tableList,
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerExportDoc adds the export_doc tool
func registerExportDoc(s *server.MCPServer) {
	tool := mcp.NewTool("export_doc",
		mcp.WithDescription("Export a document to a file"),
		mcp.WithString("doc_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
		mcp.WithString("format",
			mcp.Required(),
			mcp.Description("Export format"),
			mcp.Enum("excel", "grist"),
		),
		mcp.WithString("filename",
			mcp.Description("Output filename (optional, defaults to document name)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := req.RequireString("doc_id")
		if err != nil {
			return mcp.NewToolResultError("doc_id is required"), nil
		}

		format, err := req.RequireString("format")
		if err != nil {
			return mcp.NewToolResultError("format is required"), nil
		}

		// Get doc name for default filename
		doc := gristapi.GetDoc(docID)
		filename := req.GetString("filename", doc.Name)

		switch format {
		case "excel":
			if filename[len(filename)-5:] != ".xlsx" {
				filename += ".xlsx"
			}
			gristapi.ExportDocExcel(docID, filename)
		case "grist":
			if filename[len(filename)-6:] != ".grist" {
				filename += ".grist"
			}
			gristapi.ExportDocGrist(docID, filename)
		default:
			return mcp.NewToolResultError("invalid format: " + format), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Document exported to %s", filename)), nil
	})
}

// registerGetDocTables adds the get_doc_tables tool
func registerGetDocTables(s *server.MCPServer) {
	tool := mcp.NewTool("get_doc_tables",
		mcp.WithDescription("Get the tables in a document with their columns"),
		mcp.WithString("doc_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := req.RequireString("doc_id")
		if err != nil {
			return mcp.NewToolResultError("doc_id is required"), nil
		}

		tables := gristapi.GetDocTables(docID)

		type colInfo struct {
			ID string `json:"id"`
		}

		type tableDetail struct {
			ID      string    `json:"id"`
			Columns []colInfo `json:"columns"`
		}

		result := make([]tableDetail, len(tables.Tables))
		for i, t := range tables.Tables {
			cols := gristapi.GetTableColumns(docID, t.Id)
			colList := make([]colInfo, len(cols.Columns))
			for j, c := range cols.Columns {
				colList[j] = colInfo{ID: c.Id}
			}
			result[i] = tableDetail{
				ID:      t.Id,
				Columns: colList,
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerDeleteRecords adds the delete_records tool
func registerDeleteRecords(s *server.MCPServer) {
	tool := mcp.NewTool("delete_records",
		mcp.WithDescription("Delete records from a table by their row IDs"),
		mcp.WithString("doc_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
		mcp.WithString("table_id",
			mcp.Required(),
			mcp.Description("The table ID"),
		),
		mcp.WithArray("row_ids",
			mcp.Required(),
			mcp.Description("Array of row IDs to delete"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := req.RequireString("doc_id")
		if err != nil {
			return mcp.NewToolResultError("doc_id is required"), nil
		}

		tableID, err := req.RequireString("table_id")
		if err != nil {
			return mcp.NewToolResultError("table_id is required"), nil
		}

		rowIDs, err := req.RequireIntSlice("row_ids")
		if err != nil {
			return mcp.NewToolResultError("row_ids must be an array of integers"), nil
		}

		if len(rowIDs) == 0 {
			return mcp.NewToolResultError("row_ids cannot be empty"), nil
		}

		_, status := gristapi.DeleteRecords(docID, tableID, rowIDs)

		if status == 200 {
			return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted %d record(s)", len(rowIDs))), nil
		}

		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete records, status code: %d", status)), nil
	})
}

// registerGetDocWebhooks adds the get_doc_webhooks tool
func registerGetDocWebhooks(s *server.MCPServer) {
	tool := mcp.NewTool("get_doc_webhooks",
		mcp.WithDescription("List all webhooks configured for a document"),
		mcp.WithString("doc_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := req.RequireString("doc_id")
		if err != nil {
			return mcp.NewToolResultError("doc_id is required"), nil
		}

		webhooks := gristapi.GetDocWebhooks(docID)

		type webhookInfo struct {
			ID         string   `json:"id"`
			Name       string   `json:"name"`
			Memo       string   `json:"memo,omitempty"`
			URL        string   `json:"url"`
			Enabled    bool     `json:"enabled"`
			EventTypes []string `json:"event_types"`
			TableID    string   `json:"table_id"`
			Status     string   `json:"status"`
			NumWaiting int      `json:"num_waiting"`
		}

		result := make([]webhookInfo, len(webhooks))
		for i, wh := range webhooks {
			result[i] = webhookInfo{
				ID:         wh.Id,
				Name:       wh.Fields.Name,
				Memo:       wh.Fields.Memo,
				URL:        wh.Fields.Url,
				Enabled:    wh.Fields.Enabled,
				EventTypes: wh.Fields.EventTypes,
				TableID:    wh.Fields.TableId,
				Status:     wh.Usage.Status,
				NumWaiting: wh.Usage.NumWaiting,
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}
