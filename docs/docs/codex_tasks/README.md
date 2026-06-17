# Codex / Gemini 开发任务入口

状态：供审核  
适用阶段：Task30 已完成后，进入 v1.1 业务开发前的基础框架补齐阶段

## 1. 目标

本目录用于给 Codex、Gemini、Cursor、Copilot 或其他 AI 编码工具提供明确、可执行、可验收的开发任务和代码风格规范。

后续所有 AI 开发任务都应从本目录开始，而不是直接让 AI 阅读零散文档后自由发挥。

## 2. 文件列表

```text
00-ai-development-workflow.md   AI 开发工作流和通用提示词
01-repository-code-style.md     仓库通用代码风格和提交规范
02-backend-go-style.md          Go 后端代码风格
03-frontend-react-ts-style.md   React + TypeScript 前端代码风格
04-testing-quality-gates.md     测试与质量门禁
05-foundation-task-plan.md      Task31-Task40 基础框架任务计划
06-review-checklist.md          人类审核清单
07-reference-style-sources.md  代码风格参考来源
```

## 3. AI 开发强制流程

1. 读取 `docs/00_DOCUMENT_INDEX.md`。
2. 读取 `docs/prd/11-foundation-framework-before-v1.1.md`。
3. 读取本目录代码风格文档。
4. 读取对应任务。
5. 输出计划和预计修改文件，等待确认。
6. 只实现当前任务。
7. 运行测试和构建。
8. 输出变更摘要、验证命令、风险和下一步建议。

## 4. 禁止事项

1. 禁止一次性实现多个 Foundation Task。
2. 禁止实现未审核 v1.1 业务需求。
3. 禁止把权限判断只放在前端。
4. 禁止使用 float 计算金额。
5. 禁止修改历史 migration。
6. 禁止提交真实数据库、备份、上传文件和密钥。
7. 禁止绕过测试直接声称完成。
