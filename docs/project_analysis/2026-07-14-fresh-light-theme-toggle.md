# Fresh Light 主题切换说明

状态：源码实现与前端门禁完成；本机 staging 静态更新待发布窗口<br>
日期：2026-07-14

## 1. 为什么当前仍是深色

`frontend/src/theme/theme.ts` 的默认值仍为 `dark-glass`。这是 UI-FL 分阶段迁移的既定回退策略，不是 Fresh Light Token 未实现：AppShell、Dashboard、记账和流水已经使用语义 Token，但结算、设置、分析、导入等页面仍有部分 Dark Glass 专用样式。若现在直接把全局默认改成浅色，会让尚未迁移页面出现对比度和状态色不一致。

因此当前采用“显式可切换、偏好持久化、默认暂不翻转”的策略。Fresh Light 成为新默认仍归 UI-FL-10 全局验收，不由单个页面任务提前决定。

## 2. 本次实现

- 登录、初始化、桌面 AppShell 和移动顶部均提供太阳/月亮图标按钮。
- 当前为深色时按钮切换到白绿色 Fresh Light；当前为浅色时可切回 Dark Glass。
- 用户选择写入浏览器 `localStorage`，刷新、重新登录和下次打开继续使用相同主题。
- 浏览器禁止 storage 时仍可在当前会话切换，不影响登录或记账。
- 按钮包含可读 `aria-label`、`aria-pressed` 和 hover title，不新增 API、migration 或依赖。

## 3. 阶段边界

该按钮用于预览和逐页验收，不代表所有页面已经完成 Fresh Light 迁移。UI-FL-06/07/08/09 继续分别负责结算、设置、导入和分析，UI-FL-10 完成 375/390/430/1440、可访问性、真实业务和 Dark Glass 回退后，再评审是否把 `DEFAULT_UI_THEME` 改成 `fresh-light`。

当前 `http://localhost:38088` 仍引用旧静态产物；源码构建通过不等于运行实例已出现按钮。更新本机 staging 静态资源或重建固定候选镜像后才能在该地址验收。
