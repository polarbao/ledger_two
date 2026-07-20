# LedgerTwo AI Coding Rules

This repository is LedgerTwo, a private shared ledger web app for two-person/family expense tracking, splitting, settlement, import/export, backup, and NAS deployment.

## Read First

Before coding, AI agents must read:

1. `docs/00-文档索引.md`
2. `.agents/docs/README.md`
3. Current task entry when present:
   - `docs/codex_tasks/README.md`
   - `docs/codex_tasks/05-Foundation任务计划.md`
   - `docs/codex_tasks/08-产品路线图开发任务计划.md`
   - `docs/codex_tasks/09-Task41至Task49详细开发计划.md`
   - `docs/codex_tasks/10-Task33至Task40详细开发计划.md`
4. The task-specific document, usually one of:
   - `docs/14-后端模块实现规范.md`
   - `docs/15-前端模块实现规范.md`
   - `docs/16-测试与验收规范.md`
   - `docs/17-AI编码任务与提示词.md`
   - `docs/18-演示后AI编码任务与提示词.md`
   - `docs/prd/*`
   - `docs/tech/*`
   - `docs/ui/*`

For current work, treat `docs/codex_tasks/`, `docs/prd/20-28`, `docs/tech/18-短中期模块架构切片.md`, `docs/ui/14-v1.1至v1.2模块流程.md`, and the Task30 foundation documents as newer than the early Demo-only documents. Use `docs/13-演示版本范围锁定.md` only as historical scope control unless the user explicitly asks to work on the original Demo baseline.

## Product Scope

Task01-Task30 are considered complete and the project is in "Foundation before v1.1" hardening. Do not implement unapproved v1.1 business features unless explicitly requested.

The original Demo version supported exactly one shared ledger and exactly two users. Current code and docs include multi-ledger/membership foundation work; preserve existing behavior and do not treat early Demo-only documents as permission to remove newer functionality.

Do not implement bank sync, OCR, enterprise multi-tenant features, or native mobile app unless explicitly requested.

## Backend Rules

- Use Go.
- Use SQLite for MVP.
- Store money as integer cents. Never use float for money.
- Use REST JSON APIs.
- Use cookie-based auth for Web demo.
- Use soft delete for transactions.
- Write audit logs for amount changes and deletes.
- Keep business logic in service layer, not handlers.
- Shared expense must generate transaction_splits.
- Settlement must create settlement records and must not mutate old shared bills.

## Frontend Rules

- Use React + TypeScript + Vite.
- Use TanStack Query for server state.
- Use Zustand only for UI state.
- Use React Hook Form + Zod for forms.
- API amounts are always cents.
- UI displays yuan.
- Mobile layout first, desktop layout enhanced.

## Domain Rules

- Distinguish payer, participant, owner, and creator.
- Private bills must not be visible to the other user.
- Partner-readable bills are visible but not editable by the partner.
- Demo shared bills are editable only by creator.
- Settlement creates a settlement record. Do not modify historical shared expenses as a shortcut.

## Workflow

- Make small commits.
- Run tests before finishing.
- Do not modify applied migrations casually.
- Do not commit real secrets.
- Summarize changed files and test commands after each task.
