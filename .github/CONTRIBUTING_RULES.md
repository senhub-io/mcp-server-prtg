# Contribution Rules

## Git Commit Guidelines

### ❌ FORBIDDEN - AI Attribution

**NEVER** add AI assistants (Claude, ChatGPT, Copilot, etc.) as co-authors in commit messages.

**Forbidden patterns:**
```
Co-Authored-By: Claude <noreply@anthropic.com>
Co-Authored-By: GitHub Copilot <...>
🤖 Generated with [Claude Code](...)
Generated with AI assistance
```

**Reason:**
- All contributions are made by human developers
- AI tools are assistive technologies, not contributors
- Commit history must reflect actual human authorship
- Company policy for professional repository management

### ✅ Correct Commit Message Format

```
type(scope): subject line

Detailed description of changes if needed.

Co-Authored-By: John Doe <john@example.com>
```

**Valid types:** feat, fix, docs, style, refactor, perf, test, chore, security

### Enforcement

This rule is enforced by:
1. Git filter-branch on repository (automatic cleanup)
2. Code review process
3. CI/CD pre-commit hooks (if configured)

### Questions?

Contact repository maintainers for any clarification.
