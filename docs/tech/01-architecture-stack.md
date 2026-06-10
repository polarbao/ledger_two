# 技术：总体架构与技术选型

## 1. 推荐架构

```text
Browser / PWA
  -> React Frontend
  -> Go REST API
  -> SQLite
  -> NAS local storage
```

## 2. 技术栈

- Frontend：React + TypeScript + Vite + Tailwind + TanStack Query。
- Backend：Go + SQLite + REST JSON。
- Database：SQLite，后续可迁移 PostgreSQL。
- Deploy：Docker Compose。
- Runtime：群晖 NAS / 本地开发机。

## 3. 选型原则

- 两人或家庭使用，避免微服务。
- 数据本地优先，备份优先。
- API 为跨端预留，不绑定 Web 实现。
- 金额统一 int64 cents。

## 4. 后续扩展

- PWA：复用 Web。
- React Native / Expo：复用 REST API。
- Tauri：可选桌面端。
