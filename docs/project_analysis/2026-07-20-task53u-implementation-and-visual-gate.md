# Task53U Fresh Light UI 实现与视觉门禁记录

日期：2026-07-20<br>
结论：通过。源码、自动化契约和隔离 schema 22 浏览器视觉门禁均完成，Task53U 可以关闭

## 1. 实现范围

| 范围 | 实现结果 |
|---|---|
| 分类摘要与筛选 | auto/suggested/fallback/manual/bulk/conflict/unresolved 摘要、筛选与双层行状态并存 |
| 行级解释 | 最终分类/账户/标签、来源、置信度、可读原因常驻展示 |
| 行编辑与学习 | 标签 `N/8`、记住商户、来源范围；行保存和学习成功/失败分别反馈 |
| 批量与重分类 | 接受建议、应用相同值、相同商户、可选长期学习、reclassify dry-run/确认 |
| 规则管理 | 精确商户、来源、auto/suggest、手工/学习分组、stale、提交命中次数和时间 |
| 默认元数据 | 新账本默认基础包/显式空白账本；既有账本 profile preview/conflict/apply |
| 兜底保护 | 归档 `expense_other/income_other` 时强制选择同类型非系统替代分类 |
| 正式契约 | 前端 DTO/API client 与正式 OpenAPI 已包含 default-profile 和 `metadata_profile` |

所有动作继续遵守“预览调整不等于提交账单”。接受建议、批量调整、学习规则和重新分类均不自动 commit。

## 2. 运行环境

| Item | Value |
|---|---|
| URL | `http://127.0.0.1:38092` |
| image | `ledger-two:1.3.0-rc-task53-98c3b14` |
| revision | `98c3b14fe3bc46cb6b531a8c3b05be20ec6c798c` |
| channel/schema | staging / 22 |
| classification mode | `suggest` |
| database | Task50 schema 21 匿名副本升级，独立 runtime root |

该实例没有复用 38091/schema 21 或 production/NAS 数据库。

## 3. 浏览器证据

浏览器自动化覆盖 375、390、430、1440 与 Fresh Light/Dark Glass 共 8 个组合，生成 38 张截图，`failure=null`。每个组合均满足：

1. document/body 宽度不超过 viewport，无横向滚动。
2. `aria-live` 状态区存在，键盘焦点路径可见。
3. 8 标签、长商户、冲突、规则 stale、批量成功、重新分类、默认 profile 和兜底替代可操作。
4. 登录后 page error 为 0；仅出现预期的初始未登录资源 `401`。

21 张关键截图已整理到 `docs/ui/figma/local-review/task53-v1.3-2026-07-20/`。截图来自真实页面，但属于 generated review artifact；线上 Figma 仍为 `not_verified`。

## 4. 自动化验证

本次 Task53 候选重新执行并通过：

```text
backend: go test ./... -count=1
backend: go vet ./...
backend: go build ./cmd/server
frontend: npm run lint
frontend: npm test -- --run (40 files / 159 tests)
frontend: npm run build
contract: docs/api/openapi.yaml 72 paths / 372 local refs / 0 missing
contract: category-tag draft 12 paths / 109 refs / 0 missing
```

前端 build 仍有已知 746.09 kB chunk 提示，不阻断 Task53U。

## 5. Decision

Task53U 本地实现与视觉门禁关闭。下一步进入 Task53.5 发布收口；线上 Figma 节点复核和 NAS 部署保持独立状态，不反向阻断本地 UI 验收。
