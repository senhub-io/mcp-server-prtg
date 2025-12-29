## [1.3.0-beta.1] - 2025-12-29

### Added
- **Pedagogical Error Handling System**: New ErrorGuidance helper that provides educational, actionable error messages for LLM users
- **6 Predefined Error Messages** with contextual guidance:
  - `ErrInvalidSensorID`: Guides users to use the search tool to find valid sensor IDs
  - `ErrDeviceNotFound`: Suggests using the hierarchy or search tools to discover available devices
  - `ErrSensorNotFound`: Provides troubleshooting steps for missing sensors
  - `ErrEmptySearchTerm`: Explains minimum search term requirements
  - `ErrInvalidTimeRange`: Guides users on valid time range parameters
  - `ErrCustomQueriesDisabled`: Explains security restrictions and alternative tools
- **Comprehensive Test Coverage**: Added `errors_test.go` with full unit tests for all error scenarios
- **Enhanced Input Validation**: Added sensor_id validation to all metrics tool handlers to catch errors early

### Changed
- Error messages now guide LLM users toward resolution instead of just blocking actions
- All metrics tools now validate sensor_id parameter and return pedagogical errors for invalid inputs
- Updated documentation across README, ARCHITECTURE, TOOLS, USAGE, and TROUBLESHOOTING to reflect the new error handling approach

### Technical Details
- New files: `internal/handlers/errors.go`, `internal/handlers/errors_test.go`
- Modified: All metrics tool handlers in `internal/handlers/tools_metrics.go`
- Documentation updates: 5 markdown files updated with error handling guidance

### Test Results
- All unit tests passing (100%)
- New error handling tests: 14 test cases covering all scenarios
- Total test coverage includes error handling, metrics validation, and edge cases

---

**Binary checksums will be added after build**

---

**Release prepared by:** Matthieu Noirbusson

**Note:** This is a beta release for testing the new pedagogical error handling system. The changes are backward compatible and add new functionality without breaking existing features.
