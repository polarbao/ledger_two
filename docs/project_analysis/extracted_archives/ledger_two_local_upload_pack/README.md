# 双人共享记账工具 LedgerTwo 初版项目文档

本文档包面向“2 人使用、Web 端优先、群晖 NAS 私有部署”的双人共享记账工具。

## 文档清单

1. `docs/01_PRD.md`：产品需求文档
2. `docs/02_UI_INTERACTION_DESIGN.md`：新版 UI 交互设计稿
3. `docs/03_TECH_DESIGN.md`：技术设计文档
4. `docs/04_TECH_IMPLEMENTATION.md`：技术实现文档
5. `docs/05_FRONTEND_DESIGN.md`：前端设计与实现方案
6. `docs/06_NAS_DEPLOYMENT.md`：群晖 NAS 部署文档
7. `docs/07_DATABASE_API.md`：数据库与 API 设计文档
8. `docs/08_MVP_ROADMAP.md`：MVP 范围、里程碑与验收

## 推荐实现主线

- 前端：React + TypeScript + Vite + Tailwind CSS + TanStack Query
- 后端：Go + Chi/Gin + SQLite + goose + sqlc/GORM
- 部署：Docker Compose + 群晖 Container Manager + Tailscale 私有访问

