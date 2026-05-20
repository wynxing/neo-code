package theme

import (
	"os"
	"strings"
)

const (
	PhaseIdle              = "idle"
	PhaseRunning           = "running"
	PhaseWaiting           = "waiting"
	PhaseWaitingPermission = "waiting_permission"
	PhaseWaitingUser       = "waiting_user"
	PhaseCancelled         = "cancelled"
	PhaseError             = "error"
)

// SymbolSet 定义 Ghost Console 的语义符号集合。
type SymbolSet struct {
	Success   string
	Running   string
	Idle      string
	Waiting   string
	Error     string
	Separator string
	AccentBar string
}

// UnicodeSymbols 是默认符号集合。
var UnicodeSymbols = SymbolSet{
	Success:   "✓",
	Running:   "◉",
	Idle:      "○",
	Waiting:   "◌",
	Error:     "×",
	Separator: "·",
	AccentBar: "▎",
}

// ASCIISymbols 是不支持 Unicode 或 Nerd Font 终端的降级符号集合。
var ASCIISymbols = SymbolSet{
	Success:   "[OK]",
	Running:   "(*)",
	Idle:      "( )",
	Waiting:   "(~)",
	Error:     "[X]",
	Separator: ".",
	AccentBar: "|",
}

// Symbols 返回当前环境下应使用的符号集合。
func Symbols() SymbolSet {
	if DetectASCIISymbols() {
		return ASCIISymbols
	}
	return UnicodeSymbols
}

// DetectASCIISymbols 判断当前环境是否应降级为 ASCII 符号。
func DetectASCIISymbols() bool {
	return DetectASCIISymbolsFromEnv(os.Getenv)
}

// DetectASCIISymbolsFromEnv 使用给定 getenv 函数判断符号能力，便于测试。
func DetectASCIISymbolsFromEnv(getenv func(string) string) bool {
	if strings.EqualFold(getenv("NEOCODE_TUI_ASCII"), "1") {
		return true
	}
	term := strings.ToLower(getenv("TERM"))
	return term == "dumb" || term == "linux"
}

// StatusSymbol 返回指定运行阶段的状态符号。
func StatusSymbol(phase string) string {
	symbols := Symbols()
	switch phase {
	case PhaseRunning:
		return symbols.Running
	case PhaseWaiting, PhaseWaitingPermission, PhaseWaitingUser:
		return symbols.Waiting
	case PhaseError:
		return symbols.Error
	case PhaseCancelled:
		return symbols.Waiting
	default:
		return symbols.Idle
	}
}

// AccentBar 返回缩进指示条。
func AccentBar() string {
	return Symbols().AccentBar
}

// StreamPrefix 返回不同 StreamEntry 类型的语义前缀。
func StreamPrefix(entryType string) string {
	symbols := Symbols()
	switch entryType {
	case "tool_finished", "tool_end", "run_finished", "run_cancelled":
		return symbols.Success
	case "tool_started", "tool_start", "assistant_delta", "run_started", "permission_requested", "question":
		return symbols.Running
	case "error", "gateway_offline":
		return symbols.Error
	case "message":
		return symbols.Idle
	default:
		return symbols.Waiting
	}
}

// Separator 返回行内弱分隔符。
func Separator() string {
	return Symbols().Separator
}
