package health

import (
	"sync"
	"sync/atomic"
)

var (
	LLMCallsTotal   atomic.Uint64
	ErrorsTotal     atomic.Uint64
	TokensEstimated atomic.Uint64

	toolMu         sync.RWMutex
	toolExecutions = make(map[string]uint64)
)

func RecordLLMCall() {
	LLMCallsTotal.Add(1)
}

func RecordError() {
	ErrorsTotal.Add(1)
}

func RecordTokens(n int) {
	if n > 0 {
		TokensEstimated.Add(uint64(n))
	}
}

func RecordToolExecution(toolName string) {
	toolMu.Lock()
	defer toolMu.Unlock()
	toolExecutions[toolName]++
}

type MetricsSnapshot struct {
	LLMCallsTotal   uint64            `json:"llm_calls_total"`
	ErrorsTotal     uint64            `json:"errors_total"`
	TokensEstimated uint64            `json:"tokens_estimated"`
	ToolExecutions  map[string]uint64 `json:"tool_executions"`
}

func GetMetrics() MetricsSnapshot {
	toolMu.RLock()
	defer toolMu.RUnlock()

	tools := make(map[string]uint64, len(toolExecutions))
	for k, v := range toolExecutions {
		tools[k] = v
	}

	return MetricsSnapshot{
		LLMCallsTotal:   LLMCallsTotal.Load(),
		ErrorsTotal:     ErrorsTotal.Load(),
		TokensEstimated: TokensEstimated.Load(),
		ToolExecutions:  tools,
	}
}
