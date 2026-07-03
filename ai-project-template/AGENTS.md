# motionControlSDK Codex Instructions

## Project Identity

- Project: `motionControlSDK`
- Repository: `motion_control_sdk`
- Main branch/ref: `current working branch`
- Main application path: `src; include; app/MotionTestApp for optional Qt test UI`
- Deprecated or historical paths: `old Qt .vcxproj / legacy Qt TcpClient / Qt SerialPort / MOC-based SDK paths outside this CMake mainline`
- Tech stack: `C++20, CMake, vcpkg, MSVC, libhv, spdlog, GTest, optional Qt 5.15 MotionTestApp`
- Build system: `CMake presets with Visual Studio 18 2026 generator`
- Main build command: `cmake --preset main; cmake --build --preset main-debug; cmake --build --preset main-release`
- Main test command: `ctest --preset sdk-release-tests`
- Formal docs path: `docs/guides; docs/reference; docs/project for transitional project docs`
- Codex task path: `ai_workspace`
- Historical archive path: `docs/plans/archive; ai_workspace/*/chat_logs; ai_workspace/*/analysis_reports`

## Always-On Hard Rules

- Default response language: `zh-CN`.
- Current implementation mainline is `src; include; app/MotionTestApp for optional Qt test UI`, not `old Qt .vcxproj / legacy Qt TcpClient / Qt SerialPort / MOC-based SDK paths outside this CMake mainline`.
- Read relevant files before modifying code.
- Do not claim commands, tests, builds, deployments, or external verification ran unless they actually ran.
- Before destructive operations, dependency upgrades, data migrations, production data changes, or broad rewrites, explain the plan and wait for confirmation.
- Prefer small, scoped changes that follow existing project boundaries.
- Before starting a task, inspect the working tree with `git status --short`.
- Execute only the user-requested task; do not continue to the next task unless explicitly asked.
- After a minimal task, run the task-specific validation commands and report any validation not run.
- Do not push unless explicitly instructed.

## Evidence Classification

- A: Current source, build files, tests, runtime config, or verified command output.
- B: Current formal PRD/DEV/ADR/design docs or accepted plans.
- C: Historical docs, demos, archived notes, old task cards, or old discussions.
- D: Deprecated, conflicting, or superseded material.

Use A/B/C/D labels for high-risk work.

## Skill Routing

- Development planning / cross-module design: `motion-sdk-dev-workflow`
- Code review / diff review: `motion-sdk-code-review`
- Architecture boundaries / ADRs: `motion-sdk-architecture-guardrails`
- Build, dependency, test command, and toolchain work: `motion-sdk-build`
- Context handoff: `motion-sdk-context-handoff`
- Documentation state conflicts: `motion-sdk-doc-state-resolver`
- Save/archive current AI conversation: `motion-sdk-chat-save`

## Reference Docs

- `.agents/docs/README.md`
- `.agents/docs/always-on-rules.md`
- `.agents/docs/workflow-map.md`
- `.agents/docs/project-profile.md`
- `.agents/docs/architecture-boundary.md`
- `.agents/docs/build-and-test.md`
- `.agents/docs/code-standards.md`
- `.agents/docs/doc-state.md`
- `.agents/docs/chat-save.md`

## Current AI Task Entry

- Formal PRD/DEV/DOC documents: `docs/guides; docs/reference; docs/project for transitional project docs`
- Codex task cards and prompts: `ai_workspace`
- Historical docs and completed task cards: `docs/plans/archive; ai_workspace/*/chat_logs; ai_workspace/*/analysis_reports`
- Current task card: `ai_workspace/2026-07-01_v1.1后续处理改造计划.md`

