## [1.3.0-beta.1] - 2025-12-30

### Added

#### ASCII Visualizations for Time Series Data
- **Sparklines**: Mini-graphs using Unicode blocks (▁▂▃▄▅▆▇█) for visual trend display
- **Trend Indicators**: UP/DOWN/FLAT indicators showing direction of metric changes
- **Compact Statistics**: Min/Max/Avg/Current value summaries in token-efficient format
- **Anomaly Detection**: Statistical outlier identification (>2σ) automatically highlighted
- **Health Bars**: Visual progress indicators for percentage-based metrics
- **Automatic Integration**: Sparklines automatically appear in time series responses
- **New Module**: `internal/handlers/visualization.go` (178 lines) with comprehensive test suite (227 lines)

#### Contextual Suggestions for LLM Guidance
- **Smart Recommendations**: Context-aware "Next suggested actions" in all major response formatters
- **Alert-Based Suggestions**: When alerts detected, suggests investigating critical sensors
- **Sensor-Based Suggestions**: Recommends viewing alerts, trends, or narrowing search based on status
- **Device Overview Suggestions**: Proactive guidance for exploring device details
- **Search Result Suggestions**: Helps inspect and navigate found items
- **Command-Ready Format**: All suggestions use valid MCP command syntax, ready to execute
- **Reduces Back-and-Forth**: Proactive tool discovery reduces LLM query iterations

#### Pedagogical Error Handling System (from commit 661ccc8)
- **ErrorGuidance Helper**: Educational, actionable error messages designed for LLM users
- **6 Predefined Error Types**:
  - `ErrInvalidSensorID`: Guides users to search tools for valid sensor IDs
  - `ErrDeviceNotFound`: Suggests hierarchy/search tools to discover devices
  - `ErrSensorNotFound`: Provides troubleshooting steps for missing sensors
  - `ErrEmptySearchTerm`: Explains minimum search requirements
  - `ErrInvalidTimeRange`: Guides on valid time range parameters
  - `ErrCustomQueriesDisabled`: Explains security restrictions and alternatives
- **Enhanced Input Validation**: Early sensor_id validation in all metrics tool handlers
- **Test Coverage**: 14 test cases covering all error scenarios

### Changed
- **Time Series Formatting**: Enhanced `formatTimeSeriesForLLM()` to include automatic sparklines and trend analysis
- **Response Formatters**: All major formatters now include contextual suggestions
- **Error Philosophy**: Errors now guide toward resolution instead of just blocking actions
- **Token Efficiency**: Compact ASCII visualizations reduce token usage vs verbose JSON
- **User Experience**: LLMs can now visualize trends at a glance and get proactive guidance

### Fixed
- **Division by Zero**: Critical fix in `TrendIndicator()` when `firstAvg == 0`
- **Edge Cases**: Added test coverage for zero values (zero_to_positive, zero_to_negative, all_zeros)
- **Test Suite**: All tests now pass including new edge case scenarios

### Documentation
- **README.md**: Added ASCII Visualizations and Contextual Suggestions feature descriptions
- **ARCHITECTURE.md**: Complete visualization and suggestion system documentation (+141 lines)
- **TOOLS.md**: LLM-optimized features and updated tool response formats (+62 lines)
- **USAGE.md**: Detailed examples and usage guides for new features (+147 lines)
- **TROUBLESHOOTING.md**: Updated with error handling guidance

### Technical Details

#### New Files
- `internal/handlers/visualization.go` (178 lines)
- `internal/handlers/visualization_test.go` (227 lines)
- `internal/handlers/tools_metrics_integration_test.go` (221 lines)
- `internal/handlers/formatting_suggestions_test.go` (252 lines)
- `internal/handlers/errors.go` (from commit 661ccc8)
- `internal/handlers/errors_test.go` (from commit 661ccc8)

#### Modified Files
- `internal/handlers/formatting.go` (+93 lines for suggestions)
- `internal/handlers/tools_metrics.go` (+70 lines for sparklines, validation)
- Documentation files: 5 markdown files updated

#### Test Coverage
- All unit tests passing (100%)
- New visualization tests: 30+ test cases
- New suggestion tests: 15 test cases
- New error handling tests: 14 test cases
- Race condition tests: All passing
- Total test coverage: Comprehensive across all modules

#### Commits Included
- `193989f`: fix: handle division by zero in TrendIndicator and update documentation
- `e6ad019`: feat: add ASCII visualizations and contextual suggestions for LLM guidance
- `661ccc8`: feat: add pedagogical error handling system for LLM users

---

### Binary Checksums (SHA256)

```
6756276c3abe11497ea54c4ce52e1118c95832b2bdaf4e85466116fe9d5c8adb  mcp-server-prtg_darwin_amd64.zip
85f743088e30939a8232d103a5a1ed909dae9311d87b0b173b9196ce3c600c7e  mcp-server-prtg_darwin_arm64.zip
fb762be729d2a06124c4a67b96c00b6245dfc09572323880d0cd695cb256999b  mcp-server-prtg_linux_amd64.zip
470db0ff0b55576a8666fa8c18bb436dda63507f650e9c0963888a47b23d8916  mcp-server-prtg_linux_arm64.zip
840035d5d46fe3b5403203cc77ee7ab9b3a2f43d98a035d7c828e4b682e32243  mcp-server-prtg_windows_amd64.zip
```

---

### Benefits

**For LLMs:**
- Visual trend comprehension at a glance without parsing JSON
- Automatic anomaly alerts reduce need for statistical analysis
- Proactive command suggestions reduce back-and-forth queries
- Educational error messages guide toward successful tool usage

**For Users:**
- Token-efficient compact format reduces API costs
- Faster troubleshooting with visual indicators
- Better tool discovery through contextual suggestions
- More intuitive error handling

**For Developers:**
- Comprehensive test coverage ensures reliability
- Modular visualization system easy to extend
- Well-documented architecture for future enhancements
- Backward compatible with existing features

---

**Release prepared by:** Matthieu Noirbusson

**Code Review Score:** 10/10 (after division by zero fix)

**Note:** This is a beta release for testing the new visualization and suggestion systems. The changes are backward compatible and add new functionality without breaking existing features. Inspired by best practices from Honeycomb's MCP implementation.

**Testing Status:**
- All unit tests: PASS ✓
- Race condition tests: PASS ✓
- Security tests: PASS ✓
- Edge case tests: PASS ✓
- Integration tests: PASS ✓
