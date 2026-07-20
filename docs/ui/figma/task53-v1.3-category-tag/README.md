# Task53 v1.3 分类标签智能化 Figma Handoff

状态：本地设计要求、代码复用证据与真实页面视觉审阅稿已完成；线上 Figma 同步未验证<br>
冻结日期：2026-07-16<br>
关联任务：Task53P.5、未来 Task53U

## 1. Directory role

本目录保存 Task53 的结构化 Figma 要求和代码交接，不保存真实 `.fig`、账户凭据或线上同步承诺。

| File | Class | Purpose |
|---|---|---|
| `README.md` | requirement | 范围、事实源、Page、组件和同步规则 |
| `task53-frame-manifest.json` | requirement | required Frame、视口、状态、路由和验收映射 |
| `component-state-matrix.md` | requirement | 状态变体、文案、可访问性和禁止行为 |
| `reuse-evidence.md` | requirement / reuse evidence | required Frame 与现有 React/UI foundation 的复用映射 |

本地真实页面审阅稿已生成到 `../local-review/task53-v1.3-2026-07-20/`，并标注为 generated review artifact。后续审阅稿继续使用日期目录，不能写成“线上 Figma 已同步”。

## 2. Source priority

1. `../../../prd/34-prd-v1.3-category-tag-intelligence.md`
2. `../../../tech/26-v1.3-category-tag-intelligence-contract.md`
3. `../../../api/openapi-v1.3-category-tag-draft.yaml`
4. `../../../fixtures/category-tag/README.md`
5. `../../17-v1.3-category-tag-intelligence-flows.md`
6. Fresh Light token、现有 UI foundation 与本 manifest
7. 线上 Figma、截图和本地生成预览

设计稿不得反向改变规则优先级、历史规则 suggest 兼容、8 标签上限、Owner 权限、preview/commit 分离或批量/学习解耦。

## 3. Figma page

在 Fresh Light 工作稿新增 `08 v1.3 Classification & Metadata` Page：

1. `00 Context & Components`
2. `01 Classification Summary`
3. `02 Preview Rows & Editor`
4. `03 Bulk & Reclassify`
5. `04 Rule Management`
6. `05 Metadata Profiles`
7. `06 Metadata Safeguards`
8. `07 Error Accessibility Code Map`

Frame 名称、尺寸和状态以 manifest 为准，不创建同义别名。

## 4. Component reuse

必须复用 Button、IconButton、StatusChip、SegmentedControl、StatePanel、Dialog、BottomSheet、ResponsiveDataList、SettingsSection 和现有 Import Preview 结构。

Task53 业务组合组件：

1. `ClassificationSummary`
2. `ClassificationStatusChip`
3. `ClassificationExplanation`
4. `BulkClassificationBar`
5. `RememberMerchantControl`
6. `DefaultMetadataProfilePreview`
7. `FallbackCategoryReplacement`

业务组合不得重新定义 token、按钮、圆角或基础 Dialog。

## 5. Online sync

```text
online_figma_sync=not_verified
figma_account=unknown_for_this_handoff
figma_file_url=not_bound
verified_node_ids=[]
```

线上同步后必须记录非敏感账号/团队标识、可编辑 URL、Frame node ID、同步时间、只读复核和 manifest 差异。

## 6. UI entry gate

Task53U 入口与关闭状态：

1. Task53.3/53.4 DTO 与错误码已冻结。
2. required Frame 已有复用证据和 375/390/430/1440 双主题真实页面截图。
3. component-state-matrix 无未决业务冲突，键盘、`aria-live` 和横向溢出门禁已通过。
4. online Figma 继续标注 `not_verified`，不阻断本地 Task53U 关闭。
