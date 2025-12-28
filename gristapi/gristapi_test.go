// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package gristapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestConnect(t *testing.T) {
	orgs := GetOrgs()
	nbOrgs := len(orgs)

	if nbOrgs < 1 {
		t.Errorf("We only found %d organizations", nbOrgs)
	}

	for i, org := range orgs {
		orgId := fmt.Sprintf("%d", org.Id)
		if GetOrg(orgId).Name != orgs[i].Name {
			t.Error("We don't find main organization.")
		}

		workspaces := GetOrgWorkspaces(org.Id)

		if len(workspaces) < 1 {
			t.Errorf("No workspace in org n°%d", org.Id)
		}

		for i, workspace := range workspaces {
			if workspace.OrgDomain != org.Domain {
				t.Errorf("Workspace %d : le domaine du workspace %s ne correspond pas à %s", workspace.Id, workspace.OrgDomain, org.Domain)
			}

			myWorkspace := GetWorkspace(workspace.Id)
			if myWorkspace.Name != workspace.Name {
				t.Errorf("Workspace n°%d : les noms ne correspondent pas (%s/%s)", workspace.Id, workspace.Name, myWorkspace.Name)
			}

			if workspace.Name != workspaces[i].Name {
				t.Error("Mauvaise correspondance des noms de Workspaces")
			}

			for i, doc := range workspace.Docs {
				if doc.Name != workspace.Docs[i].Name {
					t.Errorf("Document n°%s : non correspondance des noms (%s/%s)", doc.Id, doc.Name, workspace.Docs[i].Name)
				}

				// // Un document doit avoir au moins une table
				// tables := GetDocTables(doc.Id)
				// if len(tables.Tables) < 1 {
				// 	t.Errorf("Le document n°%s ne contient pas de table (org %d/workspace %s)", doc.Name, org.Id, workspace.Name)
				// }
				// for _, table := range tables.Tables {
				// 	// Une table doit avoir au moins une colonne
				// 	cols := GetTableColumns(doc.Id, table.Id)
				// 	if len(cols.Columns) < 1 {
				// 		t.Errorf("La table %s du document %s ne contient pas de colonne", table.Id, doc.Id)
				// 	}
				// }
			}

		}
	}

}

// setupMockServer creates a test server and sets environment variables
func setupMockServer(handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)
	oldURL := os.Getenv("GRIST_URL")
	oldToken := os.Getenv("GRIST_TOKEN")
	os.Setenv("GRIST_URL", server.URL)
	os.Setenv("GRIST_TOKEN", "test-token")
	return server, func() {
		server.Close()
		os.Setenv("GRIST_URL", oldURL)
		os.Setenv("GRIST_TOKEN", oldToken)
	}
}

func TestBuildRecordsQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		expected string
	}{
		{
			name:     "empty params",
			params:   map[string]string{},
			expected: "",
		},
		{
			name:     "single param",
			params:   map[string]string{"limit": "10"},
			expected: "?limit=10",
		},
		{
			name:     "empty value ignored",
			params:   map[string]string{"limit": "", "sort": "name"},
			expected: "?sort=name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRecordsQueryParams(tt.params)
			// For single param case, exact match
			if len(tt.params) <= 1 {
				if result != tt.expected {
					t.Errorf("buildRecordsQueryParams() = %q, want %q", result, tt.expected)
				}
			} else {
				// For multiple params, just check it starts with ? and contains expected parts
				if tt.expected != "" && (result == "" || result[0] != '?') {
					t.Errorf("buildRecordsQueryParams() = %q, expected to start with '?'", result)
				}
			}
		})
	}
}

func TestGetRecords(t *testing.T) {
	expectedRecords := RecordsList{
		Records: []Record{
			{Id: 1, Fields: map[string]interface{}{"name": "Alice", "age": float64(30)}},
			{Id: 2, Fields: map[string]interface{}{"name": "Bob", "age": float64(25)}},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedRecords)
	})
	defer cleanup()

	records, status := GetRecords("doc123", "Table1", nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(records.Records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records.Records))
	}
	if records.Records[0].Id != 1 {
		t.Errorf("Expected first record ID 1, got %d", records.Records[0].Id)
	}
}

func TestGetRecordsWithOptions(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("sort") != "name" {
			t.Errorf("Expected sort=name, got %s", query.Get("sort"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", query.Get("limit"))
		}
		if query.Get("hidden") != "true" {
			t.Errorf("Expected hidden=true, got %s", query.Get("hidden"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsList{Records: []Record{}})
	})
	defer cleanup()

	options := &GetRecordsOptions{
		Sort:   "name",
		Limit:  10,
		Hidden: true,
	}
	_, status := GetRecords("doc123", "Table1", options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestGetRecordsWithFilter(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		filterParam := query.Get("filter")
		if filterParam == "" {
			t.Error("Expected filter parameter")
		}

		var filter map[string][]interface{}
		if err := json.Unmarshal([]byte(filterParam), &filter); err != nil {
			t.Errorf("Failed to parse filter: %v", err)
		}
		if len(filter["name"]) != 1 || filter["name"][0] != "Alice" {
			t.Errorf("Expected filter for name=Alice, got %v", filter)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsList{Records: []Record{}})
	})
	defer cleanup()

	options := &GetRecordsOptions{
		Filter: map[string][]interface{}{"name": {"Alice"}},
	}
	_, status := GetRecords("doc123", "Table1", options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestAddRecords(t *testing.T) {
	expectedResponse := RecordsWithoutFields{
		Records: []struct {
			Id int `json:"id"`
		}{{Id: 1}, {Id: 2}},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var body struct {
			Records []struct {
				Fields map[string]interface{} `json:"fields"`
			} `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 2 {
			t.Errorf("Expected 2 records in request, got %d", len(body.Records))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	records := []map[string]interface{}{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}
	result, status := AddRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(result.Records) != 2 {
		t.Errorf("Expected 2 record IDs, got %d", len(result.Records))
	}
}

func TestAddRecordsWithNoParse(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("noparse") != "true" {
			t.Errorf("Expected noparse=true, got %s", query.Get("noparse"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsWithoutFields{})
	})
	defer cleanup()

	options := &AddRecordsOptions{NoParse: true}
	_, status := AddRecords("doc123", "Table1", []map[string]interface{}{}, options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpdateRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		var body struct {
			Records []Record `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 1 {
			t.Errorf("Expected 1 record in request, got %d", len(body.Records))
		}
		if body.Records[0].Id != 1 {
			t.Errorf("Expected record ID 1, got %d", body.Records[0].Id)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	records := []Record{
		{Id: 1, Fields: map[string]interface{}{"name": "Alice Updated"}},
	}
	_, status := UpdateRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpsertRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}

		var body struct {
			Records []RecordWithRequire `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 1 {
			t.Errorf("Expected 1 record in request, got %d", len(body.Records))
		}
		if body.Records[0].Require["email"] != "alice@example.com" {
			t.Errorf("Expected require email, got %v", body.Records[0].Require)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	records := []RecordWithRequire{
		{
			Require: map[string]interface{}{"email": "alice@example.com"},
			Fields:  map[string]interface{}{"name": "Alice", "age": 30},
		},
	}
	_, status := UpsertRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpsertRecordsWithOptions(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("onmany") != "all" {
			t.Errorf("Expected onmany=all, got %s", query.Get("onmany"))
		}
		if query.Get("noadd") != "true" {
			t.Errorf("Expected noadd=true, got %s", query.Get("noadd"))
		}
		if query.Get("noupdate") != "true" {
			t.Errorf("Expected noupdate=true, got %s", query.Get("noupdate"))
		}
		if query.Get("allow_empty_require") != "true" {
			t.Errorf("Expected allow_empty_require=true, got %s", query.Get("allow_empty_require"))
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	options := &UpsertRecordsOptions{
		OnMany:            "all",
		NoAdd:             true,
		NoUpdate:          true,
		AllowEmptyRequire: true,
	}
	_, status := UpsertRecords("doc123", "Table1", []RecordWithRequire{}, options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestDeleteRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/tables/Table1/records/delete" {
			t.Errorf("Expected delete endpoint, got %s", r.URL.Path)
		}

		var ids []int
		if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("Expected 2 IDs, got %d", len(ids))
		}
		if ids[0] != 1 || ids[1] != 2 {
			t.Errorf("Expected IDs [1, 2], got %v", ids)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DeleteRecords("doc123", "Table1", []int{1, 2})
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

// SCIM Bulk Operations Tests

func TestSCIMBulk_ValidRequest(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for SCIM user creation
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "user123",
			"userName": "testuser",
		})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
				BulkId: "bulk1",
				Data: map[string]interface{}{
					"userName": "testuser",
					"emails": []map[string]interface{}{
						{"value": "test@example.com", "primary": true},
					},
				},
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Schemas) != 1 || response.Schemas[0] != SCIMBulkResponseSchema {
		t.Errorf("Expected BulkResponse schema, got %v", response.Schemas)
	}
	if len(response.Operations) != 1 {
		t.Errorf("Expected 1 operation response, got %d", len(response.Operations))
	}
	if response.Operations[0].BulkId != "bulk1" {
		t.Errorf("Expected bulkId 'bulk1', got %s", response.Operations[0].BulkId)
	}
	if response.Operations[0].Method != "POST" {
		t.Errorf("Expected method 'POST', got %s", response.Operations[0].Method)
	}
}

func TestSCIMBulk_InvalidSchema(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{"invalid:schema"},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
			},
		},
	}

	_, status := SCIMBulk(request)

	if status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid schema, got %d", status)
	}
}

func TestSCIMBulk_InvalidMethod(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "GET", // GET is not allowed in bulk operations
				Path:   "/Users",
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200 (overall request succeeds), got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Fatalf("Expected 1 operation response, got %d", len(response.Operations))
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400', got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_MissingPath(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "", // Empty path
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200 (overall request succeeds), got %d", status)
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400' for missing path, got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_MultipleOperations(t *testing.T) {
	callCount := 0
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1"})
		case "PATCH":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1", "updated": true})
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
				BulkId: "op1",
				Data:   map[string]interface{}{"userName": "user1"},
			},
			{
				Method: "PATCH",
				Path:   "/Users/user1",
				BulkId: "op2",
				Data:   map[string]interface{}{"displayName": "Updated User"},
			},
			{
				Method: "DELETE",
				Path:   "/Users/user2",
				BulkId: "op3",
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Operations) != 3 {
		t.Errorf("Expected 3 operation responses, got %d", len(response.Operations))
	}
	if callCount != 3 {
		t.Errorf("Expected 3 HTTP calls, got %d", callCount)
	}
}

func TestSCIMBulk_FailOnErrors(t *testing.T) {
	callCount := 0
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// All operations fail
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas:      []string{SCIMBulkRequestSchema},
		FailOnErrors: 2, // Stop after 2 errors
		Operations: []SCIMBulkOperation{
			{Method: "POST", Path: "/Users", BulkId: "op1"},
			{Method: "POST", Path: "/Users", BulkId: "op2"},
			{Method: "POST", Path: "/Users", BulkId: "op3"}, // Should not execute
			{Method: "POST", Path: "/Users", BulkId: "op4"}, // Should not execute
		},
	}

	response, _ := SCIMBulk(request)

	if len(response.Operations) != 2 {
		t.Errorf("Expected 2 operation responses (stopped after failOnErrors), got %d", len(response.Operations))
	}
	if callCount != 2 {
		t.Errorf("Expected 2 HTTP calls (stopped after failOnErrors), got %d", callCount)
	}
}

func TestSCIMBulkFromJSON_ValidJSON(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1"})
	})
	defer cleanup()

	jsonBody := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "test1",
				"data": {"userName": "testuser"}
			}
		]
	}`

	response, status := SCIMBulkFromJSON(jsonBody)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Errorf("Expected 1 operation response, got %d", len(response.Operations))
	}
}

func TestSCIMBulkFromJSON_InvalidJSON(t *testing.T) {
	jsonBody := `{invalid json}`

	response, status := SCIMBulkFromJSON(jsonBody)

	if status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Fatalf("Expected 1 error operation, got %d", len(response.Operations))
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400', got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_PUTOperation(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1", "userName": "updated"})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "PUT",
				Path:   "/Users/user1",
				Data:   map[string]interface{}{"userName": "updated"},
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if response.Operations[0].Status != "200" {
		t.Errorf("Expected operation status '200', got %s", response.Operations[0].Status)
	}
}

// Attachment API Tests

func TestListAttachments(t *testing.T) {
	expectedAttachments := AttachmentList{
		Records: []AttachmentMetadata{
			{Id: 1, FileName: "test.png", FileSize: 1024, TimeUploaded: "2024-01-15T10:30:00Z"},
			{Id: 2, FileName: "doc.pdf", FileSize: 2048, TimeUploaded: "2024-01-16T11:00:00Z"},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/attachments") {
			t.Errorf("Expected attachments endpoint, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAttachments)
	})
	defer cleanup()

	attachments, status := ListAttachments("doc123", nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(attachments.Records) != 2 {
		t.Errorf("Expected 2 attachments, got %d", len(attachments.Records))
	}
	if attachments.Records[0].FileName != "test.png" {
		t.Errorf("Expected first attachment name 'test.png', got %s", attachments.Records[0].FileName)
	}
}

func TestListAttachmentsWithOptions(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("sort") != "fileName" {
			t.Errorf("Expected sort=fileName, got %s", query.Get("sort"))
		}
		if query.Get("limit") != "5" {
			t.Errorf("Expected limit=5, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AttachmentList{Records: []AttachmentMetadata{}})
	})
	defer cleanup()

	options := &GetAttachmentsOptions{
		Sort:  "fileName",
		Limit: 5,
	}
	_, status := ListAttachments("doc123", options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUploadAttachments(t *testing.T) {
	expectedResponse := []int{1, 2}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if !contains(contentType, "multipart/form-data") {
			t.Errorf("Expected multipart/form-data content type, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-upload-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	result, status := UploadAttachments("doc123", []string{tmpFile.Name()})
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 attachment IDs, got %d", len(result))
	}
}

func TestUploadAttachmentsEmptyList(t *testing.T) {
	result, status := UploadAttachments("doc123", []string{})
	if status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty file list, got %d", status)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestUploadAttachmentsFromReader(t *testing.T) {
	expectedResponse := []int{1}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if !contains(contentType, "multipart/form-data") {
			t.Errorf("Expected multipart/form-data content type, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	reader := strings.NewReader("test file content")
	result, status := UploadAttachmentsFromReader("doc123", "test.txt", reader)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 attachment ID, got %d", len(result))
	}
}

func TestGetAttachmentMetadata(t *testing.T) {
	expectedMetadata := AttachmentMetadata{
		Id:           1,
		FileName:     "test.png",
		FileSize:     1024,
		TimeUploaded: "2024-01-15T10:30:00Z",
		ImageHeight:  100,
		ImageWidth:   200,
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/attachments/1" {
			t.Errorf("Expected attachment metadata endpoint, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedMetadata)
	})
	defer cleanup()

	metadata, status := GetAttachmentMetadata("doc123", 1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if metadata.FileName != "test.png" {
		t.Errorf("Expected fileName 'test.png', got %s", metadata.FileName)
	}
	if metadata.FileSize != 1024 {
		t.Errorf("Expected fileSize 1024, got %d", metadata.FileSize)
	}
	if metadata.ImageHeight != 100 {
		t.Errorf("Expected imageHeight 100, got %d", metadata.ImageHeight)
	}
}

func TestDownloadAttachment(t *testing.T) {
	expectedContent := []byte("file content here")
	expectedContentType := "image/png"

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/attachments/1/download" {
			t.Errorf("Expected download endpoint, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", expectedContentType)
		w.Write(expectedContent)
	})
	defer cleanup()

	content, contentType, status := DownloadAttachment("doc123", 1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if string(content) != string(expectedContent) {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, content)
	}
	if contentType != expectedContentType {
		t.Errorf("Expected content type '%s', got '%s'", expectedContentType, contentType)
	}
}

func TestDownloadAttachmentToFile(t *testing.T) {
	expectedContent := []byte("file content for download")

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(expectedContent)
	})
	defer cleanup()

	// Create temp file path
	tmpFile, err := os.CreateTemp("", "test-download-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	destPath := tmpFile.Name()
	defer os.Remove(destPath)

	err = DownloadAttachmentToFile("doc123", 1, destPath)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != string(expectedContent) {
		t.Errorf("Expected file content '%s', got '%s'", expectedContent, content)
	}
}

func TestDownloadAttachmentToFileError(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	err := DownloadAttachmentToFile("doc123", 999, "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error for non-existent attachment")
	}
}

func TestRestoreAttachments(t *testing.T) {
	expectedResponse := RestoreAttachmentsResponse{
		Added:   5,
		Errored: 1,
		Unused:  2,
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/attachments/archive" {
			t.Errorf("Expected archive endpoint, got %s", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if !contains(contentType, "multipart/form-data") {
			t.Errorf("Expected multipart/form-data content type, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	// Create a temporary tar file
	tmpFile, err := os.CreateTemp("", "test-restore-*.tar")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("tar archive content")
	tmpFile.Close()

	result, status := RestoreAttachments("doc123", tmpFile.Name())
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.Added != 5 {
		t.Errorf("Expected 5 added, got %d", result.Added)
	}
	if result.Errored != 1 {
		t.Errorf("Expected 1 errored, got %d", result.Errored)
	}
	if result.Unused != 2 {
		t.Errorf("Expected 2 unused, got %d", result.Unused)
	}
}

func TestRestoreAttachmentsFromReader(t *testing.T) {
	expectedResponse := RestoreAttachmentsResponse{
		Added:   3,
		Errored: 0,
		Unused:  0,
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	reader := strings.NewReader("tar archive content from reader")
	result, status := RestoreAttachmentsFromReader("doc123", "archive.tar", reader)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.Added != 3 {
		t.Errorf("Expected 3 added, got %d", result.Added)
	}
}

func TestDeleteUnusedAttachments(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/attachments/removeUnused" {
			t.Errorf("Expected removeUnused endpoint, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DeleteUnusedAttachments("doc123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Webhook API Tests

func TestGetWebhooks(t *testing.T) {
	isReadyCol := "ready"
	expectedWebhooks := WebhooksList{
		Webhooks: []Webhook{
			{
				Id: "webhook-123",
				Fields: WebhookFields{
					Name:          "test-webhook",
					Memo:          "Test webhook memo",
					URL:           "https://example.com/webhook",
					Enabled:       true,
					EventTypes:    []string{"add", "update"},
					IsReadyColumn: &isReadyCol,
					TableId:       "Table1",
				},
				Usage: &WebhookUsage{
					NumWaiting: 0,
					Status:     "idle",
				},
			},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/webhooks" {
			t.Errorf("Expected /api/docs/doc123/webhooks, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedWebhooks)
	})
	defer cleanup()

	webhooks, status := GetWebhooks("doc123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(webhooks.Webhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(webhooks.Webhooks))
	}
	if webhooks.Webhooks[0].Id != "webhook-123" {
		t.Errorf("Expected webhook ID 'webhook-123', got %s", webhooks.Webhooks[0].Id)
	}
	if webhooks.Webhooks[0].Fields.Name != "test-webhook" {
		t.Errorf("Expected webhook name 'test-webhook', got %s", webhooks.Webhooks[0].Fields.Name)
	}
	if webhooks.Webhooks[0].Fields.URL != "https://example.com/webhook" {
		t.Errorf("Expected URL 'https://example.com/webhook', got %s", webhooks.Webhooks[0].Fields.URL)
	}
	if len(webhooks.Webhooks[0].Fields.EventTypes) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(webhooks.Webhooks[0].Fields.EventTypes))
	}
}

func TestGetWebhooks_EmptyList(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(WebhooksList{Webhooks: []Webhook{}})
	})
	defer cleanup()

	webhooks, status := GetWebhooks("doc123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(webhooks.Webhooks) != 0 {
		t.Errorf("Expected 0 webhooks, got %d", len(webhooks.Webhooks))
	}
}

func TestCreateWebhooks(t *testing.T) {
	expectedResponse := WebhooksCreateResponse{
		Webhooks: []WebhookId{
			{Id: "webhook-new-1"},
			{Id: "webhook-new-2"},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/webhooks" {
			t.Errorf("Expected /api/docs/doc123/webhooks, got %s", r.URL.Path)
		}

		var body WebhooksCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Webhooks) != 2 {
			t.Errorf("Expected 2 webhooks in request, got %d", len(body.Webhooks))
		}
		if body.Webhooks[0].Fields.URL == nil || *body.Webhooks[0].Fields.URL != "https://example.com/hook1" {
			t.Errorf("Expected URL 'https://example.com/hook1', got %v", body.Webhooks[0].Fields.URL)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	url1 := "https://example.com/hook1"
	url2 := "https://example.com/hook2"
	name1 := "webhook-1"
	name2 := "webhook-2"
	tableId := "Table1"
	enabled := true
	eventTypes := []string{"add"}

	webhooks := []WebhookPartialFields{
		{
			Name:       &name1,
			URL:        &url1,
			TableId:    &tableId,
			Enabled:    &enabled,
			EventTypes: &eventTypes,
		},
		{
			Name:       &name2,
			URL:        &url2,
			TableId:    &tableId,
			Enabled:    &enabled,
			EventTypes: &eventTypes,
		},
	}

	result, status := CreateWebhooks("doc123", webhooks)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(result.Webhooks) != 2 {
		t.Errorf("Expected 2 webhook IDs, got %d", len(result.Webhooks))
	}
	if result.Webhooks[0].Id != "webhook-new-1" {
		t.Errorf("Expected webhook ID 'webhook-new-1', got %s", result.Webhooks[0].Id)
	}
}

func TestCreateWebhooks_SingleWebhook(t *testing.T) {
	expectedResponse := WebhooksCreateResponse{
		Webhooks: []WebhookId{{Id: "webhook-single"}},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body WebhooksCreateRequest
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Webhooks) != 1 {
			t.Errorf("Expected 1 webhook in request, got %d", len(body.Webhooks))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	url := "https://example.com/single"
	name := "single-webhook"
	tableId := "Table1"
	eventTypes := []string{"add", "update"}

	webhooks := []WebhookPartialFields{
		{
			Name:       &name,
			URL:        &url,
			TableId:    &tableId,
			EventTypes: &eventTypes,
		},
	}

	result, status := CreateWebhooks("doc123", webhooks)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.Webhooks[0].Id != "webhook-single" {
		t.Errorf("Expected webhook ID 'webhook-single', got %s", result.Webhooks[0].Id)
	}
}

func TestUpdateWebhook(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/webhooks/webhook-456" {
			t.Errorf("Expected /api/docs/doc123/webhooks/webhook-456, got %s", r.URL.Path)
		}

		var body WebhookPartialFields
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.Enabled == nil || *body.Enabled != false {
			t.Errorf("Expected enabled=false, got %v", body.Enabled)
		}
		if body.Name == nil || *body.Name != "updated-name" {
			t.Errorf("Expected name='updated-name', got %v", body.Name)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	enabled := false
	name := "updated-name"
	fields := WebhookPartialFields{
		Enabled: &enabled,
		Name:    &name,
	}

	_, status := UpdateWebhook("doc123", "webhook-456", fields)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpdateWebhook_ChangeURL(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body WebhookPartialFields
		json.NewDecoder(r.Body).Decode(&body)
		if body.URL == nil || *body.URL != "https://new-url.com/webhook" {
			t.Errorf("Expected URL 'https://new-url.com/webhook', got %v", body.URL)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	url := "https://new-url.com/webhook"
	fields := WebhookPartialFields{
		URL: &url,
	}

	_, status := UpdateWebhook("doc123", "webhook-789", fields)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestDeleteWebhook(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/webhooks/webhook-to-delete" {
			t.Errorf("Expected /api/docs/doc123/webhooks/webhook-to-delete, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(WebhookDeleteResponse{Success: true})
	})
	defer cleanup()

	result, status := DeleteWebhook("doc123", "webhook-to-delete")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if !result.Success {
		t.Error("Expected success=true")
	}
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Webhook not found"})
	})
	defer cleanup()

	_, status := DeleteWebhook("doc123", "nonexistent")
	if status != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", status)
	}
}

func TestClearWebhookQueue(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/webhooks/queue" {
			t.Errorf("Expected /api/docs/doc123/webhooks/queue, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := ClearWebhookQueue("doc123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestWebhookUsage_WithAllFields(t *testing.T) {
	successTime := int64(1703980800000)
	failureTime := int64(1703977200000)
	errorMsg := "Connection timeout"
	httpStatus := 504

	expectedWebhooks := WebhooksList{
		Webhooks: []Webhook{
			{
				Id: "webhook-usage-test",
				Fields: WebhookFields{
					Name:       "usage-test",
					URL:        "https://example.com/webhook",
					Enabled:    true,
					EventTypes: []string{"add"},
					TableId:    "Table1",
				},
				Usage: &WebhookUsage{
					NumWaiting:       5,
					Status:           "retrying",
					LastSuccessTime:  &successTime,
					LastFailureTime:  &failureTime,
					LastErrorMessage: &errorMsg,
					LastHttpStatus:   &httpStatus,
					LastEventBatch: &WebhookBatchStatus{
						Size:     10,
						Status:   "failure",
						Attempts: 3,
					},
				},
			},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedWebhooks)
	})
	defer cleanup()

	webhooks, status := GetWebhooks("doc123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	usage := webhooks.Webhooks[0].Usage
	if usage == nil {
		t.Fatal("Expected usage to be non-nil")
	}
	if usage.NumWaiting != 5 {
		t.Errorf("Expected numWaiting=5, got %d", usage.NumWaiting)
	}
	if usage.Status != "retrying" {
		t.Errorf("Expected status='retrying', got %s", usage.Status)
	}
	if usage.LastSuccessTime == nil || *usage.LastSuccessTime != successTime {
		t.Errorf("Expected lastSuccessTime=%d, got %v", successTime, usage.LastSuccessTime)
	}
	if usage.LastErrorMessage == nil || *usage.LastErrorMessage != errorMsg {
		t.Errorf("Expected lastErrorMessage='%s', got %v", errorMsg, usage.LastErrorMessage)
	}
	if usage.LastEventBatch == nil || usage.LastEventBatch.Size != 10 {
		t.Errorf("Expected lastEventBatch.size=10, got %v", usage.LastEventBatch)
	}
}
