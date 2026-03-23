package orchestrator

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pinchtab/pinchtab/internal/handlers"
)

const kasmVncPort = "6901"

// findInstanceByAuthToken finds a running instance whose bridge auth token matches.
func (o *Orchestrator) findInstanceByAuthToken(token string) *InstanceInternal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, inst := range o.instances {
		if inst.authToken == token && inst.Status == "running" && instanceIsActive(inst) {
			return inst
		}
	}
	return nil
}

// findInstanceByID finds a running instance by its ID.
func (o *Orchestrator) findInstanceByID(id string) *InstanceInternal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	inst, ok := o.instances[id]
	if !ok || inst.Status != "running" || !instanceIsActive(inst) {
		return nil
	}
	return inst
}

// deriveVncURL replaces the port in a bridge URL with the KasmVNC port.
func deriveVncURL(bridgeURL string) string {
	parsed, err := url.Parse(bridgeURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	return "http://" + host + ":" + kasmVncPort
}

// VncProxyMiddleware intercepts requests to the VNC domain (identified by
// X-VNC-Proxy header set by Caddy) and proxies them to the correct
// container's KasmVNC instance.
//
// Auth flow:
//  1. User visits vnc.domain/?token={per-user-bridge-token}
//  2. Middleware finds instance by auth token, sets vnc_route cookie, redirects to /
//  3. Subsequent requests (including WebSocket) carry cookie → proxy to KasmVNC
func VncProxyMiddleware(orch *Orchestrator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-VNC-Proxy") != "true" {
			next.ServeHTTP(w, r)
			return
		}

		// Try token from query param (initial auth)
		if token := r.URL.Query().Get("token"); token != "" {
			inst := orch.findInstanceByAuthToken(token)
			if inst == nil {
				http.Error(w, "invalid token or no running instance", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "vnc_route",
				Value:    inst.ID,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
				SameSite: http.SameSiteLaxMode,
			})
			slog.Info("vnc: authenticated", "instance", inst.ID, "profile", inst.ProfileName)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Try cookie (subsequent requests + WebSocket)
		cookie, err := r.Cookie("vnc_route")
		if err != nil || cookie.Value == "" {
			http.Error(w, "missing token — visit vnc.domain/?token=YOUR_TOKEN", http.StatusUnauthorized)
			return
		}

		inst := orch.findInstanceByID(cookie.Value)
		if inst == nil {
			http.Error(w, "session expired or instance stopped", http.StatusBadGateway)
			return
		}

		vncBase := deriveVncURL(inst.URL)
		if vncBase == "" {
			http.Error(w, "cannot determine VNC endpoint", http.StatusBadGateway)
			return
		}

		// WebSocket upgrade → tunnel via ProxyWebSocket
		if isWebSocketUpgrade(r) {
			target := vncBase + r.URL.Path
			slog.Debug("vnc: ws proxy", "instance", inst.ID, "target", target)
			handlers.ProxyWebSocket(w, r, target)
			return
		}

		// HTTP → reverse proxy
		targetURL, _ := url.Parse(vncBase)
		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.Host = targetURL.Host
			},
		}
		proxy.ServeHTTP(w, r)
	})
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}
