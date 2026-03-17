package handlers

import (
	"net/http"

	"github.com/pinchtab/pinchtab/internal/dashboard"
	"github.com/pinchtab/pinchtab/internal/web"
)

// HandleViewer serves the interactive browser viewer page.
// In dashboard mode this serves the React SPA; in bridge-only mode it
// returns 503 since the dashboard is not built.
//
// GET /viewer?apiToken=<token>&tabId=<tab-id>
func (h *Handlers) HandleViewer(w http.ResponseWriter, r *http.Request) {
	if !h.Config.AllowScreencast {
		web.ErrorCode(w, 403, "screencast_disabled", web.DisabledEndpointMessage("screencast", "security.allowScreencast"), false, map[string]any{
			"setting": "security.allowScreencast",
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if html := dashboard.SPAHTML(); html != nil {
		_, _ = w.Write(html)
	} else {
		http.Error(w, "Viewer requires the dashboard build. Use the screencast WebSocket API directly.", 503)
	}
}
