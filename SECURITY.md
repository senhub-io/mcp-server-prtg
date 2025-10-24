# Security Considerations

## Overview

This document outlines the security measures implemented in the PRTG MCP Server and provides recommendations for secure deployment.

## SQL Injection Protection

### Implemented Protections

1. **Parameterized Queries**: All database queries use PostgreSQL parameterized statements ($1, $2, etc.) to prevent SQL injection.

2. **Input Validation**: All user inputs are validated before being used in queries.

3. **Custom Query Restrictions**: The `prtg_query_sql` tool has strict limitations:
   - Only SELECT statements allowed
   - Forbidden keywords: DROP, DELETE, UPDATE, INSERT, ALTER, CREATE, TRUNCATE, EXEC, EXECUTE
   - SQL comments (/* */ and --) are blocked to prevent bypass attempts
   - Semicolons are blocked to prevent query chaining
   - Maximum result limit enforced (1000 rows)

### Known Limitations

The `prtg_query_sql` tool, while protected, still accepts raw SQL queries and should be considered a potential security risk. It is **strongly recommended** to disable this tool in production environments by:

1. Removing it from the tool registration in `internal/handlers/tools.go`
2. Or using environment variable control to enable/disable it

## Database Permissions

### Recommended Setup

Create a read-only PostgreSQL user for the MCP server:

```sql
-- Create read-only user
CREATE USER prtg_reader WITH PASSWORD 'your_secure_password';

-- Grant minimal permissions
GRANT CONNECT ON DATABASE prtg_data_exporter TO prtg_reader;
GRANT USAGE ON SCHEMA public TO prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO prtg_reader;

-- Ensure future tables also have read-only access
ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT ON TABLES TO prtg_reader;
```

### What This Prevents

- Data modification (INSERT, UPDATE, DELETE)
- Schema changes (ALTER, DROP, CREATE)
- Privilege escalation
- Access to other databases

## Credential Management

### Environment Variables

**DO NOT** hardcode credentials in configuration files. Use environment variables:

```bash
export PRTG_DB_PASSWORD="your_secure_password"
```

### Configuration Files

If using YAML configuration:
- Add `configs/config.yaml` to `.gitignore` (already done)
- Use file permissions to restrict access: `chmod 600 configs/config.yaml`
- Never commit configuration files with real credentials

### Production Secrets Management

For production deployments, use a proper secrets management system:
- **HashiCorp Vault**
- **AWS Secrets Manager**
- **Azure Key Vault**
- **Google Secret Manager**
- **Kubernetes Secrets**

## Network Security

### Database Access

1. **Firewall Rules**: Restrict PostgreSQL access to only necessary hosts
2. **SSL/TLS**: Use `PRTG_DB_SSLMODE=require` or higher in production
3. **Private Networks**: Keep database on private network, not publicly accessible

### MCP Server

The MCP server communicates via stdio (standard input/output):
- No network ports are opened
- Communication is local-only via the MCP client (e.g., Claude Desktop)
- No HTTP/REST API exposed

## Logging Security

### Sensitive Data in Logs

The server uses structured logging (slog) with these precautions:

1. **Debug Mode**: Avoid using `LOG_LEVEL=debug` in production as it may log query arguments
2. **Password Redaction**: Connection strings with passwords should not appear in logs
3. **Log Storage**: Ensure logs are stored securely with appropriate access controls

### Recommended Log Configuration

```bash
# Production
export LOG_LEVEL=info

# Development only
export LOG_LEVEL=debug
```

## Rate Limiting

**Current Status**: ⚠️ Not implemented

**Recommendation**: Implement rate limiting for production use to prevent:
- Denial of Service (DoS) attacks
- Resource exhaustion
- Database overload

**Suggested Implementation**:
```go
import "golang.org/x/time/rate"

// Add to DB struct
limiter: rate.NewLimiter(rate.Limit(100), 200)  // 100 req/s, burst 200
```

## Query Limits

### Default Limits

- `prtg_get_sensors`: 50 results (configurable, max recommended: 1000)
- `prtg_get_alerts`: 100 results (hardcoded)
- `prtg_top_sensors`: 10 results (configurable, max recommended: 100)
- `prtg_query_sql`: 100 results default, 1000 maximum (enforced)

### Purpose

Limits prevent:
- Excessive memory usage
- Long-running queries
- Database resource exhaustion
- Slow response times

## Connection Pool Security

### Current Configuration

```go
MaxOpenConns:     25
MaxIdleConns:     5
ConnMaxLifetime:  5 minutes
QueryTimeout:     30 seconds
```

### Recommendations

- **Don't increase MaxOpenConns excessively**: More connections = more database load
- **Monitor connection usage**: Watch for connection exhaustion
- **Adjust ConnMaxLifetime**: Lower if database has connection limits

## Vulnerability Scanning

### Go Vulnerabilities

Run regularly:
```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

### Dependency Updates

Keep dependencies updated:
```bash
go get -u ./...
go mod tidy
```

### Static Analysis

Use `golangci-lint` for security issues:
```bash
golangci-lint run --enable-all ./...
```

## Security Headers and Best Practices

### For Future HTTP/REST API (if added)

If you add an HTTP interface, implement:
- CORS restrictions
- Authentication (JWT, API keys)
- HTTPS only (TLS 1.2+)
- Rate limiting per client
- Request size limits
- Timeout controls

### Current Stdio-based Architecture

The current stdio-based architecture is inherently more secure because:
- No network exposure
- Client (Claude Desktop) controls access
- Local process isolation

## Incident Response

### If a Security Issue is Found

1. **Do not** open a public GitHub issue
2. Email security concerns to: [your-email]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### Security Updates

Security patches will be released as:
- Patch version bumps (1.0.x)
- Clearly marked in CHANGELOG
- Announced in GitHub releases

## Security Checklist for Production

Before deploying to production:

- [ ] Use read-only database user
- [ ] Enable SSL/TLS for database connection (`PRTG_DB_SSLMODE=require`)
- [ ] Use secrets management (not environment variables in systemd/docker files)
- [ ] Set `LOG_LEVEL=info` (not debug)
- [ ] Disable or remove `prtg_query_sql` tool
- [ ] Run `govulncheck` and fix any issues
- [ ] Review and restrict database firewall rules
- [ ] Set appropriate file permissions (chmod 600 for configs)
- [ ] Monitor connection pool usage
- [ ] Set up log rotation and secure storage
- [ ] Implement rate limiting (see recommendations above)
- [ ] Regular security updates (monthly)

## Known Security Considerations

### Custom Query Tool (`prtg_query_sql`)

**Risk Level**: Medium to High

**Mitigations**:
- Keyword blacklist (DROP, DELETE, UPDATE, etc.)
- Comment blocking (/* */ and --)
- Semicolon blocking (prevents chaining)
- SELECT-only enforcement
- Result limit enforcement (max 1000)

**Remaining Risks**:
- Complex SQL injection via nested queries
- Information disclosure (can query any table)
- Database fingerprinting

**Recommendation**: **Disable in production** or limit to trusted administrators only.

### Connection String Exposure

**Risk**: Password in connection string could appear in error messages

**Mitigation**:
- Don't log connection strings
- Handle database errors carefully
- Use password-less auth when possible (IAM, certificates)

## Compliance Considerations

### GDPR / Data Protection

If PRTG contains personal data:
- Ensure compliance with data protection regulations
- Implement data retention policies
- Provide data export capabilities
- Document data processing activities

### Audit Logging

For compliance, consider adding:
- Query audit trail
- User action logging
- Failed access attempts
- Data export events

## Updates and Maintenance

### Security Update Policy

- Critical vulnerabilities: Patch within 24 hours
- High severity: Patch within 1 week
- Medium severity: Patch within 1 month
- Low severity: Patch in next regular release

### Monitoring

Monitor for:
- Failed authentication attempts (when auth is added)
- Unusual query patterns
- Database connection errors
- High resource usage
- Dependency vulnerabilities

## Additional Resources

- [OWASP SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [Go Security Policy](https://go.dev/security/policy)
- [PostgreSQL Security](https://www.postgresql.org/docs/current/security.html)
- [PRTG Security Best Practices](https://www.paessler.com/manuals/prtg/security)

---

**Last Updated**: 2025-10-24
**Version**: 1.0.0
