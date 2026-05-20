package gateway

import "testing"

func TestEventTypeValues(t *testing.T) {
	tests := map[EventType]string{
		EventAgentChunk:            "agent_chunk",
		EventAgentMessageStart:     "agent_message_start",
		EventAgentMessageEnd:       "agent_message_end",
		EventToolStart:             "tool_start",
		EventToolEnd:               "tool_end",
		EventToolOutput:            "tool_output",
		EventSessionUpdated:        "session_updated",
		EventSessionCreated:        "session_created",
		EventSessionDeleted:        "session_deleted",
		EventRunStarted:            "run_started",
		EventRunFinished:           "run_finished",
		EventRunError:              "run_error",
		EventRunCancelled:          "run_cancelled",
		EventTokenUsage:            "token_usage",
		EventPhaseChanged:          "phase_changed",
		EventAssistantDelta:        "assistant_delta",
		EventToolStarted:           "tool_started",
		EventToolFinished:          "tool_finished",
		EventPermissionRequested:   "permission_requested",
		EventPermissionResolved:    "permission_resolved",
		EventAskUserQuestion:       "ask_user_question",
		EventUserQuestionRequested: "user_question_requested",
		EventModelChanged:          "model_changed",
		EventHealthChanged:         "health_changed",
		EventGatewayOffline:        "gateway_offline",
		EventError:                 "error",
	}

	for got, want := range tests {
		if string(got) != want {
			t.Fatalf("event value = %q, want %q", got, want)
		}
	}
}

func TestRealClientSatisfiesClient(t *testing.T) {
	var client Client = NewRealClient()
	if client == nil {
		t.Fatal("NewRealClient() = nil")
	}
}
