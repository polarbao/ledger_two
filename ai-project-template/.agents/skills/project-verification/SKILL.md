---
name: project-verification
description: Optional. Use only when {{PROJECT_NAME}} explicitly routes verification work to project-verification instead of project-build.
---

# {{PROJECT_NAME}} Verification

Read first:

1. `AGENTS.md`
2. `.agents/docs/build-and-test.md`
3. `.agents/docs/verification.md` if the project keeps a separate verification policy

## Workflow

1. Map changed files to appropriate checks.
2. Run safe, relevant checks when feasible.
3. Report exact commands and results.
4. State what remains unverified.

## Output

- 已运行命令
- 结果
- 未覆盖风险
- 建议补充验证
