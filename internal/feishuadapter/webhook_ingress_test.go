package feishuadapter

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewWebhookIngressAndRunContextCancel(t *testing.T) {
	ingress, ok := NewWebhookIngress(Config{
		ListenAddress: "127.0.0.1:0",
		EventPath:     "/events",
		CardPath:      "/cards",
	}, nil).(*WebhookIngress)
	if !ok {
		t.Fatal("expected webhook ingress instance")
	}
	if ingress.nowFn == nil {
		t.Fatal("expected default nowFn")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ingress.Run(ctx, &captureIngressHandler{})
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for webhook ingress shutdown")
	}
}

func TestNewWebhookIngressUsesProvidedClockAndRunReturnsListenError(t *testing.T) {
	fixed := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return fixed }
	ingress, ok := NewWebhookIngress(Config{
		ListenAddress: "bad::addr",
		EventPath:     "/events",
		CardPath:      "/cards",
	}, nowFn).(*WebhookIngress)
	if !ok {
		t.Fatal("expected webhook ingress instance")
	}
	if got := ingress.nowFn(); !got.Equal(fixed) {
		t.Fatalf("expected provided nowFn, got %v", got)
	}
	if err := ingress.Run(context.Background(), &captureIngressHandler{}); err == nil {
		t.Fatal("expected listen error for invalid address")
	}
}
