# LedgerTwo Project Rules

## Backend

- Use Go.
- Use SQLite for MVP.
- Store money as integer cents. Never use float for money.
- Use REST JSON APIs.
- Use cookie-based auth.
- Use soft delete for transactions.
- Write audit logs for amount changes.
- Keep business logic in service layer, not handlers.
- Shared expenses must generate split records.
- Settlements must create settlement records. Do not mutate old bills as a shortcut.

## Frontend

- Use React + TypeScript + Vite.
- Use TanStack Query for server state.
- Use Zustand only for UI state.
- Use React Hook Form + Zod for forms.
- API amounts are always cents.
- UI displays yuan.

## Product Rules

- Distinguish payer, participant, owner, and creator.
- Private bills must not be visible to the other user.
- Settlement history must be traceable.

## Workflow

- Make small commits.
- Run tests before finishing.
- Do not modify database migrations casually after they are applied.
- Ask before destructive commands.
