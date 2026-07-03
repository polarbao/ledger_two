# Code Standards And Comments

This file captures project-specific coding style, comments, API contracts,
threading rules, and technology boundaries.

Replace all placeholders before using this template.

## Scope

Applies to:

- `{{MAIN_APP_PATH}}`
- `{{TEST_PATH}}`
- `{{PROJECT_SPECIFIC_CODE_PATHS}}`

Does not apply to forced rewrites of:

- third-party code;
- generated code;
- external SDK ABI boundaries;
- vendored files unless the project explicitly owns them.

## Technology Baseline

- Tech stack: `{{TECH_STACK}}`
- Language/runtime version: `{{LANGUAGE_VERSION}}`
- UI framework, if any: `{{UI_FRAMEWORK}}`
- Build tool: `{{BUILD_TOOL}}`
- Package manager: `{{PACKAGE_MANAGER}}`

## Formatting And Naming

Replace this section with the target project's actual rules.

- File naming: `{{FILE_NAMING_RULE}}`
- Type/class naming: `{{TYPE_NAMING_RULE}}`
- Function naming: `{{FUNCTION_NAMING_RULE}}`
- Local variable naming: `{{LOCAL_VARIABLE_NAMING_RULE}}`
- Member field naming: `{{MEMBER_FIELD_NAMING_RULE}}`
- Constant naming: `{{CONSTANT_NAMING_RULE}}`
- Brace style: `{{BRACE_STYLE}}`
- Import/include ordering: `{{IMPORT_INCLUDE_ORDERING}}`

## Ownership And Lifetime

Document how the project represents ownership and lifecycle.

Examples to adapt:

- Prefer RAII / deterministic cleanup where applicable.
- Prefer clear ownership over hidden global state.
- Avoid manual resource release when the language or framework provides a safer abstraction.
- Document lifetime and thread ownership for framework objects.

Project-specific rules:

- `{{OWNERSHIP_RULE_1}}`
- `{{OWNERSHIP_RULE_2}}`

## Public API Comments

Public APIs should document behavior, not restate implementation.

Required comment style: `{{PUBLIC_API_COMMENT_STYLE}}`

Public API comments should include:

- purpose;
- parameter units, ranges, and lifetime;
- return value semantics;
- error semantics;
- blocking/asynchronous behavior;
- thread/callback context when relevant.

Example:

```text
{{PUBLIC_API_COMMENT_EXAMPLE}}
```

## File Header Comments

New source files should include a short responsibility comment when useful.

Required file header style: `{{FILE_HEADER_STYLE}}`

Example:

```text
{{FILE_HEADER_EXAMPLE}}
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

- `{{FRAMEWORK_RULE_1}}`
- `{{FRAMEWORK_RULE_2}}`

## Domain Boundary

Document which types, modules, or framework concepts must not leak across
boundaries.

Examples:

- UI-specific types should not become domain API types.
- External SDK types should stay behind an adapter.
- Persistence models should not leak into public domain interfaces unless that is the accepted architecture.

Project-specific boundaries:

- `{{DOMAIN_BOUNDARY_1}}`
- `{{DOMAIN_BOUNDARY_2}}`

## External SDK / API / ABI Boundary

Document conservative rules for external APIs, protocols, SDKs, generated code,
or ABI-sensitive files.

- Do not reorder or rewrite ABI/protocol declarations without explicit evidence.
- Do not normalize third-party naming just to match project style.
- Do not hide integration errors silently.
- State whether behavior was verified against mocks, staging, production, hardware, or a real external service.

Project-specific boundaries:

- `{{EXTERNAL_BOUNDARY_1}}`
- `{{EXTERNAL_BOUNDARY_2}}`

## Units And Numeric Rules

Document default units and numeric safety rules.

- Default length unit: `{{DEFAULT_LENGTH_UNIT}}`
- Default angle unit: `{{DEFAULT_ANGLE_UNIT}}`
- Time unit: `{{DEFAULT_TIME_UNIT}}`
- Floating-point comparison rule: `{{FLOAT_COMPARISON_RULE}}`
- Data-size and memory rule: `{{DATA_SIZE_MEMORY_RULE}}`

## Error Handling And Logging

Document the project's error model.

- Error representation: `{{ERROR_MODEL}}`
- Logging library/service: `{{LOGGING_SYSTEM}}`
- Required log context: `{{REQUIRED_LOG_CONTEXT}}`
- Retry/timeout policy: `{{RETRY_TIMEOUT_POLICY}}`

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
