# LedgerTwo 文档总入口

状态：当前文档入口
最近更新：2026-07-20

本文档用于判断 `docs/` 下哪些资料是当前事实源、哪些只是历史参考或验收证据。后续产品、架构、UI/UX、部署和 AI 开发任务都应先从这里进入，再按具体模块读取。

## 文档目录

```text
docs/
  prd/              当前产品定位、路线图、模块 PRD、验收口径
  tech/             当前架构、模块技术方案、部署与 migration 策略
  ui/               UI/UX 规范、页面流程、Fresh Light 和 Figma 配套资料
  api/              API inventory、OpenAPI 草案和接口契约冻结资料
  releases/         v1.2 发布说明、升级回滚说明和发布验收清单
  codex_tasks/      Task31+ 与版本冻结阶段的 AI/Codex 任务入口
  fixtures/         导入、验收和测试用例所需的非敏感样例说明
  project_analysis/ 阶段分析、验收证据、截图记录和历史归档
```

## 文档角色

后续判断文档时按以下角色处理，避免旧资料覆盖当前事实：

| 角色 | 目录或文件 | 使用规则 |
|---|---|---|
| 当前入口 | `docs/README.md`、`docs/00-文档索引.md` | 先读，用于确定阶段、阅读顺序和事实源优先级 |
| 产品事实源 | `docs/prd/README.md`、`docs/prd/00-产品定位与版本路线.md`、`docs/prd/20-35` | 当前产品定位、路线、范围和验收口径 |
| 技术事实源 | `docs/tech/README.md`、`docs/tech/00-Task30后当前架构.md`、`docs/tech/18-27` | 当前架构、实施契约、部署隔离和迁移策略 |
| UI/UX 事实源 | `docs/ui/README.md`、`docs/ui/14-17`、`docs/ui/figma/README.md` | 当前页面流程、长期体验专项、Figma 配套规范 |
| 任务入口 | `docs/codex_tasks/README.md` 和当前任务文件 | 只用于执行已确认任务，不替代 PRD/Tech/UI 事实源 |
| 发布证据 | `docs/releases/`、`docs/project_analysis/2026-*` | 记录已执行验收和发布状态，不单独定义新需求 |
| 历史资料 | 根目录早期 `01-18` 文档、`project_analysis/extracted_archives`、旧 zip | 背景参考；不得用于推翻 Task30 后的新能力 |

`docs/ui/figma/` 是当前 UI/UX 体系的一部分，不属于普通历史附件。该目录下的 Token、Variables、Frame Manifest、handoff 和本地审阅包必须按 `docs/ui/figma/README.md` 的事实源层级使用；本地预览不能反向覆盖已冻结的金额、权限、结算、导入和备份规则。

## 文件命名规范

1. 正式文档和人读型文档资产统一使用中文语义名称，如 `14-后端模块实现规范.md`；`LedgerTwo`、版本号、`Task`、`API`、`NAS`、`Figma` 等不可替代的产品或技术专名可以保留。
2. 有固定阅读顺序的文档保留两位数字前缀 `NN-`；阶段分析和评审报告优先使用 `YYYY-MM-DD-` 日期前缀；文件名不使用空格，元信息与中文标题之间使用半角连字符。
3. `README.md`、`openapi*.yaml`、程序脚本、结构化 fixture、Figma 机器资产和工具约定文件遵循各自生态命名，不为追求中文化而破坏工具契约。
4. `docs/project_analysis/extracted_archives/` 内的历史快照保持原始文件名；外部正式文档不得继续把快照路径当作当前事实源。
5. 重命名正式文档时，必须同步更新根 `README.md`、`AGENTS.md`、各目录 `README.md`、文档索引、任务入口及其他有效交叉引用，并检查旧路径残留。

## 当前总览文档

当前已经存在产品和技术总览，不需要再新增平行的总览文档：

| 类型 | 总览入口 | 说明 |
|---|---|---|
| 全局文档入口 | `docs/README.md`、`docs/00-文档索引.md` | 判断阶段、阅读顺序和历史/当前边界 |
| PRD 总览 | `docs/prd/README.md`、`docs/prd/00-产品定位与版本路线.md` | 产品定位、版本路线、优先级和 PRD 事实源 |
| 技术总览 | `docs/tech/README.md`、`docs/tech/00-Task30后当前架构.md` | 架构现状、技术栈、模块边界和技术事实源 |
| UI/UX 总览 | `docs/ui/README.md`、`docs/ui/15-LedgerTwo长期体验优化专项.md`、`docs/ui/figma/README.md` | 页面流程、长期 UI/UX 专项和 Figma 规范 |

后续如需新增文档，应优先补充到对应目录的 README 或现有总览中；只有新版本、新模块或新发布窗口无法被现有总览承载时，才新增独立正式文档。

## 推荐阅读顺序

1. `../CHANGELOG.md` (版本发布说明)
2. `docs/releases/README.md`
3. `docs/prd/README.md`
4. `docs/tech/README.md`
5. `docs/ui/README.md`
6. `docs/prd/00-产品定位与版本路线.md`
7. `docs/prd/20-产品复盘与定位重整.md`
8. `docs/prd/21-短中长期产品路线图.md`
9. `docs/prd/22-v1.1可信赖与高频记账.md`
10. `docs/prd/23-功能优先级与延后决策.md`
11. `docs/prd/24-短中期模块拆解.md`
12. `docs/prd/25-v1.1模块需求规格.md`
13. `docs/prd/26-v1.2导入与省时模块规格.md`
14. `docs/prd/29-v1.2导入模块业务与服务细分.md`
15. `docs/prd/30-v1.2微信XLSX与支付宝CSV导入专项.md`
16. `docs/prd/27-v1.1至v1.2验收样例矩阵.md`
17. `docs/prd/28-交易与账户口径补充.md`
18. `docs/tech/00-Task30后当前架构.md`
19. `docs/tech/18-短中期模块架构切片.md`
20. `docs/tech/23-v1.2部署环境与数据库隔离.md`
21. `docs/tech/24-v1.2-XLSX导入专项实施方案.md`
22. `docs/api/API清单.md`
23. `docs/api/API规范.md`
24. `docs/api/openapi.yaml`
25. `docs/ui/14-v1.1至v1.2模块流程.md`
26. `docs/ui/15-LedgerTwo长期体验优化专项.md`
27. `docs/ui/figma/README.md`
28. `docs/codex_tasks/README.md`
29. `docs/codex_tasks/12-v1.2微信XLSX与支付宝CSV导入专项计划.md`
30. `docs/codex_tasks/13-Fresh-Light界面交互协同开发计划.md`
31. `docs/codex_tasks/14-v1.3-Task50多账本开发前计划.md`
32. `docs/codex_tasks/15-v1.3-Task50多账本详细实施计划.md`
33. `docs/codex_tasks/16-Task50.3至Task50.6准入与后续入口.md`
34. `docs/prd/34-v1.3-Task53分类标签与导入智能归类.md`（Task53.1-Task53.3 已完成，Task53.4 准备关闭）
35. `docs/prd/35-竞品分析需求补足与中长期计划.md`（中长期产品规划和竞品能力输入）
36. `docs/tech/26-v1.3-Task53分类标签智能化实施契约.md`
37. `docs/tech/27-Task53-Schema22迁移评审.md`
38. `docs/api/openapi-v1.3-category-tag-draft.yaml`
39. `docs/ui/17-v1.3分类标签与导入智能归类流程.md`
40. `docs/fixtures/category-tag/README.md`
41. `docs/codex_tasks/18-Task53分类标签智能化开发前计划.md`
42. `docs/ui/figma/task53-v1.3-category-tag/README.md`
43. `docs/codex_tasks/19-v1.3-Task53分类标签智能化详细实施计划.md`
44. `docs/project_analysis/2026-07-16-Task53分类标签智能化开发前准备.md`
45. `docs/prd/33-Task51多人分摊场景证据与范围问题.md`（仅 Task51 非约束性发现准备）
46. `docs/project_analysis/2026-07-16-Task50准备完整度与Task51P.1启动评审.md`
47. `docs/project_analysis/task51_p1/README.md`（Task51P.1 匿名证据工作区）
48. `docs/codex_tasks/17-Task51多人分摊开发前计划.md`（Task50 技术门禁已满足，正式范围仍等待真实证据）
49. 进入具体业务模块文档。

当前项目已完成 Task01-Task49。Task49X 核心实现、运行开关、本机 schema 19、微信 XLSX/支付宝 CSV 真实 preview 和移动端视觉验收已完成；支付宝当前仍只导出 CSV。后续发布收口聚焦 NAS schema 19 staging、production 一致性备份与逐批导入确认，开发入口以 `docs/project_analysis/2026-07-12-本机WSL真实账单预览验收.md`、`docs/codex_tasks/12-v1.2微信XLSX与支付宝CSV导入专项计划.md`、专项 PRD/DEV 为准。

2026-07-17 更新：Task50.1-Task50.6 已完成，独立本机 v1.3/schema 21 staging、回滚和浏览器证据已闭环，NAS 未部署。Task53.1-Task53.3 已在本地完成 schema 22、默认元数据、确定性分类器、preview 分类快照和 reclassify；Task53.4 准备关闭，下一实现切片为 Task53.4A；Task51P.1 真实证据仍为 0，Task52 继续保持调研门禁。

2026-07-20 更新：Task53.1-Task53U 与 Task53.5 已关闭，决策为 `pass_with_suggest_only`。NAS-R1 已把 38088 重建为 schema 22 production、把 38092 重建为 schema 22 staging，并下线旧 38089。38088 已初始化且 NAS 内外基线备份通过校验，真实数据保全规则已生效；当前仅待用户确认两个账号均可从 LAN 登录。事实源为 `docs/codex_tasks/20-NAS-R1真实体验发布与数据保全计划.md`、`docs/tech/29-NAS环境分级与真实数据保全契约.md` 和最新 NAS-R1 执行记录。

## AI 开发使用方式

让 AI 编码时，不要让它一次性实现全项目。推荐提示：

```text
请先阅读 docs/README.md、docs/prd/README.md、docs/tech/README.md，
然后只实现【某一个模块】。输出计划后等待确认，不要直接开始大范围修改。
```
