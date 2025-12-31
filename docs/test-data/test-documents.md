# Test Documents Inventory

This document provides a complete inventory of Grist documents used in the validation test suite.

## Overview

The test suite uses a **dogfooding approach** - documents and test data are created using the `gristapi` package itself. This ensures that the API client is tested with real-world usage patterns.

## Primary Test Document

### Main Test Document (All Tests)

- **Document ID**: `g7pesgBnD5B5FsN4hUF9BB`
- **Name**: Main Test Document
- **Location**: Hexxa organization, Home workspace
- **URL**: https://grist.hexxa.dev/o/docs/g7pesgBnD5B5FsN4hUF9BB
- **Purpose**: Primary document for all validation tests
- **Created**: Pre-existing document (API document creation has known issues)

#### Tables in Document

| Table Name | Purpose | Columns | Records | Test Phase |
|------------|---------|---------|---------|------------|
| `TestRecords` | Record CRUD validation | Name (Text), Age (Int), Email (Text), Score (Numeric), Active (Bool), SignupDate (Date) | ~300 | Phase 2 |
| `Products` | Table management, column types | Name (Text), Price (Numeric), InStock (Bool), Category (Choice), Quantity (Int) | 75 | Phase 1 |
| `Events` | Date/DateTime testing | Title (Text), EventDate (Date), StartTime (DateTime:UTC) | 50 | Phase 1 |
| `AllTypes` | All column types validation | TextField, NumericField, IntField, BoolField, DateField, DateTimeField, ChoiceField, ChoiceListField, RefField, RefListField, AttachmentsField | 60 | Phase 1 |
| `Categories` | Reference table for AllTypes | Name (Text) | 3 | Phase 1 |
| `Formulas` | Formula validation | A (Int), B (Int), Sum (Numeric formula), Avg (Numeric formula) | 10 | Phase 4 |
| `EdgeCases` | Data type round-trip testing | BigNum (Numeric), NegNum (Numeric), TextField (Text), UnicodeField (Text), DateField (Date) | 100 | Phase 4 |

### Document Creation Attempts

Due to a known API issue, documents created via the API return 404 errors immediately after creation. The following document IDs were generated during testing but are not accessible:

- `6Weic9A5D1UgBPx1hGKYoe` - test-formulas (created but 404)
- `3s2xX6tQTxe1JP1ro5Ya6X` - test-datatypes (created but 404)

**Workaround**: All tests use the pre-existing working document `g7pesgBnD5B5FsN4hUF9BB` instead.

## Test Data Details

### Phase 1: Table/Column Management

**Test File**: `gristapi/table_validation_test.go`

#### Products Table

```
Columns:
- Name: Text (Product name)
- Price: Numeric (with 2 decimal places)
- InStock: Bool (availability flag)
- Category: Choice (Electronics, Books, Clothing, Food)
- Quantity: Int (stock level)

Sample Data:
Laptop 1, $15.99, true, Electronics
Mouse 2, $20.99, true, Books
Keyboard 3, $25.99, true, Clothing
...
(75 total records)
```

#### Events Table

```
Columns:
- Title: Text (Event name)
- EventDate: Date (Event date)
- StartTime: DateTime:UTC (Event start time)

Sample Data:
Conference 1, 2025-01-01, 2025-01-01 09:00:00 UTC
Meeting 2, 2025-01-02, 2025-01-02 10:00:00 UTC
...
(50 total records)
```

#### AllTypes Table

Demonstrates all 11 Grist column types:

```
Columns:
- TextField: Text
- NumericField: Numeric (floating point)
- IntField: Int (integer)
- BoolField: Bool (boolean)
- DateField: Date (date only)
- DateTimeField: DateTime:UTC (date and time)
- ChoiceField: Choice (single selection from list)
- ChoiceListField: ChoiceList (multiple selections)
- RefField: Ref:Categories (reference to another table)
- RefListField: RefList:Categories (multiple references)
- AttachmentsField: Attachments (file attachments)

(60 records with varied data)
```

### Phase 2: Record CRUD

**Test File**: `gristapi/record_validation_test.go`

#### TestRecords Table

```
Columns:
- Name: Text
- Age: Int
- Email: Text
- Score: Numeric
- Active: Bool
- SignupDate: Date

Test Cases:
- Single record operations (add, update, delete)
- Bulk operations (300 records)
- Filter operations (by multiple criteria)
- Sort operations (ascending/descending)
- Pagination (limit/offset)
- Upsert operations (create and update)

Edge Cases Tested:
- Unicode: 👋🌍🚀 (emoji), こんにちは (Japanese), 你好 (Chinese), 안녕 (Korean)
- Special characters: quotes, backslashes, newlines, tabs, HTML, JSON
- Null/empty values
- Large text (10KB strings)
- Large numbers (MaxInt32, MinInt32, large floats)
```

### Phase 3: Document CRUD

**Test File**: `gristapi/document_validation_test.go`

**Status**: Partial coverage due to API limitations

- Document metadata retrieval: ✅ Working
- Document creation: ⚠️ Creates but returns 404
- Document export: ❌ Not yet tested
- Document deletion: ❌ Not yet tested

### Phase 4: Formula & Data Type Validation

**Test File**: `gristapi/formula_validation_test.go`

#### Formulas Table

```
Columns:
- A: Int (input value)
- B: Int (input value)
- Sum: Numeric (formula: A + B)
- Avg: Numeric (formula: (A + B) / 2)

Test Data:
A=1, B=2, Sum=3, Avg=1.5
A=2, B=4, Sum=6, Avg=3.0
...
(10 records)

Note: Formula columns may not return values immediately (server delay)
```

#### EdgeCases Table

```
Columns:
- BigNum: Numeric (tests >1e15)
- NegNum: Numeric (tests negative values)
- TextField: Text (tests long text, special chars)
- UnicodeField: Text (tests emoji, CJK, RTL)
- DateField: Date (tests edge cases like leap years)

Test Data:
100 records covering:
- Very large numbers: 1e16, 1e17, etc.
- Negative numbers: -1, -1000, etc.
- Long text: 11,000 character strings
- Unicode: emoji (🚀), Chinese (你好), RTL (אבג)
- Special chars: !@#$%\n\t
- Date edge cases: leap days, boundary dates
```

## Workspace Structure

```
Hexxa Organization
└── Home Workspace (ID: 183662)
    └── Main Test Document (ID: g7pesgBnD5B5FsN4hUF9BB)
        ├── TestRecords (Phase 2)
        ├── Products (Phase 1)
        ├── Events (Phase 1)
        ├── AllTypes (Phase 1)
        ├── Categories (Phase 1 - reference table)
        ├── Formulas (Phase 4)
        └── EdgeCases (Phase 4)
```

## Data Volume Summary

| Test Phase | Tables | Total Records | Test Duration |
|------------|--------|---------------|---------------|
| Phase 1: Table/Column Management | 4 | 188 | 9.3s |
| Phase 2: Record CRUD | 1 | 300+ | 14.5s |
| Phase 3: Document CRUD | 0 | 0 | N/A |
| Phase 4: Formula & Data Types | 2 | 110 | 2.0s |
| **Total** | **7** | **~598** | **~26s** |

## Known Issues

### Document Creation

**Issue**: Documents created via API (`POST /workspaces/{workspaceId}/docs`) return HTTP 200 and a document ID, but immediately return 404 when accessed.

**Symptoms**:
```
Created document with ID: '6Weic9A5D1UgBPx1hGKYoe'
GET /docs/6Weic9A5D1UgBPx1hGKYoe -> HTTP 404 "document not found"
```

**Workaround**: Use pre-existing working document `g7pesgBnD5B5FsN4hUF9BB` for all tests.

**Tracking**: Mentioned in `table_validation_test.go:26-27`

### Formula Columns

**Issue**: Formula column values are not always returned immediately in GET requests.

**Symptoms**:
```
Record 1: Sum column not returned (may be delayed by server)
```

**Workaround**: Tests log this as informational rather than failing.

**Tracking**: Handled in `formula_validation_test.go:101-103`

## Test Data Cleanup

Most tests clean up after themselves:

- `TestRecordCRUD`: Uses `defer` to delete all created records
- `TestTableAndColumnManagement`: Can be configured to delete test document (currently commented out)
- `TestFormulaAndDatatypeValidation`: Tables remain in document for manual inspection

To manually clean up test data:

```bash
# Delete all tables from test document
# (Not recommended - may break future test runs)
# Use Grist UI instead for selective cleanup
```

## Manual Testing

### View Test Document

Visit: https://grist.hexxa.dev/o/docs/g7pesgBnD5B5FsN4hUF9BB

### Run Individual Tests

```bash
# Run specific test suite
go test ./gristapi -v -run TestTableAndColumnManagement
go test ./gristapi -v -run TestRecordCRUD
go test ./gristapi -v -run TestFormulaAndDatatypeValidation

# Run all tests
go test ./gristapi -v
```

### Check Stored Document IDs

```bash
cat /tmp/grist-table-validation-test-doc.txt
```

## References

- [Grist API Documentation](https://support.getgrist.com/api/)
- [API Coverage Matrix](api-coverage-matrix.md)
- [Test Data README](README.md)
