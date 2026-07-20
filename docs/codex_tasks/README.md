# Codex / Gemini 开发任务入口

状态：当前任务入口
适用阶段：Task01-Task50 已完成；Fresh Light UI-FL-01 至 UI-FL-10 已完成；Task53.1-Task53U 与 Task53.5 的 WSL2/NAS staging 验收已完成，发布决策为 `pass_with_suggest_only`；NAS-R1 双环境重建和 production 基线备份已完成，当前仅待用户确认两个真实账号均可从 LAN 登录，随后返回 Task51P.1

## 1. 目标

本目录用于给 Codex、Gemini、Cursor、Copilot 或其他 AI 编码工具提供明确、可执行、可验收的开发任务和代码风格规范。

后续所有 AI 开发任务都应从本目录开始，而不是直接让 AI 阅读零散文档后自由发挥。

## 2. 文件列表

```text
00-AI开发工作流.md   AI 开发工作流和通用提示词
01-仓库通用代码规范.md     仓库通用代码风格和提交规范
02-Go后端代码规范.md          Go 后端代码风格
03-React与TypeScript前端代码规范.md   React + TypeScript 前端代码风格
04-测试与质量门禁.md     测试与质量门禁
05-Foundation任务计划.md      Task31-Task40 基础框架任务计划
06-人工审核清单.md          人类审核清单
07-代码风格参考来源.md  代码风格参考来源
08-产品路线图开发任务计划.md  产品路线对应的 Task41+ DEV 任务计划
09-Task41至Task49详细开发计划.md   Task41-Task49 细化开发任务规格
10-Task33至Task40详细开发计划.md   Task33-Task40 细化开发任务规格
11-v1.2发布加固计划.md v1.2 RC 环境隔离、NAS staging 与 production 升级任务
12-v1.2微信XLSX与支付宝CSV导入专项计划.md v1.2 微信 XLSX/支付宝 CSV 导入专项任务
13-Fresh-Light界面交互协同开发计划.md v1.2 收口后的 Fresh Light UI/UX 协同开发计划
14-v1.3-Task50多账本开发前计划.md v1.3 Task50 多账本正式化开发前准备与准入计划
15-v1.3-Task50多账本详细实施计划.md v1.3 Task50.1-Task50.6 详细实施、验证、回滚与提交计划
16-Task50.3至Task50.6准入与后续入口.md Task50.3-Task50.6 准入、Task51P 与 Task52 后续边界
17-Task51多人分摊开发前计划.md Task51P.1-P.6 开发前准备顺序与正式门禁
18-Task53分类标签智能化开发前计划.md Task53P 分类、标签、默认元数据与分级自动化准备门禁
19-v1.3-Task53分类标签智能化详细实施计划.md Task53.1-Task53.5/Task53U 详细实施、TDD、环境、回滚与提交计划
20-NAS-R1真实体验发布与数据保全计划.md NAS-R1 真实体验 production 一次性重置、staging 清理与长期数据保全计划
../tech/28-Task53.3至Task53.5开发与发布准备契约.md Task53.3-Task53.5 flag、DTO、事务、UI 与隔离 staging 准备契约
../prd/31-v1.3-Task50多账本正式化.md v1.3 Task50 多账本正式化冻结 PRD
../prd/32-v1.3-Task50验收样例矩阵.md v1.3 Task50 匿名 Fixture 与验收矩阵
../prd/33-Task51多人分摊场景证据与范围问题.md Task51 多人分摊场景证据与范围问题
../project_analysis/task51_p1/README.md Task51P.1 匿名证据登记、记录模板与假设回放工作区
../prd/34-v1.3-Task53分类标签与导入智能归类.md Task53 分类标签智能化评审 PRD
../api/openapi-v1.3-ledger-draft.yaml v1.3 Task50 API 开发前冻结草案
../api/openapi-v1.3-category-tag-draft.yaml Task53 元数据模板、批量调整、学习与重分类 API 草案
../tech/26-v1.3-Task53分类标签智能化实施契约.md Task53 分级自动化、默认元数据和兼容契约
../tech/27-Task53-Schema22迁移评审.md Task53 migration 022 开发前评审
../ui/17-v1.3分类标签与导入智能归类流程.md Task53 Fresh Light 导入与元数据交互流程
../fixtures/category-tag/ Task53 匿名确定性 Fixture 说明
../ui/figma/task53-v1.3-category-tag/ Task53 本地 Figma handoff、Frame Manifest 与组件状态矩阵
../ui/16-v1.3-Task50多账本交互流程.md v1.3 Task50 Fresh Light 交互流程
../ui/figma/task50-v1.3-multi-ledger/ v1.3 Task50 本地 Figma handoff 与 Frame Manifest
../releases/                    v1.2 发布说明、升级回滚和发布验收清单
```

## 3. AI 开发强制流程

1. 读取 `docs/00-文档索引.md`。
2. 读取 `docs/prd/11-v1.1前基础框架补齐.md`。
3. 产品规划类任务读取 `docs/prd/20-产品复盘与定位重整.md` 到 `docs/prd/23-功能优先级与延后决策.md`。
4. 短中期业务开发读取 `docs/prd/24-短中期模块拆解.md`、`docs/prd/25-v1.1模块需求规格.md`、`docs/prd/26-v1.2导入与省时模块规格.md`。
5. 短中期冻结或开发前读取 `docs/prd/27-v1.1至v1.2验收样例矩阵.md` 和 `docs/prd/28-交易与账户口径补充.md`。
6. v1.2 Task47-Task49 开发前额外读取 `docs/prd/29-v1.2导入模块业务与服务细分.md` 和 `docs/tech/20-v1.2导入模块实施契约.md`。
7. Task49X 开发前必须读取 `docs/prd/30-v1.2微信XLSX与支付宝CSV导入专项.md`、`docs/tech/24-v1.2-XLSX导入专项实施方案.md` 和 `docs/codex_tasks/12-v1.2微信XLSX与支付宝CSV导入专项计划.md`。
8. 读取 `docs/tech/18-短中期模块架构切片.md`、`docs/tech/19-短中期实施就绪评审.md` 和 `docs/ui/14-v1.1至v1.2模块流程.md`。
9. Fresh Light 或后续业务 Task 涉及 UI 时，读取 `docs/ui/figma/2026-07-13-Fresh-Light界面实施规范.md` 和 `docs/codex_tasks/13-Fresh-Light界面交互协同开发计划.md`，登记共享组件归属和并行冲突。
10. Task50.1-Task50.6 已关闭，最终证据为 `docs/project_analysis/2026-07-17-Task50.6全模块与发布收口验收.md`；不得重复执行 schema 21 migration 或把本机 staging 误写为 NAS 已部署。
11. Task53 后续开发必须依次读取 `docs/prd/34-v1.3-Task53分类标签与导入智能归类.md`、`docs/tech/26-v1.3-Task53分类标签智能化实施契约.md`、`docs/tech/27-Task53-Schema22迁移评审.md`、`docs/tech/28-Task53.3至Task53.5开发与发布准备契约.md`、`docs/api/openapi-v1.3-category-tag-draft.yaml`、`docs/ui/17-v1.3分类标签与导入智能归类流程.md`、`docs/ui/figma/task53-v1.3-category-tag/README.md`、`docs/ui/figma/task53-v1.3-category-tag/Figma画板复用证据.md`、`docs/fixtures/category-tag/README.md`、`docs/codex_tasks/18-Task53分类标签智能化开发前计划.md` 和 `docs/codex_tasks/19-v1.3-Task53分类标签智能化详细实施计划.md`；不得重复建立平行准备文档。
12. Task51 准备必须读取 `docs/prd/33-Task51多人分摊场景证据与范围问题.md`、`docs/codex_tasks/17-Task51多人分摊开发前计划.md` 与 `docs/project_analysis/task51_p1/README.md`；P1 真实证据未形成 `continue/narrow` 前不得实现代码或解除两人约束。
13. Foundation 开发读取 `docs/codex_tasks/10-Task33至Task40详细开发计划.md`。
14. 读取本目录代码风格文档。
15. 读取对应任务。
16. 输出计划和预计修改文件，等待确认。
17. 只实现当前任务。
18. 运行测试和构建。
19. 输出变更摘要、验证命令、风险和下一步建议。

## 4. 禁止事项

1. 禁止一次性实现多个 Foundation Task。
2. 禁止实现未审核 v1.1 业务需求。
3. 禁止把权限判断只放在前端。
4. 禁止使用 float 计算金额。
5. 禁止修改历史 migration。
6. 禁止提交真实数据库、备份、上传文件和密钥。
7. 禁止绕过测试直接声称完成。
