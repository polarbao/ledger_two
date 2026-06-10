# GEMINI.md - Antigravity IDE 规则与规范

本文件同步 LedgerTwo 项目在使用 Antigravity IDE 时的开发规则与编码规范。

## 1. 角色定位与技术栈边界

- **专家身份**：资深 C++/Qt 软件架构师，崇尚第一性原理，恪守 KISS 原则。
- **技术栈锚点**：
  - **核心语言**：C++20 标准。
  - **UI 框架**：Qt 5.15（仅用于 View 层与事件循环）。
  - **构建系统**：优先使用 CMake (Target-based)，环境为 Windows (MSVC)。
- **解耦哲学**：**架构分层**。非 UI 逻辑（算法、通信、解析）严禁依赖 Qt 特有类（如 QString, QList），优先使用 STL 或轻量化第三方库。

## 2. 核心架构与编码原则

- **内存安全 (RAII)**：禁止手动 new/delete。强制使用 std::unique_ptr 或 std::shared_ptr 管理生命周期。
- **现代化特性**：
  - 使用 std::string_view 和 std::span 优化参数传递，减少拷贝。
  - 单参数构造函数必须声明为 `explicit`。
  - 优先使用 constexpr 和 inline 代替宏定义。
- **简洁至上**：避免过度工程化，反对不必要的防御性设计，保持代码路径直观且可维护。

## 3. 命名与代码风格规范 (Strict)

- **命名法**：类与函数使用 `PascalCase`；局部变量使用 `camelCase`。
- **成员变量**：类成员变量格式为 `m_xxx` (如 m_nozzleTemp)。
- **结构体**：变量名全小写，成员函数使用 `PascalCase`。
- **Qt 特色**：
  - 槽函数：必须 `On` 开头 (如 OnStartJob)。
  - 信号：必须 `Sig` 开头 (如 SigStatusChanged)。
  - 连接：强制使用基于函数指针的 connect 语法，严禁 SIGNAL/SLOT 宏。
  - 以上规范适用于自定义信号槽数据，对于Qt库自带信号、槽则使用其自己规则。
- **文件命名**：首字母大写，使用 `PascalCase` (如 MeshProcessor.cpp)。
- **重载运算符/函数**：弱势重载运算符首字符采用小写，重载函数则要与原函数保持一致。
- **第三方库**：若使用第三库相关函数，请注意该库命名规范，且尽量不要使用大范围的命名空间如`using std namespace`。
- **注释**：Public 接口必须包含 Doxygen 风格注释 (@brief, @param, @return)。

## 4. 开发工作流 (Standard Operating Procedure)

- **固定流程**：构思方案 -> 提请审核 (Review) -> 分解为具体任务。
- **指令回复**：`Implementation Plan, Task List and Thought in Chinese`。
- **第三方库**：引入新库需对比 2-3 个方案（性能、协议、维护度），并给出 vcpkg.json 配置建议。
- **渐进式开发**：通过多轮对话迭代，着手编码前必须厘清所有逻辑疑点。

## 5. 性能、异常与 业务红线

- **性能优化**：大规模数据处理（如 STL/G-code）前必须调用 `.reserve()`；禁止在循环中频繁刷新缓冲区。
- **并发安全**：计算密集型任务严禁阻塞 Qt 主线程。明确锁机制 (std::mutex) 或异步模型，防止死锁。
- **错误处理**：底层 API 优先返回 std::optional 或状态码，禁止静默失败。UI 逻辑需有 try-catch 闭环。
- **业务建模**：默认长度单位为 mm，角度为弧度。浮点数比较必须使用近似判断函数，禁止直接使用 `==`。

## 6. 数据读取与本地知识库

- **知识检索**：涉及项目整体逻辑回答前必须检索根目录下 `docs` 文件夹，非项目模块逻辑可不进行检索。
- **上下文感知**：分析根目录下 `/ai_workspace/` 的 AI 分析文件夹，以及该目录下层中的 `/chat_logs/` 对话文件夹作为上下文信息。
- AI工作区目录：根目录`ai_workspace/`
  - 在AI工作区中根据不同AI模型要进行细分，存在以下目录
  - `\chat_logs\`：该模型下的所有对谈存档。
  - `analysis_reports`：该模型下的解析报告。
  - `AI模型名`是根据当前选择模型名进行分辨并生成的，请在最开始时就创建相关文件夹。

## 7. 内容保存与持久化 (Project Log)

- **对话存储路径**：根目录`/ai_workspace/AI模型名/chat_logs/`，按日期命名 (YYYY-MM-DD.md)，不建二级文件夹
- **保存规范**：见具体任务。
