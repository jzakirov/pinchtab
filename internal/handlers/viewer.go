package handlers

import (
	"net/http"

	"github.com/pinchtab/pinchtab/internal/assets"
	"github.com/pinchtab/pinchtab/internal/web"
)

// HandleViewer serves the interactive browser viewer page.
// The viewer connects to the screencast WebSocket and forwards input events.
// Access can be granted via a share token (no API auth needed) or the normal API token.
//
// GET /viewer?token=<share-token>
// GET /viewer?tabId=<tab-id>&apiToken=<api-token>
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

// HandleShareValidate validates a share token without consuming it.
//
// GET /share/validate?token=...
func (h *Handlers) HandleShareValidate(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		web.Error(w, 400, nil)
		return
	}

	tabID, ok := h.ShareTokens.Validate(token)
	if !ok {
		web.ErrorCode(w, 401, "invalid_token", "token is invalid or expired", false, nil)
		return
	}

	web.JSON(w, 200, map[string]any{
		"valid": true,
		"tabId": tabID,
	})
}
