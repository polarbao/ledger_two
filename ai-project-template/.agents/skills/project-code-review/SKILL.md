---
name: project-code-review
description: Use for reviewing {{PROJECT_NAME}} diffs, pull requests, uncommitted changes, architecture changes, tests, and release risks.
---

# {{PROJECT_NAME}} Code Review

Read first:

1. `AGENTS.md`
2. `.agents/docs/always-on-rules.md`
3. `.agents/docs/code-standards.md`
4. `.agents/docs/build-and-test.md`
5. Relevant source, tests, and build files

## Review Stance

Lead with findings ordered by severity. Prioritize correctness, security, behavior regressions, missing tests, build/deployment risk, and maintainability.

## Output

1. Findings with file/line references.
2. Open questions or assumptions.
3. Brief summary only after findings.
