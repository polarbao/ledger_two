# Verification

This file is optional. Prefer `.agents/docs/build-and-test.md` as the primary build/test entry unless the target project needs a separate verification policy.

## Commands

```powershell
cd backend && go build ./cmd/server; cd frontend && pnpm build
./run_tests.sh
```

## Policy

- Match verification to the changed surface.
- Separate static checks, build, unit tests, integration tests, deployment checks, and manual checks.
- State partial verification clearly.
- Never report unrun commands as passed.
