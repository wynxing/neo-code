# NeoCode TUI v2 — 状态模型与 Gateway 契约

> 版本：v3.0 | 日期：2026-05-18 | 父文档：[架构导航](./tui-v2-architecture-hub.md)

---

## What — 本规范解决什么问题

本规范定义 **TUI v2 前端如何管理状态（ViewModel）以及如何与后端 Gateway 通信**。它要解决的问题是：

- **TUI 需要什么样的数据模型？** 当前 App struct 承载了几乎所有状态，组件之间没有清晰的 ViewModel 边界。需要定义一套纯展示层的数据结构，不依赖后端内部类型。
- **哪些 Gateway 接口已经存在、哪些需要新增？** 当前 24+ 个 RPC 方法大部分满足需求，但 Ghost Console 的 Soft Inspector、Mission Block、连接前探测等场景需要额外的接口支持。
- **事件流如何驱动 UI 变化？** 30+ 种 Runtime 事件需要明确映射到 ViewModel 更新路径和 UI 重绘触发条件。
- **错误如何展示？** Ghost Console 不使用全屏弹窗报错，需要定义错误分类、内联展示风格和重连策略。

**边界**：本规范仅涉及前端状态结构和后端通信契约。视觉渲染由 [UI/UX 规范](./tui-v2-ui-ux-design.md) 定义，代码目录结构由 [工程实现指南](./tui-v2-implementation-guide.md) 定义。

## Why — 为什么这样设计

### 为什么 TUI 不能直接读数据库

这是 NeoCode 最核心的架构约束。原因有三：

1. **分层隔离**：TUI 是展示层，Session 是持久化层。直接跨层访问会形成隐式契约——Session 的表结构变化会直接破坏 TUI 渲染，导致前端和后端无法独立演进。
2. **安全边界**：SQLite 文件包含完整的对话历史、工具调用记录和会话元数据。TUI 进程与 Gateway 进程可能运行在不同的权限上下文中。
3. **单一直观的数据源**：所有状态通过 Gateway 的事件流推送，TUI 无需轮询、无需推断、无需处理并发写入冲突。Gateway 是中继层，负责去重、排序和授权。

### 为什么用事件驱动而不是轮询

Ghost Console 的视觉核心是"Agent 行为流"——事件节奏驱动 UI 更新。轮询模型有三个致命问题：

1. **延迟**：轮询间隔内发生的事情会被合并，无法呈现"打字机效果"和"工具逐个启动"的时间节奏。
2. **浪费**：大部分轮询返回空结果，消耗 Gateway 和 TUI 的 CPU。
3. **复杂度**：需要处理"上次轮询后发生了什么"的 diff 逻辑，而事件流天然是追加模型——新事件 = 新 StreamEntry。

### 为什么不能在 TUI 中伪造 Runtime 状态

Ghost Console 的 Ambient Status 和 Soft Inspector 展示的 Runtime 状态（Phase、Turn、Tokens、StopReason）必须来自真实的 Runtime 事件。如果 TUI 本地模拟这些字段，会导致：

- 状态显示与实际运行不同步（例如显示 `◉ run` 但 Runtime 已因错误停止）
- 权限审批、ask_user 等交互依赖准确的状态机位置
- 调试和问题诊断时无法信任 UI 显示

---

## How — 具体设计规范

### 1. TUI 状态模型（ViewModel）

#### 1.1 核心 ViewModel

```go
// ViewState 是 TUI v2 的顶层视图状态。
// 所有字段来自 Gateway 或 TUI 本地 UI 状态。
type ViewState struct {
    Gateway    GatewayState
    ActiveSessionID string
    Sessions   []SessionSummaryVM

    Runtime    RuntimeState
    Stream     []StreamEntry
    Todos      []TodoVM

    Input      InputState
    Layout     LayoutState
    Theme      ThemeState
    Errors     []ErrorVM
}
```

#### 1.2 关键子结构体：连接与会话

```go
type GatewayState struct {
    Connected  bool
    Version    string
    LatencyMs  int64
    Since      time.Time
}
```

#### 1.3 关键子结构体：Runtime 状态

```go
type RuntimeState struct {
    Status      RuntimeStatus // idle, running, waiting, error
    Phase       string
    Turn        int
    MaxTurns    int
    TokensUsed  int64
    TokensTotal int64
    StopReason  string
}

type RuntimeStatus string
const (
    StatusIdle    RuntimeStatus = "idle"
    StatusRunning RuntimeStatus = "running"
    StatusWaiting RuntimeStatus = "waiting"
    StatusError   RuntimeStatus = "error"
)
```

#### 1.4 关键子结构体：Stream 条目

```go
// StreamEntry — Agent Stream 中的一项
// 可以是消息、工具调用、文件事件、系统事件等
type StreamEntry struct {
    ID        string
    Kind      StreamEntryKind
    Label     string          // 渲染标签：you, neo, tool.read_file, file.modified, design.shift
    Content   string          // 主体文本
    Detail    string          // 辅助信息（缩进渲染）
    Status    string          // ✓ ◉ ○ ◌ × ·
    Timestamp time.Time
    Children  []StreamEntry   // 子条目（如工具结果、文件 diff）
    Collapsed bool            // 是否折叠（用于 thinking 等）
}

type StreamEntryKind string
const (
    KindYou      StreamEntryKind = "you"
    KindNeo      StreamEntryKind = "neo"
    KindTool     StreamEntryKind = "tool"
    KindFile     StreamEntryKind = "file"
    KindEvent    StreamEntryKind = "event"
    KindError    StreamEntryKind = "error"
    KindThinking StreamEntryKind = "thinking"
)
```

#### 1.5 关键子结构体：输入状态

```go
// InputState — TUI v2 输入状态
type InputState struct {
    Text           string
    Mode           InputMode       // "input" / "normal"
    IsSending      bool
    IsWaiting      bool
    WaitingReason  string          // "permission" / "user_question"
    WaitingPrompt  string
    AttachedFiles  []string
    AgentMode      string          // "build" / "plan"
    Model          string
    ShowPalette    bool
    ShowHelp       bool
}

type InputMode string
const (
    ModeInput  InputMode = "input"   // 插入/输入模式
    ModeNormal InputMode = "normal"  // Normal Mode (Esc)
)
```

#### 1.6 关键子结构体：布局状态

```go
// LayoutState — TUI v2 本地 UI 状态
type LayoutState struct {
    Width            int
    Height           int
    InspectorVisible bool  // > 100 列时自动显示
    ShowLogViewer    bool
}
```

#### 1.7 数据来源映射

| ViewModel 字段 | 数据来源 | 类型 |
|---------------|---------|------|
| `Gateway.Connected` | TUI 本地 | 本地 |
| `Gateway.Version` | `gateway.ping` | Gateway 已有 |
| `ActiveSessionID` | TUI 本地 | 本地 |
| `Sessions` | `gateway.listSessions` | Gateway 已有 |
| `Runtime.*` | `gateway.event` 事件流 | Gateway 已有 |
| `Stream` | `gateway.event` 多种事件类型 | Gateway 已有 |
| `Todos` | `gateway.event` → `EventTodoUpdated` + `session.todos.list` | Gateway 已有 |
| `Input.Text` | TUI 本地 | 本地 |
| `Input.Mode` | TUI 本地（Normal/Input 切换） | 本地 |
| `Input.IsSending/Waiting` | TUI 本地 + Gateway ACK | 本地 |
| `Input.AgentMode/Model` | TUI 本地 + `gateway.getSessionModel` | 本地 + Gateway 已有 |
| `Layout.*` | TUI 本地 | 本地 |
| `Theme.*` | TUI 本地 | 本地 |
| `Errors` | `gateway.event` → `EventError` / JSON-RPC error | Gateway 已有 |

#### 1.8 禁止 TUI 直接读取的数据

| 数据 | 原因 | 正确获取方式 |
|------|------|------------|
| Session 原始消息列表 | SQLite 存储 | `gateway.loadSession` |
| Runtime 实例状态 | Runtime 是内部组件 | `gateway.event` 事件流 |
| Provider API Key | 安全管理 | TUI 不需要感知 |
| SQLite 数据库文件 | 持久化层 | TUI 不感知 |
| Session Store 内部状态 | 持久化实现细节 | 通过 Gateway 接口获取摘要 |

---

### 2. Gateway 接口需求分析

**重要：严格区分"已有"和"建议新增"。已有接口来自代码事实；建议新增接口来自 TUI v2 的需求推断。**

#### 2.1 已有 Gateway 接口（TUI 当前已使用，TUI v2 复用）

| 能力 | 方法 | 状态 |
|------|------|------|
| 连接认证 | `gateway.authenticate` | 已有 |
| 心跳/保活 | `gateway.ping` | 已有 |
| 事件流绑定 | `gateway.bindStream` | 已有 |
| 发起运行 | `gateway.run` | 已有 |
| 取消运行 | `gateway.cancel` | 已有 |
| 手动压缩 | `gateway.compact` | 已有 |
| 系统工具 | `gateway.executeSystemTool` | 已有 |
| 权限审批 | `gateway.resolvePermission` | 已有 |
| ask_user 回答 | `gateway.userQuestionAnswer` | 已有 |
| 会话列表 | `gateway.listSessions` | 已有 |
| 会话详情 | `gateway.loadSession` | 已有 |
| 创建会话 | `gateway.createSession` | 已有 |
| 重命名会话 | `gateway.renameSession` | 已有 |
| 删除会话 | `gateway.deleteSession` | 已有 |
| 激活技能 | `gateway.activateSessionSkill` | 已有 |
| 停用技能 | `gateway.deactivateSessionSkill` | 已有 |
| 列出激活技能 | `gateway.listSessionSkills` | 已有 |
| 列出可用技能 | `gateway.listAvailableSkills` | 已有 |
| 列出模型 | `gateway.listModels` | 已有 |
| 设置会话模型 | `gateway.setSessionModel` | 已有 |
| 获取会话模型 | `gateway.getSessionModel` | 已有 |
| 列出文件 | `gateway.listFiles` | 已有 |
| 读取文件 | `gateway.readFile` | 已有 |
| 列出 Git 变更 | `gateway.listGitDiffFiles` | 已有 |
| 读取 Git diff | `gateway.readGitDiffFile` | 已有 |
| Checkpoint 列表 | `checkpoint.list` | 已有 |
| Checkpoint 恢复 | `checkpoint.restore` | 已有 |
| Checkpoint 撤销恢复 | `checkpoint.undoRestore` | 已有 |
| Checkpoint diff | `checkpoint.diff` | 已有 |
| 工作区列表 | `gateway.listWorkspaces` | 已有 |
| 创建工作区 | `gateway.createWorkspace` | 已有 |
| 切换工作区 | `gateway.switchWorkspace` | 已有 |
| 重命名工作区 | `gateway.renameWorkspace` | 已有 |
| 删除工作区 | `gateway.deleteWorkspace` | 已有 |
| Runtime 快照 | `runtime.snapshot.get` | 已有，TUI v1 未启用 — v2 需启用 |
| Session Todo 列表 | `session.todos.list` | 已有，TUI v1 未启用 — v2 需启用 |
| 事件流推送 | `gateway.event` (notification) | 已有 |

#### 2.2 建议新增 Gateway 接口

**`gateway.health`** — 优先级 P0

TUI v2 需要在认证前探测 Gateway 是否在线。当前 `gateway.ping` 需要认证。HTTP 端点 `GET /healthz` 已存在（在 `network_server.go` 中注册），但 JSON-RPC 层无等价方法。

建议响应：
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 3600,
  "connections_active": 3
}
```

**`gateway.listToolCalls`** — 优先级 P1

Soft Inspector 和 Stream 重建需要主动查询的工具调用历史。当前只能从事件流被动收集，会话恢复后无法重建。

建议参数：`{"session_id", "limit?", "run_id?"}`

**`gateway.listFileChanges`** — 优先级 P1

需要主动查询会话级文件变更摘要。与 `gateway.listGitDiffFiles` 不同：此接口查询的是 Agent Run 产生的文件变更，而非 git 层面的工作树变更。

建议参数：`{"session_id", "limit?"}`

**`gateway.getSessionUsage`** — 优先级 P1（待确认）

当前 `gateway.loadSession` 返回的 `Session` 中可能已包含 `TokenInputTotal` 和 `TokenOutputTotal`。如已包含则直接使用，无需新增接口。

#### 2.3 接口缺口总结

| 能力 | 当前状态 | 优先级 |
|------|---------|--------|
| Gateway 健康检查（RPC） | 缺失 | P0 |
| Runtime 状态快照 | 已有，TUI 未启用 | P0 |
| Session Todo 列表 | 已有，TUI 未启用 | P0 |
| 工具调用历史查询 | 缺失 | P1 |
| 文件变更汇总查询 | 缺失 | P1 |
| Token 用量查询 | 待确认 | P1 |

---

### 3. Gateway 事件协议

#### 3.1 当前事件协议

当前 TUI 已消费 `gateway.event` 通知（JSON-RPC notification），`internal/tui/services/gateway_stream_client.go` 中 `restoreRuntimePayload()` 解码了 30+ 种事件类型。TUI v2 复用相同的事件协议。

#### 3.2 事件 → ViewModel → UI 映射

| 事件 | 更新 ViewModel | 触发 UI 变化 |
|------|---------------|-------------|
| `agent_chunk` | `Stream[last].Content += delta` | Stream 最后一条追加重绘 |
| `agent_done` | `Stream[last]` 完成标记 | Stream Markdown 最终渲染 |
| `thinking_delta` | `Stream[last].Content += delta` | Thinking 折叠区域更新 |
| `phase_changed` | `Runtime.Phase`, `Runtime.Status` | Ambient Status 状态符号更新 |
| `tool_start` | `Stream.append(StreamEntry{Kind:KindTool})` | Stream 新增工具标签 |
| `tool_result` | `Stream.append(child entry)` | Stream 工具结果追加 + Soft Inspector |
| `tool_diff` | `Stream.append(StreamEntry{Kind:KindFile})` | Stream 文件 diff + Soft Inspector |
| `token_usage` | `Runtime.TokensUsed/Total` | Ambient Status Token 计数 + Soft Inspector |
| `permission_requested` | `Input.IsWaiting = true` | Command Prompt 切换为权限模式 |
| `user_question_requested` | `Input.IsWaiting = true` | Command Prompt 切换为问答模式 |
| `runtime_snapshot_updated` | `Runtime.*`, `Todos` | Soft Inspector 全量刷新 |
| `todo_updated` | `Todos` | Stream Mission Block 更新 |
| `stop_reason_decided` | `Runtime.Status = idle` | Ambient Status 切换为 idle |
| `run_diff_summary` | Soft Inspector files | Soft Inspector 文件变更汇总 |
| `error` | `Errors.append(...)` | Stream 错误条目 + Soft Inspector |
| `checkpoint_created` | — | Ambient Status 短暂提示 |

#### 3.3 事件渲染原则

- 事件驱动，不轮询
- 每种事件有明确的 ViewModel 更新路径
- Stream 条目不可变（新事件 = 新条目追加）
- Inspector 数据为瞬时快照，每次相关事件到达时刷新

---

### 4. 错误处理设计

#### 4.1 错误分类与处理策略

| 错误场景 | Gateway 返回 | Ghost Console 展示 | 用户操作 |
|---------|-------------|-------------------|---------|
| Gateway 未启动 | 连接失败 | 启动画面 "Gateway 未运行" | 等待自动启动 |
| Gateway 连接中断 | TCP 断开 | Status 显示 `× disconnected` | 自动重连 |
| 认证失败 | `gateway_code=unauthorized` | 全屏错误提示 | 检查 token |
| Session 不存在 | `gateway_code=resource_not_found` | Status 短暂闪烁 + Stream 提示 | 选择其他会话 |
| Runtime 不可用 | `gateway_code=internal_error` | Status `× error` | 重试 |
| Runtime 正在忙 | ACK 冲突 | Command Prompt 短暂提示 | 等待或取消 |
| 模型不可用 | `gateway_code=invalid_action` | Command Prompt 提示 | 切换模型 |
| 工具调用失败 | `tool_result(is_error=true)` | Stream 内联错误 + Inspector | Agent 自动处理 |
| 权限不足 | `gateway_code=access_denied` | Status `◌ wait` + 权限弹窗 | 选择 allow/deny |
| 请求超时 | `gateway_code=timeout` | Command Prompt 短暂提示 | 重试 |
| 事件流断开 | SSE/WS 关闭 | Status `× reconnecting` | 自动重连 |
| 用户取消任务 | `EventRunCanceled` | Stream "run cancelled" + Status `○ idle` | 可重新发送 |

#### 4.2 Ghost Console 错误展示风格

错误不使用全屏弹窗（权限确认除外）。错误信息作为 Stream 条目呈现：

```
  error
    gateway_code: timeout
    compact timed out after 30s
    [retry]
```

Status 行使用状态符号变化：
- 正常：`NEOCODE  ○ idle   build   ...`
- 错误：`NEOCODE  × error  build   ...`（红色 `×`）
- 重连：`NEOCODE  ◌ reconnect   build   ...`

#### 4.3 自动重连流程

```
连接中断
  → Status: "× disconnected"
  → 自动重连（指数退避: 1s, 2s, 4s, 8s, max 30s）
  → 成功: Status 恢复 + 自动 re-authenticate + re-bindStream
  → 失败 × 10: Status "× offline" + Stream 底部 "gateway unreachable [retry] [exit]"
```

---

*本规范是 TUI v2 文档集的子文档。架构总览见 [架构导航](./tui-v2-architecture-hub.md)，视觉交互见 [UI/UX](./tui-v2-ui-ux-design.md)，工程实现见 [工程实现](./tui-v2-implementation-guide.md)。*
