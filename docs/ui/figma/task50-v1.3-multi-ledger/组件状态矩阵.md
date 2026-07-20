# Task50 多账本组件状态矩阵

状态：Task50P.5 冻结设计要求

| 组件 | 必需变体 | 禁止行为 | 代码归属 |
|---|---|---|---|
| LedgerSwitcher | loading/active/error/no-active、desktop/mobile | 混入 archived 日常列表、静默 fallback | Task50.4 |
| LedgerListItem | active/archived、owner/editor/viewer、selected | 只用颜色表示状态、行内堆叠全部危险按钮 | Task50.5 |
| LedgerCreateSurface | idle/invalid/submitting/error | 把同名当冲突、提交失败清空名称 | Task50.5 |
| LedgerRenameSurface | idle/invalid/submitting/version-conflict | archived 重命名、冲突自动覆盖 | Task50.5 |
| LedgerArchiveSurface | settled/unsettled/ready-blocked/submitting | 自动结算、ready 存在仍提交 | Task50.5 |
| LedgerRestoreSurface | idle/submitting/version-conflict | 承诺自动补周期账、非 Owner 展示主动作 | Task50.5 |
| LedgerMemberRow | owner/editor/viewer/self/other/archived | 普通 Select 产生 Owner、归档态变更成员 | Task50.5 |
| AddMemberSurface | empty/history-warning/limit-reached/error | 不确认历史可见性、添加第三人 | Task50.5 |
| OwnerTransferSurface | idle/submitting/version-conflict | 分两次更新角色、用“确定”作为主动作 | Task50.5 |
| LeaveRemoveSurface | editor-leave/viewer-leave/owner-blocked/remove-other | 删除历史数据、Owner 直接离开 | Task50.5 |
| ArchivedLedgerBanner | owner/editor/viewer | 持久化为 recent active、显示写 FAB | Task50.5 |
| NoActiveLedgerShell | empty/creating/error/archived-available | 挂载交易/报表/导入 query | Task50.4 |
| InstanceAdminSection | admin/non-admin/error | 用账本 Owner 代替实例管理员 | Task50.5 |

## 文案冻结

| 场景 | 主标题 | 主动作 |
|---|---|---|
| 创建 | 创建账本 | 创建并进入账本 |
| 重命名 | 重命名账本 | 保存名称 |
| 归档 | 归档后全员只读 | 归档账本 |
| ready 阻断 | 先处理待确认导入 | 前往导入处理 |
| 恢复 | 恢复后重新开放写入 | 恢复账本 |
| 添加成员 | 新成员可查看部分历史 | 添加成员 |
| Owner 移交 | 你将失去账本管理权限 | 移交所有权给 {username} |
| 移除成员 | 对方将立即失去访问 | 移除 {username} |
| 离开 | 你将立即失去访问 | 离开账本 |
| Owner 离开阻断 | 请先移交所有权 | 前往移交 |
| 版本冲突 | 账本已在另一处更新 | 刷新账本信息 |

所有危险主动作必须带具体对象/结果，不能使用单独“确定”“继续”或只靠红色区分。
