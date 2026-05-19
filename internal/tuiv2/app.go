// Package tuiv2 实现 Ghost Console TUI v2 的应用骨架。
package tuiv2

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"neo-code/internal/tuiv2/gateway"
)

const (
	modeInput       = "input"
	statusIdle      = "idle"
	statusRunning   = "running"
	statusOffline   = "offline"
	surfaceName     = "ghost-console"
	defaultTerminal = "0x0"
)

// StartupConfig 承载 TUI v2 独立入口解析出的启动参数和 Gateway 客户端。
type StartupConfig struct {
	Backend  string
	Scenario string
	Debug    bool
	Client   gateway.Client
}

type appModel struct {
	cfg       StartupConfig
	width     int
	height    int
	status    string
	health    string
	sessions  []gateway.SessionSummary
	detail    *gateway.SessionDetail
	models    []gateway.ModelInfo
	events    []gateway.GatewayEvent
	eventCh   <-chan gateway.GatewayEvent
	lastError string
}

var (
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	idleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	debugStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
)

// NewApp 创建最小 Bubble Tea Model，当前只渲染 Ghost Console 状态行、Prompt 和调试信息。
func NewApp(cfg StartupConfig) tea.Model {
	return appModel{cfg: cfg, status: statusIdle}
}

// Init 从 Gateway 客户端加载初始状态并建立事件订阅。
func (m appModel) Init() tea.Cmd {
	if m.cfg.Client == nil {
		return nil
	}
	return loadInitialCmd(m.cfg.Client)
}

// Update 处理终端尺寸和退出按键，后续阶段会在这里接入 Gateway 事件消息路由。
func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	case initialLoadedMsg:
		m.health = msg.health
		m.sessions = msg.sessions
		m.detail = msg.detail
		m.models = msg.models
		m.eventCh = msg.eventCh
		m.lastError = msg.errText
		if msg.errText != "" {
			m.status = statusOffline
		}
		if msg.eventCh != nil {
			return m, waitEventCmd(msg.eventCh)
		}
	case gatewayEventMsg:
		if msg.closed {
			return m, nil
		}
		m.events = append(m.events, msg.event)
		m.applyEvent(msg.event)
		return m, waitEventCmd(m.eventCh)
	}
	return m, nil
}

// View 渲染无边框的 Ghost Console 占位界面，用状态符号、颜色和缩进表达层级。
func (m appModel) View() string {
	lines := []string{
		m.statusLine(),
		"",
	}
	lines = append(lines, m.stateLines()...)
	lines = append(lines, "", promptStyle.Render("› "))
	if m.cfg.Debug {
		lines = append(lines, "", debugStyle.Render(m.debugLine()))
	}
	return strings.Join(lines, "\n")
}

// statusLine 渲染 Ghost Console 顶部状态，保持无边框并用状态符号表达运行态。
func (m appModel) statusLine() string {
	parts := []string{
		statusStyle.Render("NEOCODE"),
		m.renderStatus(),
		mutedStyle.Render(m.cfg.Backend),
		mutedStyle.Render(surfaceName),
	}
	return strings.Join(parts, "   ")
}

// debugLine 渲染调试模式下的最小运行信息，便于后续阶段观察事件与窗口尺寸。
func (m appModel) debugLine() string {
	size := defaultTerminal
	if m.width > 0 || m.height > 0 {
		size = fmt.Sprintf("%dx%d", m.width, m.height)
	}
	return fmt.Sprintf(
		"[debug] mode:%s  scenario:%s  events:%d  size:%s",
		modeInput,
		m.cfg.Scenario,
		len(m.events),
		size,
	)
}

// renderStatus 根据当前 Gateway 状态渲染顶部状态符号。
func (m appModel) renderStatus() string {
	switch m.status {
	case statusRunning:
		return statusStyle.Render("◉ " + statusRunning)
	case statusOffline:
		return errorStyle.Render("× " + statusOffline)
	default:
		return idleStyle.Render("○ " + statusIdle)
	}
}

// stateLines 渲染从 Gateway 客户端取得的会话、历史和事件摘要。
func (m appModel) stateLines() []string {
	lines := []string{}
	if m.health != "" {
		lines = append(lines, mutedStyle.Render("  "+m.health))
	}
	if m.lastError != "" {
		lines = append(lines, errorStyle.Render("  × "+m.lastError))
	}
	if len(m.sessions) > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  sessions:%d  current:%s", len(m.sessions), m.sessions[0].Title)))
	}
	if len(m.models) > 0 {
		lines = append(lines, mutedStyle.Render("  model:"+m.models[0].ID))
	}
	if m.detail != nil {
		for _, item := range tailStream(m.detail.Stream, 4) {
			lines = append(lines, renderStreamItem(item))
		}
	}
	for _, event := range tailEvents(m.events, 8) {
		lines = append(lines, renderEvent(event))
	}
	return lines
}

// applyEvent 根据 Gateway 事件更新最小运行状态。
func (m *appModel) applyEvent(event gateway.GatewayEvent) {
	switch event.Type {
	case gateway.EventRunStarted, gateway.EventAssistantDelta, gateway.EventToolStarted, gateway.EventPermissionRequested, gateway.EventUserQuestionRequested:
		m.status = statusRunning
	case gateway.EventRunFinished, gateway.EventToolFinished, gateway.EventRunCancelled:
		m.status = statusIdle
	case gateway.EventGatewayOffline, gateway.EventError:
		m.status = statusOffline
	}
}

// renderStreamItem 将会话历史记录转换为无边框的单行消息。
func renderStreamItem(item gateway.StreamItem) string {
	marker := "  •"
	if item.Role == "user" {
		marker = "  ›"
	}
	return mutedStyle.Render(marker+" "+item.Role+": ") + item.Text
}

// renderEvent 将 Gateway 事件转换为 Ghost Console 的单行状态流。
func renderEvent(event gateway.GatewayEvent) string {
	label := string(event.Type)
	text := payloadText(event.Payload)
	if text != "" {
		label += "  " + text
	}
	switch event.Type {
	case gateway.EventError, gateway.EventGatewayOffline:
		return errorStyle.Render("  × " + label)
	case gateway.EventRunFinished, gateway.EventToolFinished, gateway.EventRunCancelled:
		return idleStyle.Render("  ✓ " + label)
	default:
		return statusStyle.Render("  ◉ " + label)
	}
}

// payloadText 从事件 payload 中提取最适合在状态流中展示的摘要文本。
func payloadText(payload map[string]any) string {
	for _, key := range []string{"text", "message", "phase", "tool", "question"} {
		if value, ok := payload[key]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
}

// tailStream 返回最近的流历史记录，避免占位界面在小终端中无限增长。
func tailStream(items []gateway.StreamItem, limit int) []gateway.StreamItem {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

// tailEvents 返回最近的 Gateway 事件，保持 Ghost Console 状态流紧凑。
func tailEvents(items []gateway.GatewayEvent, limit int) []gateway.GatewayEvent {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

type initialLoadedMsg struct {
	health   string
	sessions []gateway.SessionSummary
	detail   *gateway.SessionDetail
	models   []gateway.ModelInfo
	eventCh  <-chan gateway.GatewayEvent
	errText  string
}

type gatewayEventMsg struct {
	event  gateway.GatewayEvent
	closed bool
}

// loadInitialCmd 通过 Gateway 客户端加载初始状态，并建立首个会话的事件订阅。
func loadInitialCmd(client gateway.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		msg := initialLoadedMsg{}
		health, err := client.Health(ctx)
		if err != nil {
			msg.errText = err.Error()
		} else if health != nil {
			msg.health = fmt.Sprintf("health:%s  backend:%s", health.Status, health.Backend)
		}
		sessions, err := client.ListSessions(ctx)
		if err != nil {
			msg.errText = err.Error()
			return msg
		}
		msg.sessions = sessions
		models, err := client.ListModels(ctx)
		if err != nil {
			msg.errText = err.Error()
			return msg
		}
		msg.models = models
		if len(sessions) == 0 {
			return msg
		}
		detail, err := client.LoadSession(ctx, sessions[0].ID)
		if err != nil {
			msg.errText = err.Error()
			return msg
		}
		msg.detail = detail
		eventCh, err := client.SubscribeEvents(ctx, sessions[0].ID)
		if err != nil {
			msg.errText = err.Error()
			return msg
		}
		msg.eventCh = eventCh
		return msg
	}
}

// waitEventCmd 等待 Gateway 事件 channel 的下一条事件，保持异步事件逐条进入 Update。
func waitEventCmd(events <-chan gateway.GatewayEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		return gatewayEventMsg{event: event, closed: !ok}
	}
}
