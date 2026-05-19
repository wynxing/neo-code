// Package fakegateway 提供面向 TUI v2 Gateway 客户端接口的 Fake 实现。
package fakegateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"neo-code/internal/tuiv2/gateway"
)

var (
	errGatewayOffline = errors.New("fake gateway offline")
	errClientClosed   = errors.New("fake gateway client closed")
)

// Config 描述 Fake Gateway 客户端的启动参数。
type Config struct {
	Scenario string
}

// FakeGatewayClient 模拟 Gateway 客户端的 RPC 响应、异步事件流、延迟和失败状态。
type FakeGatewayClient struct {
	mu       sync.RWMutex
	closed   bool
	scenario fakeScenario
	modelID  string
}

var _ gateway.Client = (*FakeGatewayClient)(nil)

// New 创建 Fake Gateway 客户端，占位实现只校验场景，不向 UI 暴露硬编码数据。
func New(cfg Config) (gateway.Client, error) {
	return NewFakeClient(cfg.Scenario)
}

// NewFakeClient 根据场景名创建 Fake Gateway 客户端。
func NewFakeClient(scenario string) (*FakeGatewayClient, error) {
	data, ok := scenarioByName(scenario)
	if !ok {
		return nil, fmt.Errorf("unknown fake gateway scenario %q", scenario)
	}
	return &FakeGatewayClient{scenario: data, modelID: data.currentModel}, nil
}

// Health 返回 Fake Gateway 的健康检查结果，离线场景会模拟连接失败。
func (c *FakeGatewayClient) Health(ctx context.Context) (*gateway.HealthResult, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	health := c.scenario.health
	return &health, nil
}

// ListSessions 返回当前场景的会话摘要列表。
func (c *FakeGatewayClient) ListSessions(ctx context.Context) ([]gateway.SessionSummary, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	return append([]gateway.SessionSummary(nil), c.scenario.sessions...), nil
}

// LoadSession 返回当前场景中的会话详情。
func (c *FakeGatewayClient) LoadSession(ctx context.Context, id string) (*gateway.SessionDetail, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	detail, ok := c.scenario.details[id]
	if !ok {
		return nil, fmt.Errorf("fake session %q not found", id)
	}
	detail.Stream = append([]gateway.StreamItem(nil), detail.Stream...)
	return &detail, nil
}

// CreateSession 返回当前场景的首个会话摘要，空会话场景会创建一个临时会话。
func (c *FakeGatewayClient) CreateSession(ctx context.Context) (*gateway.SessionSummary, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	if len(c.scenario.sessions) > 0 {
		summary := c.scenario.sessions[0]
		return &summary, nil
	}
	summary := defaultSessionSummary()
	return &summary, nil
}

// SendMessage 返回当前场景配置的 run 确认。
func (c *FakeGatewayClient) SendMessage(ctx context.Context, sessionID string, text string) (*gateway.RunAck, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	ack := c.scenario.sendAck
	ack.SessionID = sessionID
	ack.Message = text
	return &ack, nil
}

// CancelRun 模拟取消当前 run。
func (c *FakeGatewayClient) CancelRun(ctx context.Context, sessionID string, runID string) error {
	return c.beforeRPC(ctx)
}

// SubscribeEvents 按场景预设时间序列异步推送 GatewayEvent。
func (c *FakeGatewayClient) SubscribeEvents(ctx context.Context, sessionID string) (<-chan gateway.GatewayEvent, error) {
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	if c.scenario.healthError != nil {
		return nil, c.scenario.healthError
	}
	events := make(chan gateway.GatewayEvent)
	scheduled := append([]scheduledEvent(nil), c.scenario.events...)
	go func() {
		defer close(events)
		for _, item := range scheduled {
			if !sleepContext(ctx, item.after) {
				return
			}
			next := item.event
			if next.SessionID == "" {
				next.SessionID = sessionID
			}
			select {
			case <-ctx.Done():
				return
			case events <- next:
			}
		}
	}()
	return events, nil
}

// ResolvePermission 接收 Fake Gateway 的权限决策结果。
func (c *FakeGatewayClient) ResolvePermission(ctx context.Context, decision gateway.PermissionDecision) error {
	return c.beforeRPC(ctx)
}

// AnswerUserQuestion 接收 Fake Gateway 的 ask_user 回答。
func (c *FakeGatewayClient) AnswerUserQuestion(ctx context.Context, answer gateway.UserQuestionAnswer) error {
	return c.beforeRPC(ctx)
}

// ListModels 返回当前场景的模型列表。
func (c *FakeGatewayClient) ListModels(ctx context.Context) ([]gateway.ModelInfo, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return nil, err
	}
	return append([]gateway.ModelInfo(nil), c.scenario.models...), nil
}

// SetModel 模拟切换当前会话模型。
func (c *FakeGatewayClient) SetModel(ctx context.Context, sessionID string, modelID string) error {
	if err := c.beforeRPC(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelID = modelID
	return nil
}

// GetModel 返回 Fake Gateway 的当前模型 ID。
func (c *FakeGatewayClient) GetModel(ctx context.Context, sessionID string) (string, error) {
	if err := c.beforeRPC(ctx); err != nil {
		return "", err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modelID, nil
}

// Close 标记 Fake Gateway 客户端已关闭。
func (c *FakeGatewayClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// beforeRPC 模拟 RPC 延迟，并在请求前检查 context 与关闭状态。
func (c *FakeGatewayClient) beforeRPC(ctx context.Context) error {
	if err := c.checkOpen(); err != nil {
		return err
	}
	if !sleepContext(ctx, c.scenario.rpcDelay) {
		return ctx.Err()
	}
	if c.scenario.healthError != nil {
		return c.scenario.healthError
	}
	return c.checkOpen()
}

// checkOpen 判断 Fake Gateway 客户端是否仍可接收请求。
func (c *FakeGatewayClient) checkOpen() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return errClientClosed
	}
	return nil
}

// sleepContext 等待指定延迟，同时响应 context 取消。
func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
