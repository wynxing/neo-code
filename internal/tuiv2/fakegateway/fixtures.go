package fakegateway

import (
	"fmt"
	"time"

	"neo-code/internal/tuiv2/gateway"
)

const (
	defaultSessionID = "session-ghost-console"
	defaultRunID     = "run-ghost-001"
	defaultModelID   = "neo-fake-pro"
	tick             = 10 * time.Millisecond
)

var fixtureTime = time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)

// defaultSessionSummary 返回 Ghost Console 演示场景的默认会话摘要。
func defaultSessionSummary() gateway.SessionSummary {
	return gateway.SessionSummary{
		ID:        defaultSessionID,
		Title:     "Ghost Console demo",
		Mode:      "input",
		Model:     defaultModelID,
		UpdatedAt: fixtureTime,
	}
}

// defaultSessionDetail 返回包含基础对话历史的默认会话详情。
func defaultSessionDetail() gateway.SessionDetail {
	return detailWithStream([]gateway.StreamItem{
		userItem("msg-user-1", "Refactor the TUI without touching v1."),
		assistantItem("msg-agent-1", "Plan accepted. I will keep all data behind the Gateway client contract."),
	})
}

// detailWithStream 使用给定流历史构造默认会话详情。
func detailWithStream(stream []gateway.StreamItem) gateway.SessionDetail {
	return gateway.SessionDetail{
		Summary: defaultSessionSummary(),
		Stream:  append([]gateway.StreamItem(nil), stream...),
		Usage:   gateway.TokenUsage{Input: 128, Output: 256, Total: 384},
	}
}

// defaultModels 返回 Fake Gateway 暴露给 TUI 的模型列表。
func defaultModels() []gateway.ModelInfo {
	return []gateway.ModelInfo{
		{ID: defaultModelID, Name: "Neo Fake Pro", Provider: "fake", Current: true, Capabilities: []string{"tools", "streaming"}},
		{ID: "neo-fake-fast", Name: "Neo Fake Fast", Provider: "fake", Current: false, Capabilities: []string{"streaming"}},
	}
}

// defaultEvents 返回默认场景的完整演示事件序列。
func defaultEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "running"))},
		{after: tick, event: event(gateway.EventAssistantDelta, defaultSessionID, defaultRunID, payload("text", "Ghost Console ready."))},
		{after: tick, event: event(gateway.EventToolStarted, defaultSessionID, defaultRunID, payload("tool", "filesystem.read"))},
		{after: tick, event: event(gateway.EventToolFinished, defaultSessionID, defaultRunID, payload("tool", "filesystem.read", "status", "ok"))},
		{after: tick, event: event(gateway.EventRunFinished, defaultSessionID, defaultRunID, payload("tokens", 384))},
	}
}

// streamingEvents 返回逐块输出的助手流式事件。
func streamingEvents() []scheduledEvent {
	parts := []string{"Streaming ", "chat ", "arrives ", "chunk ", "by ", "chunk."}
	events := []scheduledEvent{{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "streaming"))}}
	for _, part := range parts {
		events = append(events, scheduledEvent{
			after: tick,
			event: event(gateway.EventAssistantDelta, defaultSessionID, defaultRunID, payload("text", part)),
		})
	}
	events = append(events, scheduledEvent{after: tick, event: event(gateway.EventRunFinished, defaultSessionID, defaultRunID, payload("phase", "done"))})
	return events
}

// toolApprovalEvents 返回工具权限等待流程的事件序列。
func toolApprovalEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "running"))},
		{after: tick, event: event(gateway.EventToolStarted, defaultSessionID, defaultRunID, payload("tool", "bash", "command", "go test ./..."))},
		{after: tick, event: event(gateway.EventPermissionRequested, defaultSessionID, defaultRunID, payload("request_id", "perm-001", "tool", "bash"))},
	}
}

// toolFailedEvents 返回工具执行失败流程的事件序列。
func toolFailedEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventToolStarted, defaultSessionID, defaultRunID, payload("tool", "webfetch"))},
		{after: tick, event: event(gateway.EventToolFinished, defaultSessionID, defaultRunID, payload("tool", "webfetch", "status", "failed"))},
		{after: tick, event: event(gateway.EventError, defaultSessionID, defaultRunID, payload("message", "tool failed: timeout"))},
	}
}

// runtimeErrorEvents 返回 Runtime 错误流程的事件序列。
func runtimeErrorEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "running"))},
		{after: tick, event: event(gateway.EventError, defaultSessionID, defaultRunID, payload("message", "runtime error: provider stream interrupted"))},
		{after: tick, event: event(gateway.EventRunFinished, defaultSessionID, defaultRunID, payload("phase", "failed"))},
	}
}

// askUserEvents 返回 ask_user 问答等待流程的事件序列。
func askUserEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "running"))},
		{after: tick, event: event(gateway.EventUserQuestionRequested, defaultSessionID, defaultRunID, payload("question_id", "ask-001", "question", "Which branch should I use?"))},
	}
}

// cancelRunningEvents 返回运行中取消流程的事件序列。
func cancelRunningEvents() []scheduledEvent {
	return []scheduledEvent{
		{after: tick, event: event(gateway.EventRunStarted, defaultSessionID, defaultRunID, payload("phase", "running"))},
		{after: 2 * tick, event: event(gateway.EventAssistantDelta, defaultSessionID, defaultRunID, payload("text", "Working..."))},
		{after: tick, event: event(gateway.EventRunCancelled, defaultSessionID, defaultRunID, payload("phase", "cancelled"))},
	}
}

// longStreamItems 返回大量历史消息，用于滚动区域验证。
func longStreamItems() []gateway.StreamItem {
	items := make([]gateway.StreamItem, 0, 32)
	for i := 1; i <= 32; i++ {
		items = append(items, assistantItem(fmt.Sprintf("long-%02d", i), fmt.Sprintf("Long output line %02d for scroll testing.", i)))
	}
	return items
}

// manySessionSummaries 返回大量会话摘要，用于 picker 场景验证。
func manySessionSummaries() []gateway.SessionSummary {
	sessions := make([]gateway.SessionSummary, 0, 24)
	for i := 1; i <= 24; i++ {
		sessions = append(sessions, gateway.SessionSummary{
			ID:        fmt.Sprintf("session-%02d", i),
			Title:     fmt.Sprintf("Session %02d", i),
			Mode:      "input",
			Model:     defaultModelID,
			UpdatedAt: fixtureTime.Add(time.Duration(i) * time.Minute),
		})
	}
	return sessions
}

// detailsForSessions 为会话摘要批量生成对应详情。
func detailsForSessions(sessions []gateway.SessionSummary) map[string]gateway.SessionDetail {
	details := make(map[string]gateway.SessionDetail, len(sessions))
	for _, summary := range sessions {
		details[summary.ID] = gateway.SessionDetail{
			Summary: summary,
			Stream:  []gateway.StreamItem{assistantItem("item-"+summary.ID, "Session fixture loaded.")},
			Usage:   gateway.TokenUsage{Input: 10, Output: 20, Total: 30},
		}
	}
	return details
}

// userItem 构造用户角色的历史消息。
func userItem(id string, text string) gateway.StreamItem {
	return streamItem(id, "message", "user", text, "done")
}

// assistantItem 构造助手角色的历史消息。
func assistantItem(id string, text string) gateway.StreamItem {
	return streamItem(id, "message", "assistant", text, "done")
}

// streamItem 构造通用历史流记录。
func streamItem(id string, kind string, role string, text string, status string) gateway.StreamItem {
	return gateway.StreamItem{ID: id, Kind: kind, Role: role, Text: text, Status: status, CreatedAt: fixtureTime}
}

// event 构造带固定时间戳的 Gateway 事件。
func event(kind gateway.EventType, sessionID string, runID string, data map[string]any) gateway.GatewayEvent {
	return gateway.GatewayEvent{Type: kind, SessionID: sessionID, RunID: runID, Payload: data, At: fixtureTime}
}

// payload 将键值参数转换为事件 payload。
func payload(values ...any) map[string]any {
	data := make(map[string]any, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		data[key] = values[i+1]
	}
	return data
}
