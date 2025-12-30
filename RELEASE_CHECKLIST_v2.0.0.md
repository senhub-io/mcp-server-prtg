# Release Checklist: v2.0.0 Final

**Current Status:** v2.0.0-alpha.1 (local tag only, not pushed)

**Target:** v2.0.0 (full production release with npm publication)

---

## Phase 1: Alpha Validation (Current)

### Completed ✓
- [x] stdio mode implementation (`cmd/server/stdio.go`)
- [x] Environment-based configuration
- [x] npm package structure (`npm-package/`)
- [x] Cross-platform binary wrapper (`npm-package/bin/index.js`)
- [x] All platform binaries built (`build/` directory)
- [x] Documentation (4 new markdown files)
- [x] CHANGELOG for v2.0.0-alpha.1

### Pending ⚠️
- [ ] **Stage all platform binaries in npm package:**
  ```bash
  # Copy from build/ to npm-package/bin/binaries/
  cp build/mcp-server-prtg_darwin_amd64 npm-package/bin/binaries/mcp-server-prtg-darwin-amd64
  cp build/mcp-server-prtg_darwin_arm64 npm-package/bin/binaries/mcp-server-prtg-darwin-arm64
  cp build/mcp-server-prtg_linux_amd64 npm-package/bin/binaries/mcp-server-prtg-linux-amd64
  cp build/mcp-server-prtg_linux_arm64 npm-package/bin/binaries/mcp-server-prtg-linux-arm64

  # Extract Windows binary from zip first
  unzip build/mcp-server-prtg_windows_amd64.zip -d /tmp
  cp /tmp/mcp-server-prtg_windows_amd64.exe npm-package/bin/binaries/mcp-server-prtg-windows-amd64.exe
  ```

- [ ] **Test local npm package:**
  ```bash
  cd npm-package
  npm pack
  # Creates: senhub-io-mcp-server-prtg-2.0.0-alpha.1.tgz

  # Install globally for testing
  npm install -g ./senhub-io-mcp-server-prtg-2.0.0-alpha.1.tgz

  # Test binary auto-detection
  which mcp-server-prtg
  mcp-server-prtg --help  # Should fail with helpful error (needs stdio mode)

  # Test stdio mode manually
  export PRTG_URL="https://your-prtg.com:1616"
  export PRTG_API_TOKEN="your-token"
  node $(which mcp-server-prtg) stdio
  ```

- [ ] **Validate with Claude Desktop (macOS):**
  ```json
  // Add to ~/Library/Application Support/Claude/claude_desktop_config.json
  {
    "mcpServers": {
      "prtg-alpha": {
        "command": "mcp-server-prtg",
        "args": ["stdio"],
        "env": {
          "PRTG_URL": "https://prtg.example.com:1616",
          "PRTG_API_TOKEN": "your-token",
          "PRTG_VERIFY_SSL": "false"
        }
      }
    }
  }
  ```
  - [ ] Restart Claude Desktop
  - [ ] Verify "prtg-alpha" server appears in MCP servers list
  - [ ] Test `prtg_get_channel_current_values` tool
  - [ ] Test `prtg_get_sensor_timeseries` tool
  - [ ] Test `prtg_get_sensor_history_custom` tool
  - [ ] Check stderr logs for any errors

---

## Phase 2: Release Candidate (v2.0.0-rc.1)

### Pre-RC Requirements
- [ ] All alpha validation tests pass
- [ ] All platform binaries staged and tested
- [ ] At least 1 successful Claude Desktop integration test
- [ ] No critical bugs in stdio mode

### RC Tasks
- [ ] Update `npm-package/package.json` version to `2.0.0-rc.1`
- [ ] Create CHANGELOG_2.0.0-rc.1.md (based on alpha + fixes)
- [ ] Rebuild all platform binaries (if code changes)
- [ ] Re-stage binaries in npm-package/bin/binaries/
- [ ] Create git tag `v2.0.0-rc.1` (local)
- [ ] Test npm pack + install on 3+ platforms
- [ ] **DO NOT publish to npm yet** (RC is still testing)

### RC Testing
- [ ] **macOS (ARM64):** Test with Claude Desktop
- [ ] **macOS (x64):** Test on Intel Mac (if available)
- [ ] **Linux (x64):** Test with Continue.dev or Cursor
- [ ] **Windows (x64):** Test with Claude Desktop or Cursor
- [ ] **npm installation:** Test `npx -y @senhub-io/mcp-server-prtg` (dry-run, before publish)

### RC Success Criteria
- All 3 PRTG API v2 tools work correctly
- Binary auto-detection works on all platforms
- Environment variables properly passed to Go binary
- No crashes or stderr errors during normal operation
- Clean shutdown on SIGINT/SIGTERM

---

## Phase 3: Final Release (v2.0.0)

### Pre-Release Requirements
- [ ] RC testing completed successfully
- [ ] All critical bugs fixed
- [ ] Documentation reviewed and updated
- [ ] main README.md updated to highlight stdio mode

### Main README.md Updates
- [ ] Update "Quick Installation" section to prioritize npm/npx
- [ ] Add stdio mode as primary deployment method
- [ ] Move HTTP server mode to "Alternative: HTTP Server Mode" section
- [ ] Update features list to mention dual-mode support
- [ ] Add npm installation badges

### npm Publication Preparation
- [ ] Update `npm-package/package.json` version to `2.0.0`
- [ ] Verify `files` field includes all necessary files:
  ```json
  "files": [
    "bin/",
    "README.md",
    "LICENSE"
  ]
  ```
- [ ] Copy LICENSE file to npm-package/ directory:
  ```bash
  cp LICENSE npm-package/LICENSE
  ```
- [ ] Test `npm pack` output for correct file inclusion:
  ```bash
  cd npm-package
  npm pack --dry-run
  # Verify bin/binaries/* are included
  ```

### Git Tag & GitHub Release
- [ ] Create git tag `v2.0.0`:
  ```bash
  git tag -a v2.0.0 -F CHANGELOG_2.0.0.md
  ```
- [ ] **CONFIRMATION REQUIRED:** Push tag to remote:
  ```bash
  git push origin v2.0.0
  ```
- [ ] Create GitHub Release:
  ```bash
  gh release create v2.0.0 \
    build/mcp-server-prtg_darwin_amd64.zip \
    build/mcp-server-prtg_darwin_arm64.zip \
    build/mcp-server-prtg_linux_amd64.zip \
    build/mcp-server-prtg_linux_arm64.zip \
    build/mcp-server-prtg_windows_amd64.zip \
    --title "Release 2.0.0 - stdio Mode Support" \
    --notes-file CHANGELOG_2.0.0.md
  ```

### npm Publication
- [ ] **CONFIRMATION REQUIRED:** Publish to npm:
  ```bash
  cd npm-package
  npm publish --access public
  # This makes the package available at:
  # https://www.npmjs.com/package/@senhub-io/mcp-server-prtg
  ```
- [ ] Verify npm publication:
  ```bash
  npm view @senhub-io/mcp-server-prtg
  npm info @senhub-io/mcp-server-prtg dist.tarball
  ```
- [ ] Test installation from npm:
  ```bash
  npx -y @senhub-io/mcp-server-prtg --help
  ```

### Post-Release
- [ ] Update project README.md with npm installation instructions
- [ ] Announce release on GitHub Discussions (if applicable)
- [ ] Update documentation links to point to v2.0.0
- [ ] Monitor npm download stats and GitHub issues for problems

---

## Phase 4: Future Enhancements (v2.1.0+)

### Planned Features
- [ ] **CI/CD Automation:**
  - GitHub Actions workflow for multi-platform binary builds
  - Automatic binary staging in npm-package/bin/binaries/
  - Automated npm publish on git tag
  - SHA256 checksum generation and verification

- [ ] **Database Tool Migration:**
  - Migrate 12 PostgreSQL tools to PRTG API v2
  - Enable full 15-tool support in stdio mode
  - Deprecate PostgreSQL dependency

- [ ] **Enhanced MCP Client Support:**
  - Test and validate Continue.dev configuration
  - Test and validate Cursor configuration
  - Test and validate Cline configuration
  - Add platform-specific installation guides

- [ ] **npm Package Improvements:**
  - Add `postinstall` script for binary verification
  - Add platform-specific installation messages
  - Add telemetry (opt-in) for usage statistics

---

## Known Issues & Limitations

### Alpha Known Issues
1. **npm binaries incomplete:** Only darwin-arm64 staged (others need copying)
2. **No Windows testing:** stdio mode untested on Windows
3. **No CI/CD:** Manual process for binary packaging

### Design Limitations (Won't Fix in v2.0)
1. **Database tools unavailable in stdio mode:** By design, requires HTTP mode
2. **Environment variables only:** No config file support in stdio mode
3. **No multi-PRTG support:** One PRTG instance per MCP server instance

---

## Testing Matrix

### Platforms
| Platform | Architecture | Binary | npm Install | Claude Desktop | Continue.dev | Cursor | Cline |
|----------|-------------|--------|-------------|----------------|--------------|--------|-------|
| macOS    | ARM64 (M1)  | ✓      | ⚠️          | ⚠️             | ❌           | ❌     | ❌    |
| macOS    | x64 (Intel) | ✓      | ❌          | ❌             | ❌           | ❌     | ❌    |
| Linux    | x64         | ✓      | ❌          | ❌             | ❌           | ❌     | ❌    |
| Linux    | ARM64       | ✓      | ❌          | ❌             | ❌           | ❌     | ❌    |
| Windows  | x64         | ✓      | ❌          | ❌             | ❌           | ❌     | ❌    |

Legend:
- ✓ = Built/Completed
- ⚠️ = Partially tested
- ❌ = Not tested
- N/A = Not applicable

### Tools Testing (stdio mode)
| Tool | Status | Notes |
|------|--------|-------|
| `prtg_get_channel_current_values` | ⚠️ | Needs Claude Desktop testing |
| `prtg_get_sensor_timeseries` | ⚠️ | Needs Claude Desktop testing |
| `prtg_get_sensor_history_custom` | ⚠️ | Needs Claude Desktop testing |

---

## Release Timeline (Estimated)

- **v2.0.0-alpha.1:** 2025-12-30 (TODAY - local tag only)
- **v2.0.0-rc.1:** 2025-01-XX (after alpha validation + platform testing)
- **v2.0.0 final:** 2025-01-XX (after RC validation + npm publish)
- **v2.1.0:** TBD (database tool migration to API v2)

---

## Contacts & Resources

**Release Manager:** Matthieu Noirbusson (matthieu.noirbusson@sensorfactory.eu)

**Repository:** https://github.com/senhub-io/mcp-server-prtg

**npm Package:** https://www.npmjs.com/package/@senhub-io/mcp-server-prtg (not published yet)

**MCP Protocol:** https://modelcontextprotocol.io

**Support:** GitHub Issues - https://github.com/senhub-io/mcp-server-prtg/issues
