## [1.2.1] - 2025-10-31

### Fixed
- Corrected `GetBusinessProcesses` SQL query by removing invalid columns that caused query failures
- Fixed database queries to use `self_group_id` instead of incorrect `parent_id` field, resolving data retrieval issues

### Performance
- Optimized `GetStatistics` query to prevent timeout issues on large PRTG databases

---

### Commits included:
- docs: improve GetStatistics godoc to document pg_class optimization (625f549)
- perf: optimize GetStatistics query to prevent timeout (25874f4)
- fix: correct GetBusinessProcesses SQL query - remove invalid columns (4fa87c1)
- fix: use self_group_id instead of parent_id in database queries (bf197d4)

### Checksums:
```
0c8a85ad74c869a78af099ad28483dd63f6459cb46e12d4c81babb3933bfea10  mcp-server-prtg_windows_amd64.zip
29cffa775d915891759ec9c3858e620086205c5cfcabfe103242a4e129da9eb7  mcp-server-prtg_darwin_arm64.zip
86f55b74a872032988b1838d35946d5f489848f6af21036bc3dd3d198386e659  mcp-server-prtg_linux_arm64.zip
96f0c17172fafb62e7b136cc2e7e1370d27a635a437a81afa8d4c9ce0aad3a62  mcp-server-prtg_linux_amd64.zip
f5d820ae2edd05f891624cb322b56798ced59f84617a020922c9856fac3e677a  mcp-server-prtg_darwin_amd64.zip
```

---

**Release prepared by:** Matthieu Noirbusson
