package bridge

import (
	"context"

	"github.com/chromedp/cdproto/performance"
	"github.com/chromedp/chromedp"
)

// MemoryMetrics holds Chrome memory statistics
type MemoryMetrics struct {
	JSHeapUsedMB  float64 `json:"jsHeapUsedMB"`
	JSHeapTotalMB float64 `json:"jsHeapTotalMB"`
	Documents     int64   `json:"documents"`
	Frames        int64   `json:"frames"`
	Nodes         int64   `json:"nodes"`
	Listeners     int64   `json:"listeners"`
}

// GetMemoryMetrics retrieves memory metrics for a specific tab
func (b *Bridge) GetMemoryMetrics(tabID string) (*MemoryMetrics, error) {
	ctx, _, err := b.TabContext(tabID)
	if err != nil {
		return nil, err
	}

	return getMetricsFromContext(ctx)
}

// GetBrowserMemoryMetrics retrieves memory metrics for the entire browser
func (b *Bridge) GetBrowserMemoryMetrics() (*MemoryMetrics, error) {
	if b.BrowserCtx == nil {
		return nil, nil
	}
	return getMetricsFromContext(b.BrowserCtx)
}

func getMetricsFromContext(ctx context.Context) (*MemoryMetrics, error) {
	// Enable performance metrics collection
	if err := chromedp.Run(ctx, performance.Enable()); err != nil {
		return nil, err
	}

	var metrics []*performance.Metric
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		metrics, err = performance.GetMetrics().Do(ctx)
		return err
	})); err != nil {
		return nil, err
	}

	result := &MemoryMetrics{}
	for _, m := range metrics {
		switch m.Name {
		case "JSHeapUsedSize":
			result.JSHeapUsedMB = m.Value / (1024 * 1024)
		case "JSHeapTotalSize":
			result.JSHeapTotalMB = m.Value / (1024 * 1024)
		case "Documents":
			result.Documents = int64(m.Value)
		case "Frames":
			result.Frames = int64(m.Value)
		case "Nodes":
			result.Nodes = int64(m.Value)
		case "JSEventListeners":
			result.Listeners = int64(m.Value)
		}
	}

	return result, nil
}
