# Task53 Figma Frame 本地复用证据

状态：代码结构复用已核对；视觉审阅稿待 Task53.3/53.4 DTO 冻结后生成<br>
核对日期：2026-07-17<br>
产物分类：requirement / reuse evidence，不是 generated review artifact

## 1. Verified foundations

| Capability | Existing source | Reuse conclusion |
|---|---|---|
| 状态语义 | `frontend/src/components/ui/StatusChip.tsx` | 已有 neutral/success/info/warning/danger/accent，可增业务 label，不新增 chip 系统 |
| 模式切换 | `frontend/src/components/ui/SegmentedControl.tsx` | 复用筛选、profile 和规则分组 |
| 确认流程 | `frontend/src/components/ui/ConfirmDialog.tsx` | 复用批量确认和 reclassify 确认；分类选择不塞入危险确认弹窗 |
| 移动编辑 | `frontend/src/components/ui/BottomSheet.tsx` | 复用行编辑、同商户应用和替代分类移动容器 |
| 空/错状态 | `frontend/src/components/ui/StatePanel.tsx` | 复用无建议、规则失效和部分失败状态 |
| 桌面/移动列表 | `frontend/src/components/ui/ResponsiveDataList.tsx` | 复用 1440 表格和 375/390/430 卡片切换 |
| 导入工作台 | `frontend/src/pages/ImportPage.tsx` | 增量增加分类摘要、筛选、批量和重分类，不建立第二入口 |
| 预览行 | `frontend/src/components/import/ImportPreviewRows.tsx` | 增量增加最终分类、标签、状态和可读解释 |
| 行编辑器 | `frontend/src/components/import/ImportRowEditor.tsx` | 增加 N/8、remember merchant 与双结果反馈 |
| 规则管理 | `frontend/src/components/import/ImportRuleManager.tsx` | 增加来源、行为、精确商户和 stale 状态 |
| 元数据管理 | `frontend/src/pages/MetadataManagePage.tsx` | 增加 profile preview/apply、规则引用和 fallback replacement |
| 账本创建 | `frontend/src/pages/LedgerManagementPage.tsx` | 在 Task50 状态机上增加 metadata profile，不重写账本生命周期 |

`SettingsSection` 当前是 `SettingsPage.tsx` 内部组合，不应为了 Task53 强行抽成全局组件；Task53 设置入口优先沿用现有页面布局。

## 2. Frame mapping

| Manifest Frame | Reused code surface | Local status |
|---|---|---|
| Classification Context and State Matrix 1440 | StatusChip + Fresh Light tokens | reuse-confirmed, visual-pending |
| Import Classification Auto Selected Desktop 1440 | ImportPage + ImportPreviewRows + ResponsiveDataList | reuse-confirmed, visual-pending |
| Import Classification Suggestions Mobile 390 | ImportPreviewRows mobile cards | reuse-confirmed, visual-pending |
| Import Classification Fallback Narrow 375 | ImportPreviewRows + StatePanel | reuse-confirmed, visual-pending |
| Import Classification Conflict Mobile 390 | ImportPreviewRows + BottomSheet | reuse-confirmed, visual-pending |
| Import Row Editor Remember Merchant Mobile 390 | ImportRowEditor + BottomSheet | reuse-confirmed, visual-pending |
| Bulk Accept Suggestions Confirmation Mobile 390 | ImportPage + ConfirmDialog | reuse-confirmed, visual-pending |
| Apply Same Values and Same Merchant Desktop 1440 | ImportPage + ConfirmDialog/BottomSheet | reuse-confirmed, visual-pending |
| Reclassify Dry Run Desktop 1440 | ImportPage + ConfirmDialog + StatePanel | reuse-confirmed, visual-pending |
| Import Rule Manager Learned Rules Desktop 1440 | ImportRuleManager + SegmentedControl | reuse-confirmed, visual-pending |
| New Ledger Metadata Profile Mobile 390 | LedgerManagementPage + SegmentedControl | reuse-confirmed, visual-pending |
| Existing Ledger Profile Preview Desktop 1440 | MetadataManagePage + ResponsiveDataList | reuse-confirmed, visual-pending |
| Fallback Category Replacement Mobile 390 | MetadataManagePage + BottomSheet | reuse-confirmed, visual-pending |
| Metadata Rules References Desktop 1440 | MetadataManagePage + StatusChip | reuse-confirmed, visual-pending |
| Task53 Error Accessibility Code Map 1440 | manifest + OpenAPI + UI 17 | structure-confirmed, visual-pending |

## 3. Remaining gate

复用证据只证明无需重建基础组件，不证明布局、文案、对比度和响应式已经验收。Task53U 开工前仍需：

1. Task53.3/53.4 DTO 和错误码落盘。
2. 按 manifest 生成本地审阅稿并核对 375/390/430/1440。
3. 对照实现截图执行长商户、8 标签、键盘焦点和 Fresh Light/Dark Glass 验收。

线上 Figma 账号、URL 和 node ID 仍未验证，继续保持 `not_verified`。
