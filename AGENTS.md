# LedgerTwo AI Coding Rules

This repository is a two-person shared ledger web app.

## Read First

Before coding, AI agents must read:

1. `docs/00_DOCUMENT_INDEX.md`
2. `docs/13_DEMO_SCOPE_LOCK.md`
3. The task-specific document, usually one of:
   - `docs/14_BACKEND_MODULE_SPEC.md`
   - `docs/15_FRONTEND_MODULE_SPEC.md`
   - `docs/16_TEST_ACCEPTANCE_SPEC.md`
   - `docs/17_AI_CODING_TASKS.md`

## Product Scope

Demo version supports exactly one shared ledger and exactly two users.
Do not implement multi-ledger, multi-tenant, bank sync, OCR, budget, or mobile app unless explicitly requested.

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
