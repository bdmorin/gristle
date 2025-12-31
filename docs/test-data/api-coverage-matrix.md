# Grist API Coverage Matrix

This document tracks the coverage of Grist API endpoints in the `gristle` validation test suite.

## Summary

| Category | Endpoints Tested | Total Endpoints | Coverage |
|----------|------------------|-----------------|----------|
| **Tables & Columns** | 8 | 10 | 80% |
| **Records** | 12 | 15 | 80% |
| **Documents** | 3 | 8 | 38% |
| **Formulas & Data Types** | 4 | 10 | 40% |
| **Webhooks** | 0 | 6 | 0% |
| **Attachments** | 0 | 5 | 0% |
| **Users (SCIM)** | N/A | N/A | Excluded |
| **Organizations** | N/A | N/A | Excluded |
| **Workspaces** | N/A | N/A | Excluded |
| **Overall** | 27 | 54 | **50%** |

## Detailed Coverage

### Tables & Columns APIs

| Endpoint | Method | Status | Test File | Notes |
|----------|--------|--------|-----------|-------|
| `/docs/{docId}/tables` | GET | ✅ Tested | table_validation_test.go | List tables |
| `/docs/{docId}/tables` | POST | ✅ Tested | table_validation_test.go | Create table |
| `/docs/{docId}/tables/{tableId}` | DELETE | ⚠️ Partial | table_validation_test.go | Delete table |
| `/docs/{docId}/tables/{tableId}/columns` | GET | ✅ Tested | table_validation_test.go | List columns |
| `/docs/{docId}/tables/{tableId}/columns` | POST | ✅ Tested | table_validation_test.go | Add columns |
| `/docs/{docId}/tables/{tableId}/columns` | PATCH | ✅ Tested | table_validation_test.go | Update columns |
| `/docs/{docId}/tables/{tableId}/columns` | PUT | ⚠️ Partial | - | Update all columns |
| `/docs/{docId}/tables/{tableId}/columns/{colId}` | DELETE | ✅ Tested | table_validation_test.go | Delete column |
| `/docs/{docId}/tables/{tableId}/columns/{colId}` | PATCH | ✅ Tested | table_validation_test.go | Rename column |
| `/docs/{docId}/apply` | POST | ❌ Not Tested | - | Apply user actions |

#### Column Types Tested

All 11 Grist column types have been tested:

- ✅ Text
- ✅ Numeric
- ✅ Int
- ✅ Bool
- ✅ Date
- ✅ DateTime:UTC
- ✅ Choice
- ✅ ChoiceList
- ✅ Ref (Reference)
- ✅ RefList (Reference List)
- ✅ Attachments

### Records APIs

| Endpoint | Method | Status | Test File | Notes |
|----------|--------|--------|-----------|-------|
| `/docs/{docId}/tables/{tableId}/records` | GET | ✅ Tested | record_validation_test.go | Get records |
| `/docs/{docId}/tables/{tableId}/records` | POST | ✅ Tested | record_validation_test.go | Add records |
| `/docs/{docId}/tables/{tableId}/records` | PATCH | ✅ Tested | record_validation_test.go | Update records |
| `/docs/{docId}/tables/{tableId}/records` | PUT | ✅ Tested | record_validation_test.go | Upsert records |
| `/docs/{docId}/tables/{tableId}/records` | DELETE | ✅ Tested | record_validation_test.go | Delete records |
| `/docs/{docId}/tables/{tableId}/data` | GET | ⚠️ Partial | - | Bulk data download |
| **Filter Support** | - | ✅ Tested | record_validation_test.go | Filter by column values |
| **Sort Support** | - | ✅ Tested | record_validation_test.go | Sort ascending/descending |
| **Pagination** | - | ✅ Tested | record_validation_test.go | Limit/offset |
| **Bulk Operations** | - | ✅ Tested | record_validation_test.go | 100-300 records |
| **noParse Option** | - | ⚠️ Partial | record_validation_test.go | String literal insertion |
| **Upsert Options** | - | ✅ Tested | record_validation_test.go | onMany parameter |
| `/docs/{docId}/tables/{tableId}/data/delete` | POST | ⚠️ Partial | - | Bulk delete with filters |
| `/docs/{docId}/sql` | GET | ❌ Not Tested | - | SQL queries |
| `/docs/{docId}/sql` | POST | ❌ Not Tested | - | Parameterized SQL |

#### Edge Cases Tested

- ✅ Unicode (emoji, CJK, RTL scripts)
- ✅ Special characters (quotes, backslashes, newlines, tabs)
- ✅ HTML/JSON strings
- ✅ Null/empty values
- ✅ Large numbers (>1e15, MaxInt32, MinInt32)
- ✅ Large text (>10k characters)
- ✅ Multi-line text

### Documents APIs

| Endpoint | Method | Status | Test File | Notes |
|----------|--------|--------|-----------|-------|
| `/orgs/{orgId}/workspaces/{workspaceId}/docs` | POST | ⚠️ Partial | document_validation_test.go | Known issue: 404 after creation |
| `/docs/{docId}` | GET | ✅ Tested | document_validation_test.go | Get document metadata |
| `/docs/{docId}` | PATCH | ⚠️ Partial | - | Update document |
| `/docs/{docId}` | DELETE | ❌ Not Tested | - | Delete document |
| `/docs/{docId}/download` | GET | ❌ Not Tested | - | Export as Excel |
| `/docs/{docId}/download?format=grist` | GET | ❌ Not Tested | - | Export as Grist/SQLite |
| `/docs/{docId}/copy` | POST | ❌ Not Tested | - | Copy document |
| `/docs/{docId}/replace` | POST | ❌ Not Tested | - | Replace document content |

### Formulas & Data Types

| Feature | Status | Test File | Notes |
|---------|--------|-----------|-------|
| **Arithmetic Formulas** | ✅ Tested | formula_validation_test.go | SUM, AVERAGE, MIN, MAX |
| **Logical Formulas** | ⚠️ Partial | formula_validation_test.go | IF, AND (basic) |
| **Text Formulas** | ⚠️ Partial | formula_validation_test.go | CONCATENATE, UPPER, LOWER (basic) |
| **Date Formulas** | ⚠️ Partial | formula_validation_test.go | DATE (basic) |
| **Lookup Formulas** | ❌ Not Tested | - | VLOOKUP, lookupOne, lookupRecords |
| **Aggregation Formulas** | ❌ Not Tested | - | COUNT, COUNTIF, SUMIF |
| **Round-trip Numeric** | ✅ Tested | formula_validation_test.go | Large numbers, negatives, zero |
| **Round-trip Text** | ✅ Tested | formula_validation_test.go | Unicode, long text, special chars |
| **Round-trip Dates** | ⚠️ Partial | formula_validation_test.go | Basic dates, leap years |
| **Round-trip Boolean** | ✅ Tested | formula_validation_test.go | True/false values |

### Webhooks APIs

| Endpoint | Method | Status | Test File | Notes |
|----------|--------|--------|-----------|-------|
| `/docs/{docId}/webhooks` | GET | ❌ Not Tested | - | List webhooks |
| `/docs/{docId}/webhooks` | POST | ❌ Not Tested | - | Create webhook |
| `/docs/{docId}/webhooks/{webhookId}` | PATCH | ❌ Not Tested | - | Update webhook |
| `/docs/{docId}/webhooks/{webhookId}` | DELETE | ❌ Not Tested | - | Delete webhook |
| `/docs/{docId}/webhooks/{webhookId}/queue` | GET | ❌ Not Tested | - | Get webhook queue |
| `/docs/{docId}/webhooks/{webhookId}/queue` | DELETE | ❌ Not Tested | - | Clear webhook queue |

### Attachments APIs

| Endpoint | Method | Status | Test File | Notes |
|----------|--------|--------|-----------|-------|
| `/docs/{docId}/attachments` | GET | ❌ Not Tested | - | List attachments |
| `/docs/{docId}/attachments` | POST | ❌ Not Tested | - | Upload attachment |
| `/docs/{docId}/attachments/{attachmentId}` | GET | ❌ Not Tested | - | Download attachment |
| `/docs/{docId}/attachments/{attachmentId}` | DELETE | ❌ Not Tested | - | Delete attachment |
| `/docs/{docId}/attachments/{attachmentId}/restore` | POST | ❌ Not Tested | - | Restore attachment |

### Excluded from Scope

The following API categories are explicitly **excluded** from the test validation scope as per the epic requirements:

- **SCIM/User Management APIs**: User provisioning, group management
- **Organization Management APIs**: Organization CRUD operations
- **Workspace Management APIs**: Workspace CRUD operations

## Priority Roadmap

### Next Steps (Priority Order)

1. **Webhooks** (P1)
   - Create webhook test suite
   - Test all CRUD operations
   - Validate webhook queue management

2. **Attachments** (P2)
   - Upload/download attachment testing
   - Test attachment lifecycle
   - Validate attachment metadata

3. **Advanced Formulas** (P2)
   - VLOOKUP/lookup functions
   - Aggregation formulas (COUNT, COUNTIF, SUMIF)
   - Complex formula expressions

4. **Document Operations** (P3)
   - Export testing (Excel, Grist formats)
   - Document copy/replace operations
   - Document deletion

5. **SQL Queries** (P3)
   - Test SQL endpoint
   - Parameterized queries
   - Query result validation

### Known Limitations

1. **Document Creation Issue**: API-created documents return 404 immediately after creation. Tests work around this by using existing documents.

2. **Formula Calculation Delay**: Formula column values may not be immediately available in GET responses. Tests handle this with optional validation.

3. **No Transaction Support**: Tests cannot be fully isolated; they may create persistent test data in documents.

## Test Statistics

### Current Coverage

- **Test Files**: 4 files
- **Test Functions**: 50+ test functions
- **Total Test Records**: 500+ records
- **Total Assertions**: 200+ assertions
- **Test Execution Time**: ~26 seconds (all tests)

### Test Data Volume

- Table/Column Management: 50-100 records
- Record CRUD: 300 records (bulk operations)
- Formula Validation: 10 records
- Data Type Validation: 100 records

## References

- [Grist API Documentation](https://support.getgrist.com/api/)
- [gristle Test Suite](../../gristapi/)
- [Test Documents Inventory](test-documents.md)
