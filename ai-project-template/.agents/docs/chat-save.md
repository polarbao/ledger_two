# {{PROJECT_NAME}} Chat Save Policy

## Purpose

Chat saves are durable project memory for useful engineering context across AI sessions.

Use this policy only when the user explicitly asks to save, archive, checkpoint, persist, or record the current AI conversation.

## Default Archive Location

```text
{{AI_WORKSPACE_PATH}}/<model>/chat_logs/YYYY-MM-DD.md
```

Longer standalone reports go to:

```text
{{AI_WORKSPACE_PATH}}/<model>/analysis_reports/
{{AI_WORKSPACE_PATH}}/integrated_reports/
{{AI_WORKSPACE_PATH}}/context_handoff/
```

## What To Save

- Current objective and why it matters.
- Decisions made and rationale.
- Evidence-backed project facts.
- Files read or modified.
- Commands run and observed results.
- Open risks, TODOs, and next recommended prompt.

## What Not To Save

- Secrets, tokens, credentials, cookies.
- Unsupported validation claims.
- Raw long transcripts unless explicitly requested.
- Personal data unrelated to the project.

## Format

```markdown
## YYYY-MM-DD
### <topic>
用户请求：
> ...

### 当前状态
### 关键决策
### 涉及文件
### 验证情况
### 未解决问题
### 下一步建议
```
