# Build And Test

## Baseline

- Build tool: `go, pnpm, docker compose`
- Package manager: `Go modules, pnpm`
- Main build command: `cd backend && go build ./cmd/server; cd frontend && pnpm build`
- Main test command: `./run_tests.sh`
- Quick validation command: `cd backend && go test ./...`
- UI smoke command, if any: `cd frontend && pnpm lint && pnpm test && pnpm build`
- Integration command, if any: `docker compose build`

## Rules

- Prefer target/module-level configuration over global configuration.
- For new third-party dependencies, compare at least two candidates.
- Explain license, maintenance, build, deployment, and security impact.
- Do not claim configure/build/test success unless the command actually ran.
- Match validation to the changed surface.
- Separate static checks, build, unit tests, integration tests, deployment checks, hardware checks, and manual checks.
- State partial validation clearly.
- Do not treat UI smoke, mock, or diagnostic-only checks as production/hardware proof.
