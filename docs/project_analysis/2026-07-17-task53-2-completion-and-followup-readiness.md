# Task53.2 完成与后续准备评审

状态：Task53.2 本地实现和验证完成，待原子提交<br>
日期：2026-07-17<br>
部署状态：未更新 WSL/NAS，未读取或修改真实账单

## 1. Task53.2 delivered scope

1. 新增纯 `classifier`：NFKC/小写/trim/空白折叠、规则与 built-in candidate、确定性 resolver、fallback、冲突和 8 标签上限。
2. 手工和批量结果受保护；duplicate、invalid、skipped 不参与分类；历史 `suggest` 规则仍不写 selected。
3. repository 可一次读取当前账本 active rules、分类、标签和账户；跨账本、归档、类型错配或非法 `result_json` 不进入有效候选。
4. built-in v1 只包含匿名通用外卖、工资和退款词条，只产生 medium 建议；显式用户/学习建议存在时，built-in 仅保留诊断候选，不混入可接受标签。
5. `IMPORT_CLASSIFICATION_MODE=off|suggest|graded` 默认 `off`，health 暴露当前值；Task53.2 不接入 preview/commit。

## 2. Evidence

TDD 已覆盖 CT-R01-CT-R12、规范化、同输入 100 次稳定、规则优先级、同级冲突、历史 suggest、built-in 负例、跨账本/归档/非法类型元数据、8 标签上限、repository 查询和配置默认值。

提交前验证结果：

1. Qt MinGW CGO 环境下 `go test ./... -count=1`、`go vet ./...` 和 server build 通过。
2. 前端 ESLint、38 个测试文件/147 项测试和 production build 通过；仅保留既有大 chunk warning。
3. OpenAPI/compose YAML、Figma manifest JSON、OpenAPI `$ref`/path parameter 和 15 Frame/source checks 通过。
4. `git diff --check` 通过。
5. 当前 Windows shell 无 Docker CLI，因此没有运行 `docker compose config`；只完成 YAML 解析，未启动 38092、WSL 或 NAS。

## 3. Follow-up readiness

| Task | Preparation | Decision |
|---|---|---|
| Task53.3 | PRD、Tech、OpenAPI 草案、Fixture、持久化/开关/重分类边界已补齐 | Task53.2 提交后可开发 |
| Task53.4 | bulk、learn、metadata safeguard、错误和事务边界已补齐 | 等待 Task53.3 DTO 落盘后开发 |
| Task53U | UI 17、15 Frame manifest、状态矩阵、代码复用证据完整 | 等待 53.3/53.4 DTO 和本地视觉审阅稿 |
| Task53.5 | 38092 独立 staging compose/env、flag 演练和验收顺序已定义 | 等待全部实现，不得当前部署 |

因此“后续准备”不是全部无条件关闭：后端 53.3/53.4 的开发输入已经完整；53U/53.5 仍保留合理的实现依赖，不能提前伪造视觉或部署证据。

## 4. Task51 decision

Task53 完成后不等于 Task51 自动开工。Task51 的准备阶段 `Task51P.1` 早已启动，方法、匿名模板和 3/5 人假设 Fixture 齐全，但当前：

```text
valid_group_records=0
complete_workflow_replays=0
decision=pending
```

Task53.5 关闭后的下一步是重新评审 Task51P.1 证据，而不是直接创建 Task51 migration 或解除两人约束。只有真实匿名证据满足门槛并得出 `continue/narrow`，才顺序进入 P2-P6；材料不足默认 `defer`。

## 5. Next atomic task

下一实现任务为 Task53.3。开工顺序固定为：DTO/持久化 snapshot failing tests -> classification context 单次读取 -> preview 接入与解释持久化 -> summary -> reclassify dry-run/execute -> OpenAPI 与前端类型同步。Task53U 和 Task51 代码均不与该切片并行修改共享导入类型。
