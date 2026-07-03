# Build And Test

## Baseline

- Build tool: `{{BUILD_TOOL}}`
- Package manager: `{{PACKAGE_MANAGER}}`
- Main build command: `{{BUILD_COMMAND}}`
- Main test command: `{{TEST_COMMAND}}`
- Quick validation command: `{{QUICK_VALIDATION_COMMAND}}`
- UI smoke command, if any: `{{UI_SMOKE_COMMAND}}`
- Integration command, if any: `{{INTEGRATION_TEST_COMMAND}}`

## Rules

- Prefer target/module-level configuration over global configuration.
- For new third-party dependencies, compare at least two candidates.
- Explain license, maintenance, build, deployment, and security impact.
- Do not claim configure/build/test success unless the command actually ran.
- Match validation to the changed surface.
- Separate static checks, build, unit tests, integration tests, deployment checks, hardware checks, and manual checks.
- State partial validation clearly.
- Do not treat UI smoke, mock, or diagnostic-only checks as production/hardware proof.
