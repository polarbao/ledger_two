# LedgerTwo UI/Figma 设计包与 v1.1 收口判断

日期：2026-07-08  
阶段：v1.1 收口，v1.2 导入模块准备

## 1. 本轮目标

1. 基于现有 PRD、UI 文档、当前前端页面和 `docs/ui/lynntest(1).html`，输出 Figma 配套文件。
2. 判断 v1.1 当前 UI/UX 风格是否需要进一步调整。
3. 继续 v1.1 收口检查，判断是否可进入 v1.2。

## 2. 已生成的 UI/Figma 配套文件

新增目录：

```text
docs/ui/figma/
```

文件清单：

| 文件 | 作用 |
|---|---|
| `README.md` | Figma 配套设计包入口 |
| `ledger-two-design-system-brief.md` | 设计方向、当前 UI 差异、v1.1 是否调整判断 |
| `ledger-two.design-tokens.json` | 前端和设计系统 token 草案 |
| `ledger-two.figma-variables.json` | Figma Variables 集合草案 |
| `ledger-two-frame-manifest.json` | Figma 页面、Frame 和组件建模清单 |
| `component-library.md` | 组件库规格、状态和前端映射 |
| `v1.1-v1.2-ui-draft-spec.md` | v1.1/v1.2 逐屏 UI 设计稿规格 |
| `handoff-checklist.md` | UI 设计与开发交接检查清单 |

已同步更新：

1. `docs/ui/README.md`
2. `docs/ui/15-ledgertwo-ux-optimization-program.md`
3. `docs/00_DOCUMENT_INDEX.md`

## 3. 设计判断

### 3.1 当前 v1.1 UI/UX 是否需要调整

结论：v1.1 不建议做大规模 UI/UX 风格迁移。

原因：

1. 当前深色玻璃体系已经完成多轮真实业务 UI 验收。
2. v1.1 目标是可信赖与高频记账冻结，不是视觉重构。
3. `lynntest(1).html` 的浅色财务方向与现有前端差异较大，直接迁移会影响 AppShell、卡片、抽屉、设置页和移动端断点。
4. v1.1 剩余风险主要是验收证据和 NAS 复核，不是视觉系统不可用。

### 3.2 `lynntest(1).html` 的采用方式

建议把它作为 v1.2 和长期 UI/UX 专项的设计蓝本，而不是立即替换 v1.1。

可吸收内容：

1. 浅色背景和绿色财务主色。
2. 月度摘要、分段筛选、交易卡片和状态 chip。
3. 更轻、更适合长期记账的视觉气质。

不建议照搬内容：

1. 手机原型壳式大容器。
2. 过多装饰性图标或 emoji。
3. 单纯移动端单列结构。
4. 对高风险操作不够强的边界表达。

### 3.3 推荐演进路线

| 阶段 | UI/UX 策略 |
|---|---|
| v1.1 | 保持现有 Dark Glass，只修复布局、文案、状态和验收问题 |
| v1.2 | 在导入工作台试点 Fresh Light 变量与更高信息密度组件 |
| v1.3+ | 评估全局浅色主题或双主题，减少玻璃态和强紫色占比 |

## 4. v1.1 收口验证

本轮执行了只读健康检查，未执行部署、恢复、删除或数据写入。

### 4.1 本机 WSL2

命令：

```powershell
Invoke-WebRequest -Uri 'http://localhost:38088/api/healthz' -UseBasicParsing -TimeoutSec 5
Invoke-WebRequest -Uri 'http://localhost:38088/' -UseBasicParsing -TimeoutSec 5
```

结果：

```text
/api/healthz: {"success":true,"data":{"db":"ok","schema_version":12,"status":"ok","version":"1.1.0-rc"}}
/: 200, content length 745
```

### 4.2 NAS 局域网地址

命令：

```powershell
Invoke-WebRequest -Uri 'http://192.168.0.115:38088/api/healthz' -UseBasicParsing -TimeoutSec 5
Invoke-WebRequest -Uri 'http://192.168.0.115:38088/' -UseBasicParsing -TimeoutSec 5
```

结果：

```text
/api/healthz: {"success":true,"data":{"db":"ok","schema_version":12,"status":"ok","version":"1.1.0-rc"}}
/: 200, content length 745
```

判断：

1. NAS 上的 LedgerTwo 服务可通过 `192.168.0.115:38088` 访问。
2. NAS 版本、schema 和数据库健康状态与本机当前 v1.1 rc 预期一致。
3. 本轮没有进行浏览器登录、Dashboard、设置页、附件上传、备份下载和恢复确认的真实 UI 验收。

## 5. 是否可以执行 v1.2

结论：可以开始 v1.2 的准备性工作，但正式 Task47 开发建议等待 v1.1 冻结或明确豁免。

已经具备：

1. v1.2 PRD、模块、业务、服务、技术契约和 UI 工作台文档已经闭环。
2. 本轮补齐了 Figma 配套设计包、逐屏设计稿规格和组件库规格。
3. 本机和 NAS 均可访问 v1.1.0-rc 服务。

仍需收口：

1. v1.1 PRD 中“普通支出 10 秒内完成”“共同支出 20 秒内完成”仍需证据化。
2. “历史账单保留归档项展示”仍需证据化或补验证记录。
3. NAS 地址下仍需完成登录、Dashboard、设置页、附件、备份和恢复确认的 UI 级复核。

建议：

1. 下一步优先完成 NAS UI 复核和 PRD 剩余验收证据。
2. 若无阻断缺陷，标记 v1.1 冻结。
3. v1.1 冻结后进入 v1.2 Task47 CSV 导入预览。
4. v1.2 UI 设计以 `docs/ui/figma/` 为准，导入工作台可试点 Fresh Light。

## 6. 验证

已执行：

```powershell
git diff --check
```

结果：通过。仅有 Git 行尾提示，不影响 diff 检查。

未执行：

1. 前端测试。
2. 后端测试。
3. 浏览器 UI 验收。

原因：本轮只修改文档和设计交付物，并执行只读 HTTP 健康检查。

