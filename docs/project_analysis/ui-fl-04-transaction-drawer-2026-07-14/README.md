# UI-FL-04 记账抽屉验收记录

日期：2026-07-14

任务：`docs/codex_tasks/13-fresh-light-ui-interaction-plan.md` / UI-FL-04

## 1. 验收结论

UI-FL-04 已完成记账抽屉的 Fresh Light 迁移。金额、类型、分类、账户、日期保留为高频路径；共同账单在提交前展示付款人、分摊方式、参与状态和承担金额；标题、标签、可见性、备注、附件和模板进入低频区。既有普通/共同账单 API、默认值、复制、模板、草稿、附件上传和 query invalidation 未改变。

## 2. 实现范围

- `transactionFormState.ts`：低频区展开判定和共同支出承担预览；金额使用整数分，均分余数归付款人。
- `SharedExpensePreview.tsx`：只展示成员名称、付款/参与语义和金额，不暴露用户 ID。
- `TransactionFormFooter.tsx`：普通、复制、草稿、离线四种主动作，保留“保存并继续”。
- `TransactionFormDrawer.tsx`：高低频编排、桌面右抽屉、移动 Bottom Sheet、脏字段关闭确认、焦点圈定与返回。
- `TransactionFormDrawer.css`：只使用语义 Token，不新增页面级渐变、颜色常量、断点或第三方依赖。

## 3. 视觉证据

| 文件 | 状态 | 视口 |
|---|---|---|
| `fresh-light-entry-1440.png` | 桌面共同账单，高低频区同屏 | 1440 x 1000 |
| `fresh-light-entry-shared-390.png` | 移动共同账单，承担预览与固定 Footer | 390 x 844 |
| `fresh-light-entry-advanced-430.png` | 移动更多选项，模板/标题/标签/备注 | 430 x 900 |
| `fresh-light-entry-discard-375.png` | 移动脏字段放弃确认 | 375 x 812 |
| `metrics.json` | CDP 宽度、横向滚动和验证结果 | JSON |

截图使用真实 `TransactionFormDrawer`、共享 UI 组件、Fresh Light Token、确定性 Query 响应和本地 Zustand 状态生成。临时预览入口和 CDP 脚本已删除；这些证据验证组件布局与状态，不替代真实后端创建、附件上传、模板写入、离线恢复或跨页 E2E。

## 4. 验证结果

```text
corepack pnpm test
16 test files passed, 59 tests passed

corepack pnpm lint
passed

corepack pnpm build
passed
```

Chrome 150 CDP 指标显示 375/390/430/1440 四个视口的 `innerWidth` 均等于页面 `scrollWidth`，抽屉 `clientWidth` 与 `scrollWidth` 也一致。截图已人工检查：金额和承担金额未裁切，分摊选项可读，390px Footer 无横向溢出，375px 危险确认不遮挡自身动作。

生产构建主 JavaScript chunk 约 664 kB，仍超过 500 kB。UI-FL-04 未新增依赖，该告警继续归属后续性能专项。

本机构建产物已同步到现有 WSL staging 的静态目录，未重建后端、未执行 migration、未修改数据库。`http://localhost:38088/api/healthz` 回读为 `staging / schema 19 / db ok / import_xlsx_enabled=true`，首页引用本轮产物 `assets/index-DnSyg_LY.js`。

## 5. 业务保持说明

- 普通收支仍调用 `createTransaction`，共同支出仍调用 `createSharedExpense`，API 金额继续使用整数分。
- 共同支出预览只复刻服务端 `equal/payer_only` 已有规则并声明服务端权威，不新增分摊方法或参与人编辑。
- 保存并继续仍清空金额、标题、备注和附件，保留分类、账户、付款人、标签、可见性及分摊方式。
- 复制生成新账单且不修改来源；离线只写本地草稿；viewer 仍不可新增或提交。
- 关闭确认只保护未保存表单和上传进度，不改变模板、草稿或正式账单生命周期。

## 6. 后续关系

UI-FL-05 可复用已稳定的抽屉入口继续迁移流水工作台。Task49X 仍只剩支付宝真实 XLSX、线上 Figma 主文件和 NAS staging schema 19 等外部门禁；UI-FL-08 最终收口继续受该边界约束，本次未访问 NAS 或提交真实导入批次。
