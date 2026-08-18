// Package metrics instruments mcp-exporter itself — every MCP method call is
// counted and timed via the SDK's own AddReceivingMiddleware hook, then
// exposed at /metrics in Prometheus exposition format.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics holds every Prometheus collector mcp-exporter registers.
type Metrics struct {
	Registry *prometheus.Registry

	MethodTotal    *prometheus.CounterVec
	MethodDuration *prometheus.HistogramVec
	ToolCallsTotal *prometheus.CounterVec
	ToolDuration   *prometheus.HistogramVec
	ResourceReads  *prometheus.CounterVec

	startTime time.Time
}

// New creates a fresh registry with mcp_* collectors plus the standard Go
// runtime/process collectors, and registers everything.
func New(version string) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		Registry:  reg,
		startTime: time.Now(),

		MethodTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcp_method_calls_total",
			Help: "Total JSON-RPC method calls received, by method and outcome.",
		}, []string{"method", "status"}),

		MethodDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mcp_method_duration_seconds",
			Help:    "Duration of JSON-RPC method handling, by method.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),

		ToolCallsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "Total tools/call invocations, by tool name and outcome.",
		}, []string{"tool", "status"}),

		ToolDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mcp_tool_call_duration_seconds",
			Help:    "Duration of tool handler execution, by tool name.",
			Buckets: prometheus.DefBuckets,
		}, []string{"tool"}),

		ResourceReads: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcp_resource_reads_total",
			Help: "Total resources/read calls, by resource URI and outcome.",
		}, []string{"resource", "status"}),
	}

	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mcp_exporter_build_info",
		Help: "Build metadata, value is always 1.",
	}, []string{"version"})
	buildInfo.WithLabelValues(version).Set(1)

	startTimeGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mcp_exporter_start_time_seconds",
		Help: "Unix timestamp when this server started.",
	})
	startTimeGauge.Set(float64(m.startTime.Unix()))

	reg.MustRegister(
		m.MethodTotal, m.MethodDuration,
		m.ToolCallsTotal, m.ToolDuration,
		m.ResourceReads,
		buildInfo, startTimeGauge,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return m
}

// Uptime returns how long this process has been running.
func (m *Metrics) Uptime() time.Duration {
	return time.Since(m.startTime)
}
