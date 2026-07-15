# Fresh Light 波次 C 收口与波次 D 准备度评审

状态：UI-FL-06/07 已完成，波次 C 关闭；UI-FL-08 准入已满足<br>
日期：2026-07-14；2026-07-15 收口复核

## 1. 结论

UI-FL-06（结算）和 UI-FL-07（设置与元数据）均已完成源码、自动化、本机固定镜像和真实账号只读路径验收。UI-FL-07 未新增 API、migration 或依赖，设置页文件所有权已经释放给后续 Task50 设计阶段，但 Task50 仍不得在开发准入前编码。UI-FL-08 的 Task49X 契约和共享列表外壳前置条件已满足，可进入波次 D。

Fresh Light 目前仍是逐页迁移状态，默认主题继续为 Dark Glass。全局默认切换只能在 UI-FL-10 完成全部页面、Dark Glass 回退和真实业务验收后执行。

## 2. UI-FL-06 准备项

关联事实源：Task43/45、结算 PRD、`SettlementPage`、共享 `ConfirmDialog` 和 balance/settlement API。

实施切片：

1. 06A：盘点并冻结 `paid/share/raw_net/settlement/final_net` 文案和展示层级。
2. 06B：迁移范围切换、转账行动、复制文案和登记确认，不修改结算计算。
3. 06C：回归“登记生成 settlement、不改历史共同支出”、复制失败兜底和 375/390/430/1440 响应式。

收口结果：复用现有 ConfirmDialog/Button/SegmentedControl；最终余额直接消费后端 DTO；登记前说明影响，登记后刷新 balance、settlements、dashboard、transactions 和 reports。复制失败兜底、375/1440px 和隔离数据库真实登记均通过，证据见 `docs/project_analysis/ui-fl-06-runtime-2026-07-15/`。

## 3. UI-FL-07 准备项

关联事实源：Task32/38/39、设置安全验收、分类/标签/账户生命周期、备份恢复和诊断契约。

实施切片：

1. 07A：设置导航和账号/账本/角色信息分区。
2. 07B：分类、标签、账户、模板和周期规则的新增、编辑、归档、恢复及历史引用提示。
3. 07C：导出、备份、恢复、诊断和高风险操作确认；viewer 隐藏管理入口且后端继续拒绝越权。

准入门禁：不得将归档改成删除；恢复和备份继续使用现有 API；危险动作必须写清影响范围；开发、staging、production 数据目录和数据库保持物理隔离。

## 4. 协同与文件所有权

- UI-FL-06 拥有结算页面和结算确认组件，不修改交易表单或导入工作台。
- UI-FL-07 拥有设置页及元数据管理组件；共享 Button、ConfirmDialog、BottomSheet 只复用，不平行创建同类组件。
- UI-FL-08/09 准备事实已存在，但在波次 C 期间不并行修改 `ResponsiveDataList` 和交易筛选契约。
- 任何业务缺陷优先于视觉迁移；若发现 API 或权限口径缺口，退出 UI-FL 页面任务并单独评审。

## 5. 进入条件

1. UI-FL-05E 源码与文档已独立提交。
2. 固定后端镜像完成 handler 合约测试和真实账号编辑回归。
3. 分别为 06、07 建立目标文件清单、测试矩阵和截图目录。
4. 保留 Dark Glass 回退，页面迁移不等于全局默认主题切换。

2026-07-15 收口复核：以上条件均已满足。UI-FL-07 证据见 `docs/project_analysis/ui-fl-07-settings-metadata-2026-07-15/`。波次 C 关闭，下一步执行 UI-FL-08；Task50 只并行开展准备文档，不进入业务代码或 migration。
