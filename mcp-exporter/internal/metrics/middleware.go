package metrics

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Middleware returns an mcp.Middleware that instruments every incoming
// JSON-RPC method call — the same extension point the SDK's own
// examples/server/middleware example uses for logging.
func (m *Metrics) Middleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			result, err := next(ctx, method, req)
			duration := time.Since(start).Seconds()

			status := "ok"
			if err != nil {
				status = "error"
			}

			m.MethodTotal.WithLabelValues(method, status).Inc()
			m.MethodDuration.WithLabelValues(method).Observe(duration)

			if ctr, ok := req.(*mcp.CallToolRequest); ok {
				toolStatus := status
				if err == nil {
					if res, ok := result.(*mcp.CallToolResult); ok && res.IsError {
						toolStatus = "error"
					}
				}
				m.ToolCallsTotal.WithLabelValues(ctr.Params.Name, toolStatus).Inc()
				m.ToolDuration.WithLabelValues(ctr.Params.Name).Observe(duration)
			}

			if rrr, ok := req.(*mcp.ReadResourceRequest); ok {
				m.ResourceReads.WithLabelValues(rrr.Params.URI, status).Inc()
			}

			return result, err
		}
	}
}
