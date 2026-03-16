package handlers

import (
	"net/http"

	"github.com/pinchtab/pinchtab/internal/assets"
	"github.com/pinchtab/pinchtab/internal/web"
)

// HandleViewer serves the interactive browser viewer page.
// The viewer is a static HTML page that connects to the screencast WebSocket.
// Auth is handled by passing the API token as a query parameter to the WebSocket URL.
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
	_, _ = w.Write([]byte(assets.ViewerHTML))
}
