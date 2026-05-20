package state

import (
	"testing"

	"neo-code/internal/tuiv2/gateway"
)

func TestReduceAgentChunkMergesUnfinishedMessage(t *testing.T) {
	current := NewViewState()
	first := Reduce(current, event(gateway.EventAgentChunk, map[string]any{"text": "hel"}))
	second := Reduce(first, event(gateway.EventAgentChunk, map[string]any{"text": "lo"}))

	if len(second.Stream) != 1 {
		t.Fatalf("stream len = %d, want 1", len(second.Stream))
	}
	if second.Stream[0].Content != "hello" {
		t.Fatalf("content = %q, want hello", second.Stream[0].Content)
	}
	if len(first.Stream) != 1 && first.Stream[0].Content != "hel" {
		t.Fatalf("input state was mutated: %+v", first.Stream)
	}
}

func TestReduceAgentChunkCreatesNewMessageAfterDoneOrNonMessage(t *testing.T) {
	doneState := Reduce(NewViewState(), event(gateway.EventAgentMessageStart, map[string]any{"text": "done"}))
	doneState = Reduce(doneState, event(gateway.EventAgentMessageEnd, map[string]any{}))
	next := Reduce(doneState, event(gateway.EventAgentChunk, map[string]any{"text": "new"}))
	if len(next.Stream) != 2 {
		t.Fatalf("stream len after done = %d, want 2", len(next.Stream))
	}

	toolState := Reduce(NewViewState(), event(gateway.EventToolStart, map[string]any{"tool": "bash"}))
	next = Reduce(toolState, event(gateway.EventAgentChunk, map[string]any{"text": "message"}))
	if len(next.Stream) != 2 || next.Stream[1].Type != "message" {
		t.Fatalf("chunk after tool = %+v, want new message", next.Stream)
	}
}

func TestReduceDoesNotMutateInputSlices(t *testing.T) {
	current := NewViewState()
	current.Stream = []StreamEntry{{ID: "old", Type: "message", Content: "old", Metadata: map[string]any{"done": true}}}
	current.Gateway.Sessions = []gateway.SessionSummary{{ID: "s1", Title: "one"}}

	next := Reduce(current, event(gateway.EventSessionCreated, map[string]any{"id": "s2", "title": "two"}))
	if len(current.Gateway.Sessions) != 1 {
		t.Fatalf("input sessions mutated: %+v", current.Gateway.Sessions)
	}
	if len(next.Gateway.Sessions) != 2 {
		t.Fatalf("next sessions len = %d, want 2", len(next.Gateway.Sessions))
	}

	next = Reduce(current, event(gateway.EventAgentChunk, map[string]any{"text": "new"}))
	if len(current.Stream) != 1 || current.Stream[0].Content != "old" {
		t.Fatalf("input stream mutated: %+v", current.Stream)
	}
	if len(next.Stream) != 2 {
		t.Fatalf("next stream len = %d, want 2", len(next.Stream))
	}
}

func TestReduceCoversEventStateTransitions(t *testing.T) {
	tests := []struct {
		name   string
		event  gateway.GatewayEvent
		assert func(t *testing.T, next *ViewState)
	}{
		{
			name:  "agent_message_start",
			event: event(gateway.EventAgentMessageStart, map[string]any{"text": "start"}),
			assert: func(t *testing.T, next *ViewState) {
				assertLastEntry(t, next, "message", "start")
			},
		},
		{
			name:  "agent_message_end",
			event: event(gateway.EventAgentMessageEnd, map[string]any{"input": 1, "output": 2, "total": 3}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Tokens.Total != 3 {
					t.Fatalf("tokens = %+v, want total 3", next.Runtime.Tokens)
				}
			},
		},
		{
			name:  "tool_start",
			event: event(gateway.EventToolStart, map[string]any{"tool": "bash", "input": "go test"}),
			assert: func(t *testing.T, next *ViewState) {
				assertLastEntry(t, next, "tool_start", "go test")
				if next.Stream[len(next.Stream)-1].ToolName != "bash" {
					t.Fatalf("tool name = %q, want bash", next.Stream[len(next.Stream)-1].ToolName)
				}
			},
		},
		{
			name:  "tool_end",
			event: event(gateway.EventToolEnd, map[string]any{"tool": "bash", "output": "ok"}),
			assert: func(t *testing.T, next *ViewState) {
				assertLastEntry(t, next, "tool_end", "ok")
			},
		},
		{
			name:  "tool_output",
			event: event(gateway.EventToolOutput, map[string]any{"output": "line"}),
			assert: func(t *testing.T, next *ViewState) {
				assertLastEntry(t, next, "tool_output", "line")
			},
		},
		{
			name:  "permission_requested",
			event: event(gateway.EventPermissionRequested, map[string]any{"tool": "bash"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseWaitingPermission || next.Input.Mode != InputStateModePermissionResponse {
					t.Fatalf("permission state = phase %q input %q", next.Runtime.Phase, next.Input.Mode)
				}
				assertLastEntry(t, next, "permission", "bash")
			},
		},
		{
			name:  "permission_resolved",
			event: event(gateway.EventPermissionResolved, map[string]any{"decision": "allow"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseRunning {
					t.Fatalf("phase = %q, want running", next.Runtime.Phase)
				}
				assertLastEntry(t, next, "status", "allow")
			},
		},
		{
			name:  "ask_user_question",
			event: event(gateway.EventAskUserQuestion, map[string]any{"question": "branch?", "options": []any{"main", "dev"}}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseWaitingUser || next.Input.Mode != InputStateModeQuestionAnswer {
					t.Fatalf("ask state = phase %q input %q", next.Runtime.Phase, next.Input.Mode)
				}
				if len(next.Input.Options) != 2 {
					t.Fatalf("options = %+v, want two", next.Input.Options)
				}
			},
		},
		{
			name:  "phase_changed",
			event: event(gateway.EventPhaseChanged, map[string]any{"phase": RuntimePhaseWaitingPermission}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseWaitingPermission {
					t.Fatalf("phase = %q", next.Runtime.Phase)
				}
			},
		},
		{
			name:  "run_started",
			event: withRun(event(gateway.EventRunStarted, nil), "run-1"),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseRunning || next.Runtime.RunID != "run-1" {
					t.Fatalf("runtime = %+v", next.Runtime)
				}
			},
		},
		{
			name:  "run_finished",
			event: event(gateway.EventRunFinished, map[string]any{"tokens": 9}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseIdle || next.Runtime.Tokens.Total != 9 {
					t.Fatalf("runtime = %+v", next.Runtime)
				}
			},
		},
		{
			name:  "run_error",
			event: event(gateway.EventRunError, map[string]any{"message": "boom"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseError {
					t.Fatalf("phase = %q", next.Runtime.Phase)
				}
				assertLastEntry(t, next, "error", "boom")
			},
		},
		{
			name:  "run_cancelled",
			event: event(gateway.EventRunCancelled, map[string]any{"phase": "cancelled"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Phase != RuntimePhaseCancelled {
					t.Fatalf("phase = %q", next.Runtime.Phase)
				}
				assertLastEntry(t, next, "status", "cancelled")
			},
		},
		{
			name:  "token_usage",
			event: event(gateway.EventTokenUsage, map[string]any{"input_tokens": 3, "output_tokens": 4, "total_tokens": 7}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Runtime.Tokens != (TokenUsage{Input: 3, Output: 4, Total: 7}) {
					t.Fatalf("tokens = %+v", next.Runtime.Tokens)
				}
			},
		},
		{
			name:  "session_created",
			event: event(gateway.EventSessionCreated, map[string]any{"id": "s2", "title": "two"}),
			assert: func(t *testing.T, next *ViewState) {
				if len(next.Gateway.Sessions) != 2 {
					t.Fatalf("sessions = %+v", next.Gateway.Sessions)
				}
			},
		},
		{
			name:  "session_deleted",
			event: event(gateway.EventSessionDeleted, map[string]any{"id": "s1"}),
			assert: func(t *testing.T, next *ViewState) {
				if len(next.Gateway.Sessions) != 0 {
					t.Fatalf("sessions = %+v", next.Gateway.Sessions)
				}
			},
		},
		{
			name:  "session_updated",
			event: event(gateway.EventSessionUpdated, map[string]any{"id": "s1", "title": "updated"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Gateway.Sessions[0].Title != "updated" {
					t.Fatalf("session = %+v", next.Gateway.Sessions[0])
				}
			},
		},
		{
			name:  "model_changed",
			event: event(gateway.EventModelChanged, map[string]any{"model_id": "model-2"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Gateway.ActiveModel != "model-2" {
					t.Fatalf("active model = %q", next.Gateway.ActiveModel)
				}
			},
		},
		{
			name:  "health_changed",
			event: event(gateway.EventHealthChanged, map[string]any{"connected": true}),
			assert: func(t *testing.T, next *ViewState) {
				if !next.Gateway.Connected {
					t.Fatal("connected = false, want true")
				}
			},
		},
		{
			name:  "gateway_offline",
			event: event(gateway.EventGatewayOffline, map[string]any{"message": "offline"}),
			assert: func(t *testing.T, next *ViewState) {
				if next.Gateway.Connected || next.Runtime.Phase != RuntimePhaseError {
					t.Fatalf("state = connected %t phase %q", next.Gateway.Connected, next.Runtime.Phase)
				}
				assertLastEntry(t, next, "error", "offline")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := NewViewState()
			current.Gateway.Sessions = []gateway.SessionSummary{{ID: "s1", Title: "one"}}
			current.Stream = []StreamEntry{{ID: "seed", Type: "message", Content: "seed", Metadata: map[string]any{"done": true}}}
			next := Reduce(current, tt.event)
			tt.assert(t, next)
		})
	}
}

func event(eventType gateway.EventType, payload map[string]any) gateway.GatewayEvent {
	if payload == nil {
		payload = map[string]any{}
	}
	return gateway.GatewayEvent{Type: eventType, Payload: payload}
}

func withRun(event gateway.GatewayEvent, runID string) gateway.GatewayEvent {
	event.RunID = runID
	return event
}

func assertLastEntry(t *testing.T, state *ViewState, entryType string, content string) {
	t.Helper()
	if len(state.Stream) == 0 {
		t.Fatal("stream is empty")
	}
	last := state.Stream[len(state.Stream)-1]
	if last.Type != entryType || last.Content != content {
		t.Fatalf("last entry = %+v, want type %q content %q", last, entryType, content)
	}
}
