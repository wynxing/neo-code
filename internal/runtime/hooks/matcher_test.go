package hooks

import "testing"

func TestHasHookMatcherConfig(t *testing.T) {
	t.Parallel()

	if HasHookMatcherConfig(nil) {
		t.Fatal("nil matcher config should be false")
	}
	if HasHookMatcherConfig(map[string]any{}) {
		t.Fatal("empty matcher config should be false")
	}
	if !HasHookMatcherConfig(map[string]any{"tool_name": "bash"}) {
		t.Fatal("tool_name matcher should be true")
	}
	if !HasHookMatcherConfig(map[string]any{"tool_name_regex": []any{"^bash$"}}) {
		t.Fatal("tool_name_regex matcher should be true")
	}
	if !HasHookMatcherConfig(map[string]any{"arguments_contains": []string{"rm -rf"}}) {
		t.Fatal("arguments_contains matcher should be true")
	}
}

func TestCompileHookMatcherAndMatch(t *testing.T) {
	t.Parallel()

	matcher, err := CompileHookMatcher(HookPointBeforeToolCall, map[string]any{
		"tool_name":          []any{"bash", "filesystem"},
		"tool_name_regex":    []string{`^(bash|shell)$`},
		"arguments_contains": []string{"rm -rf"},
	})
	if err != nil {
		t.Fatalf("CompileHookMatcher() error = %v", err)
	}
	if matcher == nil {
		t.Fatal("expected matcher to be compiled")
	}

	if !matcher.Match(HookContext{
		Metadata: map[string]any{
			"tool_name":              "bash",
			"tool_arguments_preview": "sudo rm -rf /tmp/test",
		},
	}) {
		t.Fatal("expected matcher to pass for matching metadata")
	}
	if matcher.Match(HookContext{
		Metadata: map[string]any{
			"tool_name":              "bash",
			"tool_arguments_preview": "echo hello",
		},
	}) {
		t.Fatal("expected matcher to fail when arguments_contains not matched")
	}
	if matcher.Match(HookContext{
		Metadata: map[string]any{
			"tool_name":              "filesystem",
			"tool_arguments_preview": "rm -rf /tmp",
		},
	}) {
		t.Fatal("expected matcher to fail when tool_name_regex not matched")
	}
}

func TestCompileHookMatcherValidation(t *testing.T) {
	t.Parallel()

	if _, err := CompileHookMatcher(HookPointSessionStart, map[string]any{
		"tool_name": "bash",
	}); err == nil {
		t.Fatal("expected session_start tool_name matcher to be rejected")
	}

	if _, err := CompileHookMatcher(HookPointAfterToolResult, map[string]any{
		"arguments_contains": []string{"rm -rf"},
	}); err == nil {
		t.Fatal("expected after_tool_result arguments_contains to be rejected")
	}

	if _, err := CompileHookMatcher(HookPointBeforeToolCall, map[string]any{
		"tool_name_regex": "(",
	}); err == nil {
		t.Fatal("expected invalid regex to fail")
	}

	longRegex := make([]byte, MaxHookMatcherRegexLength+1)
	for i := range longRegex {
		longRegex[i] = 'a'
	}
	if _, err := CompileHookMatcher(HookPointBeforeToolCall, map[string]any{
		"tool_name_regex": string(longRegex),
	}); err == nil {
		t.Fatal("expected overlong regex to fail")
	}
}
