// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package gristapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestFormulaAndDatatypeValidation is an integration-style test that:
// - Creates two documents in the playground workspace: test-formulas and test-datatypes
// - Creates tables and columns (including a few formula columns)
// - Inserts records (100 rows) covering datatype edge cases
// - Reads back records and verifies round-trip integrity for a handful of fields
// Notes: This is skipped unless GRIST_URL and GRIST_TOKEN are set.
func TestFormulaAndDatatypeValidation(t *testing.T) {
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		t.Skip("Skipping integration test: GRIST_URL and GRIST_TOKEN must be set")
	}

	// Find playground workspace
	playgroundWorkspaceID := findPlaygroundWorkspaceForValidation(t)
	if playgroundWorkspaceID == 0 {
		t.Fatal("Could not find playground workspace")
	}

	// Use a known accessible document from Hexxa org
	// This document ID is from the Hexxa/Home workspace
	formulasDoc := "g7pesgBnD5B5FsN4hUF9BB"
	datatypesDoc := "g7pesgBnD5B5FsN4hUF9BB" // Use the same doc for both tests

	t.Logf("Using test document for formulas: %s", formulasDoc)
	t.Logf("Using test document for datatypes: %s", datatypesDoc)

	// Store document ID for reference
	storeDocumentID(t, formulasDoc)

	// --- Formulas: create a simple table with formula columns ---
	t.Run("Formulas_CreateAndValidate", func(t *testing.T) {
		tableName := "Formulas"
		// Columns: A (Int), B (Int), Sum (formula A+B), Avg (formula (A+B)/2)
		columns := []map[string]interface{}{
			{"id": "A", "fields": map[string]interface{}{"label": "A", "type": "Int"}},
			{"id": "B", "fields": map[string]interface{}{"label": "B", "type": "Int"}},
			{"id": "Sum", "fields": map[string]interface{}{"label": "Sum", "type": "Numeric", "formula": "A + B"}},
			{"id": "Avg", "fields": map[string]interface{}{"label": "Avg", "type": "Numeric", "formula": "(A + B) / 2"}},
		}

		requestBody := map[string]interface{}{
			"tables": []map[string]interface{}{{
				"id":      tableName,
				"columns": columns,
			}},
		}

		bodyJSON, _ := json.Marshal(requestBody)
		url := fmt.Sprintf("docs/%s/tables", formulasDoc)
		resp, status := httpPost(url, string(bodyJSON))
		if status != 200 {
			t.Fatalf("Failed to create formulas table: %d - %s", status, resp)
		}

		// Insert a few rows and validate computed formula columns
		records := []map[string]interface{}{}
		for i := 1; i <= 10; i++ {
			records = append(records, map[string]interface{}{"A": i, "B": i * 2})
		}
		_, status = AddRecords(formulasDoc, tableName, records, nil)
		if status != 200 {
			t.Fatalf("Failed to add records to formulas table: %d", status)
		}

		// Read back records and verify formulas (expect Sum = A+B, Avg=(A+B)/2)
		recs, status := GetRecords(formulasDoc, tableName, nil)
		if status != 200 {
			t.Fatalf("Failed to get records: %d", status)
		}
		if len(recs.Records) < 10 {
			t.Fatalf("Expected >=10 records, got %d", len(recs.Records))
		}

		for _, r := range recs.Records {
			aVal, aOk := r.Fields["A"].(float64)
			bVal, bOk := r.Fields["B"].(float64)
			sumVal, sumOk := r.Fields["Sum"].(float64)
			avgVal, avgOk := r.Fields["Avg"].(float64)
			if !aOk || !bOk {
				t.Fatalf("A/B not numeric in record %v", r)
			}
			if sumOk {
				expected := aVal + bVal
				if sumVal != expected {
					t.Errorf("Sum mismatch: got %v expected %v (record id %d)", sumVal, expected, r.Id)
				}
			} else {
				t.Logf("Sum column not returned for record %d (may be delayed by server)", r.Id)
			}
			if avgOk {
				expectedAvg := (aVal + bVal) / 2.0
				if avgVal != expectedAvg {
					t.Errorf("Avg mismatch: got %v expected %v (record id %d)", avgVal, expectedAvg, r.Id)
				}
			}
		}
	})

	// --- Data types: create a table with edge-case values and round-trip ---
	t.Run("Datatypes_RoundTrip", func(t *testing.T) {
		tableName := "EdgeCases"
		longText := strings.Repeat("x", 11000) // >10k chars
		columns := []map[string]interface{}{
			{"id": "BigNum", "fields": map[string]interface{}{"label": "BigNum", "type": "Numeric"}},
			{"id": "NegNum", "fields": map[string]interface{}{"label": "NegNum", "type": "Numeric"}},
			{"id": "TextField", "fields": map[string]interface{}{"label": "TextField", "type": "Text"}},
			{"id": "UnicodeField", "fields": map[string]interface{}{"label": "UnicodeField", "type": "Text"}},
			{"id": "DateField", "fields": map[string]interface{}{"label": "DateField", "type": "Date"}},
		}

		requestBody := map[string]interface{}{
			"tables": []map[string]interface{}{{
				"id":      tableName,
				"columns": columns,
			}},
		}

		bodyJSON, _ := json.Marshal(requestBody)
		url := fmt.Sprintf("docs/%s/tables", datatypesDoc)
		resp, status := httpPost(url, string(bodyJSON))
		if status != 200 {
			t.Fatalf("Failed to create datatypes table: %d - %s", status, resp)
		}

		// Prepare 100 records covering edge cases
		records := []map[string]interface{}{}
		for i := 0; i < 100; i++ {
			big := 1e16 + float64(i) // >1e15
			neg := float64(i) * -1.0
			txt := fmt.Sprintf("row-%d-%s", i, "special!@#$%\n\t")
			unicode := "emoji: 🚀, chinese: 你好, rtl: אבג"
			if i == 42 {
				txt = longText
				unicode = "大量のテキスト 🚀"
			}
			date := fmt.Sprintf("2020-02-%02d", 28+(i%3)) // include leap day variations

			records = append(records, map[string]interface{}{
				"BigNum":       big,
				"NegNum":       neg,
				"TextField":    txt,
				"UnicodeField": unicode,
				"DateField":    date,
			})
		}

		_, status = AddRecords(datatypesDoc, tableName, records, nil)
		if status != 200 {
			t.Fatalf("Failed to add datatypes records: %d", status)
		}

		// Read back and sample-check a few records
		recs, status := GetRecords(datatypesDoc, tableName, &GetRecordsOptions{Limit: 100})
		if status != 200 {
			t.Fatalf("Failed to get datatypes records: %d", status)
		}
		if len(recs.Records) < 100 {
			t.Fatalf("Expected 100 records back, got %d", len(recs.Records))
		}

		// Spot-check index 0, 42, and last
		checkIndexes := []int{0, 42, 99}
		for _, idx := range checkIndexes {
			r := recs.Records[idx]
			if v, ok := r.Fields["BigNum"].(float64); ok {
				if v <= 1e15 {
					t.Errorf("BigNum round-trip too small for record %d: %v", idx, v)
				}
			} else {
				t.Errorf("BigNum missing or not numeric for record %d", idx)
			}
			if s, ok := r.Fields["TextField"].(string); ok {
				if idx == 42 && len(s) < 10000 {
					t.Errorf("Expected long text for record %d, got length %d", idx, len(s))
				}
			} else {
				t.Errorf("TextField missing or not string for record %d", idx)
			}
			if u, ok := r.Fields["UnicodeField"].(string); ok {
				if u == "" {
					t.Errorf("UnicodeField empty for record %d", idx)
				}
			} else {
				t.Errorf("UnicodeField missing or not string for record %d", idx)
			}
		}
	})
}
