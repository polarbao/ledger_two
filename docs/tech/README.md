# 技术文档模块目录

状态：当前技术事实源入口
最近更新：2026-07-20

本目录按照工程模块拆分 LedgerTwo 的技术设计与实现方案。

## 总览判断

当前技术文档已有总览文档，不需要新增平行的架构总览：

| 层级 | 文件 | 作用 |
|---|---|---|
| 当前架构总览 | `00-Task30后当前架构.md` | Task30 后后端、前端、数据库、部署和风险总览 |
| 短中期架构切片 | `18-短中期模块架构切片.md` | v1.1/v1.2 后端、前端、数据模型、API、服务层和测试切片 |
| 实施就绪评审 | `19-短中期实施就绪评审.md` | 判断文档充分性、执行顺序和不建议启动的任务 |
| 导入实施契约 | `20-v1.2导入模块实施契约.md` | v1.2 导入 API、状态机、DTO、权限、数据模型和回滚策略 |
| 部署隔离总览 | `23-v1.2部署环境与数据库隔离.md` | development/staging/production 物理隔离、发布顺序和运行开关 |
| XLSX 专项方案 | `24-v1.2-XLSX导入专项实施方案.md` | Task49X reader、migration 019、安全、测试和回滚 |
| Task50 技术契约 | `25-v1.3-Task50多账本实施契约.md` | 多账本 lifecycle、LedgerContext、实例管理员、migration 020/021 和回滚 |
| Task53 技术契约 | `26-v1.3-Task53分类标签智能化实施契约.md`、`27-Task53-Schema22迁移评审.md`、`28-Task53.3至Task53.5开发与发布准备契约.md` | 分级自动化、schema 22、后续 DTO/事务、隔离 staging 和回滚 |
| NAS 环境与数据 | `29-NAS环境分级与真实数据保全契约.md` | 38088/38092 分级、公网边界与 production 数据保全 |

后续技术规划应优先更新这些总览和契约。只有当 v1.3 新能力完成 PRD 范围冻结，且现有总览无法承载新的架构边界时，才新增独立技术总览或 ADR。

## 当前技术阶段

截至 2026-07-15，当前架构判断如下：

1. Go + SQLite + REST JSON + React/Vite 的总体选型继续成立，不需要整体推倒重写。
2. `development`、`staging`、`production` 应继续通过部署实例、物理目录、端口、密钥、数据库文件隔离；不能为了“统一部署”共享数据目录。
3. v1.2 RC 的关键技术门禁是 schema 19 staging、XLSX 开关、备份链、health 校验和回滚脚本。
4. Fresh Light 属于前端体验专项，不应改变后端金额、权限、导入、结算或 migration 契约。
5. v1.3 前应重新评审多账本、多成员、多人分摊的数据模型、权限矩阵和 migration 策略。
6. UI-FL-10、Task50P.1-P.6、Task50.1-Task50.6 与 Task53.1-Task53U 已完成；Task53 包含 schema 22、默认元数据、确定性 preview/reclassify、bulk-adjust、explicit learn、规则生命周期、metadata safeguard 和双主题 UI。
7. Task53 migration、OpenAPI、Fixture、浏览器和发布门禁已形成闭环；commit 固定读取持久化快照且不重分类，WSL2 结论为 `pass_with_suggest_only`。
8. WSL 38091 保留 Task50 schema 21 候选；NAS-R1 已把固定 Task53 schema 22/suggest 候选分别部署到 NAS 38088 production 和 38092 staging。38088 已初始化并建立 NAS 内外基线备份，数据保全规则已生效；38092 保持未初始化的可重建 QA 环境。

当前已知技术债：

1. 交易表单、交易 service/repository、导入工作台仍需按领域逐步拆分。
2. API/OpenAPI、实际 handler、前端类型需要随每个版本继续同步。
3. 大包体和前端分包是性能专项，不应混入业务发布门禁。
4. 旧 Demo 文档和历史压缩包不能作为当前实现依据。

## 文件列表

```text
01-总体架构与技术选型.md       总体架构与技术选型
02-后端模块设计.md          后端模块设计
03-前端模块设计.md         前端模块设计
04-数据库与API设计.md             数据库与 API 设计
05-分摊与结算算法.md     分摊与结算算法
06-导入导出与备份恢复.md     导入、导出、备份恢复
07-跨端技术方案.md      跨端技术方案
08-NAS部署方案.md           NAS 部署方案
23-v1.2部署环境与数据库隔离.md v1.2 staging/production 与数据库物理隔离
09-测试与质量保障.md             测试与质量保障
13-v1.1前基础框架总览.md Foundation before v1.1 技术方案
14-配置安全与部署一致性.md 配置、安全与部署
15-账本上下文与RBAC权限框架.md      LedgerContext 与 RBAC
16-API契约OpenAPI与错误码规范.md API 契约、OpenAPI 与错误码
17-数据迁移测试与质量门禁.md 数据迁移、测试与质量门禁
18-短中期模块架构切片.md 短中期模块架构切片
19-短中期实施就绪评审.md 短中期实施就绪评审
20-v1.2导入模块实施契约.md v1.2 导入模块实施契约
21-v1.2导入模块迁移评审.md v1.2 导入模块 Migration 评审
22-v1.2-Task47导入预览实施计划.md v1.2 Task47 导入预览实施计划
24-v1.2-XLSX导入专项实施方案.md v1.2 Task49X XLSX 导入专项实施方案
25-v1.3-Task50多账本实施契约.md v1.3 Task50 多账本技术与 Migration 冻结契约
26-v1.3-Task53分类标签智能化实施契约.md Task53 分类/标签/导入智能归类技术契约
27-Task53-Schema22迁移评审.md Task53 schema 21 -> 22 Migration 评审
28-Task53.3至Task53.5开发与发布准备契约.md Task53.3-Task53.5 flag、DTO、事务、UI 与隔离 staging 准备契约
29-NAS环境分级与真实数据保全契约.md NAS 38088/38092 环境角色、公网测试与 production 数据保全
```

## 技术原则

- 后端：Go + SQLite + REST JSON。
- 前端：React + TypeScript + Vite。
- 金额：统一 int64 cents，禁止 float。
- 结算：只生成 settlement 记录，不修改历史账单。
- 删除：soft delete。
- 统计：以后端聚合为准，前端只展示。
- 部署：staging/production 必须物理隔离，schema 与镜像成对升级和回滚。
- UI：Figma 和 Fresh Light 只能约束表现层，不得覆盖金额、权限、导入、结算和备份契约。

## 冲突处理

技术文档发生冲突时按以下顺序判断：

1. 当前代码、migration、测试和已执行命令结果。
2. 最新发布/验收记录。
3. 本目录 `00`、`18-24` 当前技术事实源。
4. `docs/prd/` 当前 PRD 与验收口径。
5. `docs/ui/` 当前 UI/UX 规范。
6. 早期 `01-17` 技术文档和根目录 Demo 文档。

## 当前推荐入口

Task30 后的技术规划建议优先阅读：

1. `00-Task30后当前架构.md`
2. `13-v1.1前基础框架总览.md`
3. `18-短中期模块架构切片.md`
4. `19-短中期实施就绪评审.md`
5. `20-v1.2导入模块实施契约.md`（进入 Task47-Task49 前必读）
6. `21-v1.2导入模块迁移评审.md`（进入 Task47-Task49 前用于确认 migration 切片）
7. `22-v1.2-Task47导入预览实施计划.md`（Task47 开工时用于确认 parser、repository、service、handler 和前端切片）
8. `../prd/29-v1.2导入模块业务与服务细分.md`（进入 Task47-Task49 前用于确认业务对象、服务边界和 UI 工作台）
9. `24-v1.2-XLSX导入专项实施方案.md`（Task49X 开发前用于确认依赖、reader、migration 019、安全和测试边界）
10. `25-v1.3-Task50多账本实施契约.md`（Task50 开发前用于确认 lifecycle、实例管理员、migration 020/021 和回滚）
11. `26-v1.3-Task53分类标签智能化实施契约.md`、`27-Task53-Schema22迁移评审.md` 与 `28-Task53.3至Task53.5开发与发布准备契约.md`（当前 Task53 技术评审入口）
