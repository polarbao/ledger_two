# LedgerTwo Project Profile

## Identity

- Project: `LedgerTwo`
- Repository: `ledger_two`
- Domain: `private shared ledger web app for two-person/family expense tracking, splitting, settlement, import/export, backup, and NAS deployment`
- Current phase: `Foundation before v1.1; Task01-Task30 completed; v1.1 business scope not yet approved`
- Main application path: `backend; frontend`
- Deprecated or historical paths: `docs/project_analysis/extracted_archives; ai_workspace historical AI logs; early upload packs`
- Tech stack: `Go 1.22+, SQLite, chi, goose, React 19, TypeScript, Vite, TanStack Query, Zustand, React Hook Form, Zod, Docker Compose`
- Build system: `Go toolchain + Vite + Docker Compose`
- Test command: `./run_tests.sh`

## Core Capabilities

Replace this section with concise, evidence-backed project capabilities.

```text
- Two-person/shared ledger and evolving multi-ledger foundation
- Transaction CRUD, shared expense splits, settlements, reports
- CSV import/export, SQLite backup/restore guidance, attachments, PWA/offline drafts
- NAS Docker Compose deployment
```

## Critical Constraints

Replace this section with constraints that must be preserved during planning and implementation.

```text
- Money is integer cents only
- Soft delete for transactions
- Audit high-risk operations
- Backend is the source of truth for settlement and reports
- Do not commit secrets, databases, backups, or uploads
```

## Current Risk Areas

Use this section for code concentration, stale docs, fragile integration points, data migration risk, hardware risk, or release blockers.

```text
- Current docs mix Demo/v0.3/v1.0/Foundation states
- Config names need unification across docs, docker-compose, and config.go
- LedgerContext/RBAC and attachment access need hardening before v1.1
- Large transaction and frontend form modules need gradual decomposition
```
