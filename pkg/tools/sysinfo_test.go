package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSysinfoTool(t *testing.T) {
	tool := NewSysinfoTool()
	ctx := context.Background()

	res := tool.Execute(ctx, map[string]interface{}{})
	assert.NotNil(t, res)
	assert.False(t, res.IsError, "Sysinfo tool execution should not return an error")

	var parsed SysinfoResult
	err := json.Unmarshal([]byte(res.ForLLM), &parsed)
	assert.NoError(t, err, "Should respond with valid JSON")

	// Verify we are actually populating results (assuming machine isn't completely idle 0.00% across all metrics at once)
	assert.GreaterOrEqual(t, parsed.CPUUsagePercent, 0.0, "CPU percentage should be >= 0")
	assert.Greater(t, parsed.MemUsedPercent, 0.0, "Memory usage should be > 0 on an active system")
	assert.Greater(t, parsed.DiskUsedPercent, 0.0, "Root disk usage should be > 0 on an active system")
}
