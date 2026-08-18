package metrics

import (
	"time"

	dto "github.com/prometheus/client_model/go"
)

// Snapshot is what the mcp_exporter_snapshot tool hands back to an agent —
// mcp-exporter reporting on its own operation, read straight from the same
// registry promhttp serves at /metrics via Gather(), the canonical read path.
type Snapshot struct {
	UptimeSeconds   float64            `json:"uptimeSeconds" jsonschema:"Seconds since this server started."`
	ToolCallsByName map[string]float64 `json:"toolCallsByName" jsonschema:"Total tool calls observed so far, keyed by tool name."`
	ToolErrors      float64            `json:"toolErrors" jsonschema:"Total tool calls that returned an error."`
	MethodCalls     float64            `json:"methodCalls" jsonschema:"Total JSON-RPC method calls of any kind."`
	ResourceReads   float64            `json:"resourceReads" jsonschema:"Total resources/read calls observed so far."`
}

// Gather reads the live registry and summarizes it. Uses the same
// []*dto.MetricFamily path promhttp itself uses to render /metrics.
func (m *Metrics) Gather() Snapshot {
	s := Snapshot{
		UptimeSeconds:   m.Uptime().Seconds(),
		ToolCallsByName: map[string]float64{},
	}

	families, err := m.Registry.Gather()
	if err != nil {
		return s
	}

	for _, fam := range families {
		switch fam.GetName() {
		case "mcp_tool_calls_total":
			for _, metric := range fam.GetMetric() {
				name := labelValue(metric, "tool")
				s.ToolCallsByName[name] += metric.GetCounter().GetValue()
				if labelValue(metric, "status") == "error" {
					s.ToolErrors += metric.GetCounter().GetValue()
				}
			}
		case "mcp_method_calls_total":
			for _, metric := range fam.GetMetric() {
				s.MethodCalls += metric.GetCounter().GetValue()
			}
		case "mcp_resource_reads_total":
			for _, metric := range fam.GetMetric() {
				s.ResourceReads += metric.GetCounter().GetValue()
			}
		}
	}

	return s
}

func labelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}

var _ = time.Second // keep time import if Uptime moves; harmless if unused later
