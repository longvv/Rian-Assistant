package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// SysinfoTool retrieves cross-platform system resource metrics.
type SysinfoTool struct{}

// NewSysinfoTool creates a new instance of the SysinfoTool.
func NewSysinfoTool() *SysinfoTool {
	return &SysinfoTool{}
}

// Name returns the tool's name.
func (t *SysinfoTool) Name() string { return "sysinfo" }

// Description returns the tool's description for the LLM.
func (t *SysinfoTool) Description() string {
	return "Retrieves system resource metrics (CPU, Memory, Disk usage) cleanly without shell commands. Returns a formatted JSON object with percentages."
}

// Parameters defines the expected JSON schema (none required).
func (t *SysinfoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

// SysinfoResult holds the system metrics payload.
type SysinfoResult struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemUsedPercent  float64 `json:"memory_used_percent"`
	DiskUsedPercent float64 `json:"disk_used_percent"`
}

// Execute performs the metric collection.
func (t *SysinfoTool) Execute(_ context.Context, _ map[string]interface{}) *ToolResult {
	var res SysinfoResult

	// CPU: Calculate total usage over a 100ms interval for a quick and accurate reading
	if percents, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(percents) > 0 {
		res.CPUUsagePercent = percents[0]
	}

	// Memory: Fetch total virtual memory usage
	if v, err := mem.VirtualMemory(); err == nil {
		res.MemUsedPercent = v.UsedPercent
	}

	// Disk: Fetch usage for the root partition
	if d, err := disk.Usage("/"); err == nil {
		res.DiskUsedPercent = d.UsedPercent
	}

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal sysinfo results: %v", err))
	}

	resultStr := string(b)
	return &ToolResult{
		ForLLM:  resultStr,
		ForUser: resultStr,
	}
}
