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
