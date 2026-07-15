# UI-FL-10 全局验收启动评审

状态：准入通过，进入实现与全局验收<br>
评审日期：2026-07-15

## 1. 任务契约

```text
关联业务 Task：Task46、UI-FL-01 至 UI-FL-09；Task50 仅并行文档准备
路由与页面：登录、初始化、AppShell、Dashboard、流水、分析、结算、设置、元数据、导入、周期规则
共享组件归属：UI-FL-01 基础原语、UI-FL-02 AppShell、UI-FL-10 全局缺陷收口
API/DTO/权限/金额：保持不变
并行任务与冲突判断：Task50P.1-P.5 已冻结但代码禁入，不修改未来多账本路由或状态
Figma/Frame/本地规范：Fresh Light 29 Frame 基线；Task50 28 Frame 不计入本次 v1.2 验收
验收视口：375/390/430/1440
自动化测试：frontend lint/test/build、必要的后端全量回归
真实业务路径：登录后只读遍历全部路由、主题切换、导入/结算入口；不提交真实业务写入
回滚方式：Dark Glass 显式主题、UI-FL-10 原子提交、本机镜像回退；不改 migration
```

## 2. 准入事实

1. UI-FL-01 至 UI-FL-09 已有独立代码与验收提交，波次 A-D 已关闭。
2. 2026-07-15 基线门禁通过：27 个测试文件、99 个测试，lint 与 production build 均成功。
3. 本机 staging 运行在 `http://localhost:38088`，schema 19、XLSX 开关和数据库健康检查已在前序任务验证。
4. 本次不部署 NAS，不创建 migration，不开始 Task50 业务代码。
5. Fresh Light 当前不是无偏好用户的默认主题；这是 UI-FL-10 必须关闭的全局体验缺口。

## 3. 启动审计发现

| 等级 | 缺口 | 处理 |
|---|---|---|
| P1 | `DEFAULT_UI_THEME` 仍为 Dark Glass，新浏览器首次进入与 Fresh Light 目标不一致 | 切换默认值并保留用户显式 Dark Glass 偏好 |
| P1 | AppShell 缺少跳转主内容入口和路由切换焦点落点 | 增加 skip link、main target 和 route focus |
| P1 | 登录密码显隐按钮不在键盘顺序且没有可读名称 | 恢复 Tab 顺序并补 aria label/pressed |
| P1 | 离线草稿箱使用无焦点管理的旧 Drawer 和原生 confirm，状态色为硬编码 | 复用 BottomSheet、ConfirmDialog、Button、StatePanel |
| P1 | 周期规则页仍有 Dark Glass 专属 rgba/硬编码危险色和旧确认弹窗 | 改为语义 Token 并复用 ConfirmDialog |
| P2 | 全局旧样式动画未完整响应 reduced motion | 增加全局 reduced-motion 收口 |
| P2 | 单个 JS bundle 仍有约 678 kB 告警 | 记录为后续性能专项；不为清告警在本任务重构业务模块 |

## 4. 验收策略

1. 无主题偏好的新会话必须进入 `fresh-light`；已有 `dark-glass` 偏好保持不变。
2. 375/390/430/1440 遍历核心和工具路由，检查 `scrollWidth <= innerWidth`、关键操作无底栏/FAB 遮挡。
3. Dark Glass 显式切换后仍可登录、导航和读取主页面，再切回 Fresh Light。
4. 检查 skip link、键盘焦点、Dialog/Sheet 焦点返回、按钮可读名称和非颜色状态。
5. 运行前后记录 SQLite `quick_check`、流水/结算/导入批次数量，确保只读验收不改变业务数据。

## 5. 准入结论

UI-FL-10 可直接执行。上述 P1 均属于现有表现层和可访问性缺陷，不需要新增 API、依赖或 migration；修复后再决定是否关闭整个 Fresh Light 专项和进入 Task50P.6。
