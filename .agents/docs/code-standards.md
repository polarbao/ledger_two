# Code Standards And Comments

This file captures project-specific coding style, comments, API contracts,
threading rules, and technology boundaries.

Replace all placeholders before using this template.

## Scope

Applies to:

- `backend; frontend`
- `backend *_test.go; frontend Vitest tests; run_tests.sh`
- `backend/internal; backend/migrations; frontend/src; deploy; docker-compose.yml`

Does not apply to forced rewrites of:

- third-party code;
- generated code;
- external SDK ABI boundaries;
- vendored files unless the project explicitly owns them.

## Technology Baseline

- Tech stack: `Go 1.22+, SQLite, chi, goose, React 19, TypeScript, Vite, TanStack Query, Zustand, React Hook Form, Zod, Docker Compose`
- Language/runtime version: `Go 1.22+; TypeScript 6.x; React 19`
- UI framework, if any: `React + Vite responsive Web/PWA`
- Build tool: `go, pnpm, docker compose`
- Package manager: `Go modules, pnpm`

## Formatting And Naming

Replace this section with the target project's actual rules.

- File naming: `Go package files use lower_snake or conventional names; React components use PascalCase.tsx; utility files use camel/lowercase names`
- Type/class naming: `Go exported types PascalCase; TypeScript types/interfaces PascalCase`
- Function naming: `Go exported functions PascalCase and unexported camelCase; TypeScript functions camelCase; React components PascalCase`
- Local variable naming: `camelCase where language idiomatic; short local names only for narrow Go scopes`
- Member field naming: `Go struct fields PascalCase for exported DTO/model fields; TypeScript object fields match API snake_case only in DTO layer`
- Constant naming: `Go constants PascalCase or camelCase by export; TypeScript constants SCREAMING_SNAKE only for true constants`
- Brace style: `Language default: gofmt for Go; Prettier/ESLint style for TypeScript`
- Import/include ordering: `Go gofmt/goimports grouping; TypeScript external imports before local imports, type-only imports with import type`

## Ownership And Lifetime

Document how the project represents ownership and lifecycle.

Examples to adapt:

- Prefer RAII / deterministic cleanup where applicable.
- Prefer clear ownership over hidden global state.
- Avoid manual resource release when the language or framework provides a safer abstraction.
- Document lifetime and thread ownership for framework objects.

Project-specific rules:

- `Keep money in integer cents end to end; never use float for persisted money or settlement math`
- `Keep business rules in service/usecase layers; handlers parse HTTP and repositories persist data`

## Public API Comments

Public APIs should document behavior, not restate implementation.

Required comment style: `Go exported public APIs should have concise doc comments when non-obvious; TypeScript public helpers need short comments only for domain-critical behavior`

Public API comments should include:

- purpose;
- parameter units, ranges, and lifetime;
- return value semantics;
- error semantics;
- blocking/asynchronous behavior;
- thread/callback context when relevant.

Example:

```text
// CalculateBalance returns per-member final settlement net amounts in cents.
```

## File Header Comments

New source files should include a short responsibility comment when useful.

Required file header style: `No mandatory file headers; prefer focused package/component comments only when useful`

Example:

```text
// Package settlement contains balance and transfer suggestion calculations.
```

Avoid comments that promise unverified production, hardware, deployment, or
external-service behavior.

## Internal Comments

Prefer comments that explain Why / How / Failure semantics.

Add comments for:

- cross-thread or async handoff;
- state-machine branches;
- external SDK/API calls;
- retries, timeouts, and error mapping;
- units and conversions;
- performance-sensitive buffers, batching, caching, or memory peaks;
- compatibility, ABI, protocol, or migration boundaries.

Avoid comments that merely repeat the next line of code.

## Framework And UI Rules

If the project has UI or framework-specific constraints, document them here.

Examples to adapt:

- UI updates must happen on the UI/main thread.
- Heavy work must move to a worker, service, queue, or background task.
- UI/presentation code must not bypass application/service boundaries.
- Framework object lifetimes and callback context must be explicit.

Project-specific rules:

- `TanStack Query owns server state; Zustand owns UI-only state`
- `React Hook Form + Zod should validate forms; API client must use credentials include and unified error parsing`

## Domain Boundary

Document which types, modules, or framework concepts must not leak across
boundaries.

Examples:

- UI-specific types should not become domain API types.
- External SDK types should stay behind an adapter.
- Persistence models should not leak into public domain interfaces unless that is the accepted architecture.

Project-specific boundaries:

- `Settlement creates settlement records and must not mutate historical shared expenses`
- `private bills and related attachments must not be visible through API or static paths to unauthorized users`

## External SDK / API / ABI Boundary

Document conservative rules for external APIs, protocols, SDKs, generated code,
or ABI-sensitive files.

- Do not reorder or rewrite ABI/protocol declarations without explicit evidence.
- Do not normalize third-party naming just to match project style.
- Do not hide integration errors silently.
- State whether behavior was verified against mocks, staging, production, hardware, or a real external service.

Project-specific boundaries:

- `SQLite data, backups, uploads, .env, and real secrets must never be committed`
- `NAS deployment must keep data/backups/uploads/logs mounted outside the container image`

## Units And Numeric Rules

Document default units and numeric safety rules.

- Default length unit: `not applicable`
- Default angle unit: `not applicable`
- Time unit: `ISO8601 timestamps; month filters use YYYY-MM`
- Floating-point comparison rule: `Do not use floating point for money; ratios/shares must round deterministically to cents`
- Data-size and memory rule: `Use pagination for transaction lists and streaming/preview for imports; avoid loading full history into UI state`

## Error Handling And Logging

Document the project's error model.

- Error representation: `Unified APIError/AppError with stable code, message, details and HTTP status mapping`
- Logging library/service: `Go standard log or project logging wrapper; do not log secrets`
- Required log context: `request_id when available, user_id, ledger_id, path, method, error_code, duration`
- Retry/timeout policy: `Prefer explicit request timeouts for external/file operations; no silent retries for money writes`

Rules:

- Do not silently swallow lower-level errors.
- Keep recovery policy in the correct layer.
- Include enough context to debug without leaking secrets.

## Review Checklist

- Relevant files, tests, and docs were read.
- Changes follow project naming and formatting rules.
- Public APIs have required comments.
- Complex logic explains Why / How / Failure semantics.
- Ownership and lifecycle are clear.
- Async/thread boundaries are explicit.
- External SDK/API/ABI boundaries are preserved.
- Units, numeric comparisons, and data-size implications are clear.
- Verification scope and remaining gaps are stated.
