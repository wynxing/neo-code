# NeoCode TUI v2 设计文档：Ghost Console

> 版本：v3.0 — TUI v2 Focus-Only Ghost Console
> 日期：2026-05-18
> 状态：设计草案
> 原始文档已拆分为模块化文档集。

---

## 文档已拆分

本文档原为 1628 行的单体设计文档，现已拆分为一个模块化文档集，位于 `docs/tui-v2/` 目录下。

请从导航页开始阅读：

### [→ TUI v2 文档导航页（Hub）](./tui-v2/tui-v2-architecture-hub.md)

包含：架构总览、核心约束、v1/v2 共存策略、禁止事项、验收标准。

### 子文档

| 主题 | 文档 | 说明 |
|------|------|------|
| **视觉与交互** | [`tui-v2/tui-v2-ui-ux-design.md`](./tui-v2/tui-v2-ui-ux-design.md) | Ghost Console 视觉语言、Focus-Only 布局、三层键位系统、LazyVim 色彩 |
| **数据与契约** | [`tui-v2/tui-v2-data-and-gateway.md`](./tui-v2/tui-v2-data-and-gateway.md) | ViewModel 状态模型、Gateway RPC 接口需求、事件协议、错误处理 |
| **工程实现** | [`tui-v2/tui-v2-implementation-guide.md`](./tui-v2/tui-v2-implementation-guide.md) | 目录结构、组件拆分原则、App 装配、Phase 1-4 实施路线图 |
| **HTML 原型** | [`design/tui-v2-terminal-preview.html`](./design/tui-v2-terminal-preview.html) | 可交互的终端视觉原型 |

---

## 拆分原因

原单体文档 1628 行，涵盖视觉、交互、数据、工程四个独立主题域。拆分为文档集后：

- **导航页**（266 行）：提供架构全貌和子文档索引，快速定位
- **UI/UX 规范**（625 行）：独立可读的视觉交互设计文档
- **数据契约**（375 行）：独立可读的状态模型和 Gateway API 文档
- **工程指南**（311 行）：独立可读的代码实施文档

每个子文档在开篇以 What / Why / How 三段式结构引导读者理解目标、设计考量和具体方案。

---

*本文档内容已迁移至 `docs/tui-v2/` 目录。后续所有更新请提交至对应子文档。*
