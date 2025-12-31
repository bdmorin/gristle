# Grist API Test Data & Validation Infrastructure

This directory contains documentation for the comprehensive test data and validation infrastructure used to test the Grist API implementation in `gristle`.

## Overview

The test infrastructure uses a **dogfooding approach** - we use `gristapi` itself to create test documents and validate API operations. Tests are fully automated and integrated with `go test`.

## Quick Start

```bash
# Set environment variables
export GRIST_URL="https://grist.hexxa.dev"
export GRIST_TOKEN="your-api-token"

# Run all validation tests
go test ./gristapi -v

# Run specific test suites
go test ./gristapi -v -run TestTableAndColumnManagement
go test ./gristapi -v -run TestRecordCRUD
go test ./gristapi -v -run TestFormulaAndDatatypeValidation
```

## Test Documents

See [test-documents.md](test-documents.md) for a complete inventory of test documents, their purposes, and structures.

### Primary Test Document

- **Document ID**: `g7pesgBnD5B5FsN4hUF9BB`
- **Location**: Hexxa/Home workspace
- **Purpose**: Main test document for all validation tests
- **URL**: https://grist.hexxa.dev/o/docs/g7pesgBnD5B5FsN4hUF9BB

## Test Coverage

See [api-coverage-matrix.md](api-coverage-matrix.md) for detailed API endpoint coverage analysis.

### Implemented Test Suites

1. **Phase 1: Table/Column Management** ([gristapi/table_validation_test.go](../../gristapi/table_validation_test.go))
   - Table CRUD operations
   - Column type testing (all 11 Grist column types)
   - Column modifications and type conversions
   - 50-100 records per test table

2. **Phase 2: Record CRUD** ([gristapi/record_validation_test.go](../../gristapi/record_validation_test.go))
   - Complete CRUD operations (Add, Get, Update, Delete, Upsert)
   - Edge cases: Unicode, special characters, nulls, large data
   - Bulk operations (300+ records)
   - Filtering, sorting, pagination

3. **Phase 3: Document CRUD** ([gristapi/document_validation_test.go](../../gristapi/document_validation_test.go))
   - Document creation and metadata retrieval
   - Partial coverage (some endpoints not yet implemented)

4. **Phase 4: Formula & Data Type Validation** ([gristapi/formula_validation_test.go](../../gristapi/formula_validation_test.go))
   - Arithmetic formulas (SUM, AVERAGE, MIN, MAX)
   - Data type round-trip integrity (100 records)
   - Edge cases: large numbers (>1e15), long text (>10k chars), Unicode

## Test Execution Results

### Latest Test Run

All tests passing as of Phase 4 completion:

```
✓ TestTableAndColumnManagement (9.3s)
✓ TestRecordCRUD (14.5s)
✓ TestFormulaAndDatatypeValidation (2.0s)
```

### Known Issues

1. **Document Creation**: Newly created documents via API return 404 errors immediately after creation. Workaround: Use existing known-good document ID.

2. **Formula Columns**: Formula column values are not always returned immediately in GET requests (server may need time to calculate). Tests handle this gracefully.

3. **Export Operations**: Not yet fully tested (Phase 3 partial coverage).

## Data Size

- **Total Test Records**: 500+ records across all test suites
- **Record CRUD Test**: 300 bulk records
- **Data Type Validation**: 100 edge case records
- **Table Management**: 50-100 records per table type

## Edge Cases Covered

### Unicode Support
- ✓ Emoji: 👋🌍🚀
- ✓ Japanese: こんにちは
- ✓ Chinese: 你好
- ✓ Korean: 안녕
- ✓ Arabic (RTL): אבג

### Special Characters
- ✓ Quotes and backslashes
- ✓ Newlines and tabs
- ✓ HTML tags
- ✓ JSON strings

### Numeric Edge Cases
- ✓ Very large numbers (>1e15)
- ✓ Negative numbers
- ✓ Zero values
- ✓ MaxInt32/MinInt32
- ✓ Floating point precision

### Text Edge Cases
- ✓ Empty strings
- ✓ Very long text (>10k characters)
- ✓ Multi-line text

### Null/Empty Handling
- ✓ Null values
- ✓ Empty fields
- ✓ Missing fields

## Integration with CI/CD

Tests require:
- `GRIST_URL` environment variable
- `GRIST_TOKEN` environment variable

If these are not set, tests are skipped automatically.

## Example API Calls

### Create a Table

```go
tableData := map[string]interface{}{
    "tables": []map[string]interface{}{
        {
            "id": "Products",
            "columns": []map[string]interface{}{
                {"id": "Name", "fields": map[string]interface{}{"label": "Product Name", "type": "Text"}},
                {"id": "Price", "fields": map[string]interface{}{"label": "Price", "type": "Numeric"}},
            },
        },
    },
}
```

### Add Records

```go
records := []map[string]interface{}{
    {"Name": "Laptop", "Price": 999.99},
    {"Name": "Mouse", "Price": 29.99},
}
result, status := gristapi.AddRecords(docID, "Products", records, nil)
```

### Get Records with Filters

```go
opts := &gristapi.GetRecordsOptions{
    Filter: map[string][]interface{}{
        "Price": {">", 50.0},
    },
    Sort: "-Price",
    Limit: 10,
}
records, status := gristapi.GetRecords(docID, "Products", opts)
```

## Future Work

See [api-coverage-matrix.md](api-coverage-matrix.md) for planned test coverage improvements.

### Not Yet Tested

- SCIM/user management APIs (excluded from scope)
- Organization/workspace management (excluded from scope)
- Advanced formula functions (VLOOKUP, COUNTIF, SUMIF)
- Webhook operations
- Attachment operations
- SQL query endpoint

## Contributing

When adding new test coverage:

1. Add test functions to appropriate `*_validation_test.go` file
2. Document test document IDs in the beads issue
3. Update this documentation with new coverage
4. Ensure tests are idempotent and clean up after themselves
5. Handle edge cases and error conditions

## References

- [Grist API Documentation](https://support.getgrist.com/api/)
- [Test Documents Inventory](test-documents.md)
- [API Coverage Matrix](api-coverage-matrix.md)
- [CLAUDE.md Project Instructions](../../CLAUDE.md)
