package orchestrator

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pinchtab/pinchtab/internal/dashboard"
	"github.com/pinchtab/pinchtab/internal/handlers"
	"github.com/pinchtab/pinchtab/internal/web"
)

// presignedPayload is the data encoded into a presigned live share URL.
type presignedPayload struct {
	InstanceID string
	ExpiresAt  time.Time
}

// signPayload creates an HMAC-SHA256 signature for a presigned URL.
// Format: {instanceId}:{expiresUnix}:{signature}
func (o *Orchestrator) signPayload(instanceID string, expiresAt time.Time) (string, error) {
	secret, err := o.presignSecret()
	if err != nil {
		return "", err
	}
	expStr := strconv.FormatInt(expiresAt.Unix(), 10)
	msg := instanceID + ":" + expStr
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	return instanceID + ":" + expStr + ":" + sig, nil
}

// verifyPayload verifies and extracts data from a presigned token.
func (o *Orchestrator) verifyPayload(token string) (*presignedPayload, error) {
	secret, err := o.presignSecret()
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(token, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}
	instanceID, expStr, sig := parts[0], parts[1], parts[2]

	// Verify signature
	msg := instanceID + ":" + expStr
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Check expiry
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid expiry")
	}
	expiresAt := time.Unix(expUnix, 0)
	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return &presignedPayload{
		InstanceID: instanceID,
		ExpiresAt:  expiresAt,
	}, nil
}

// presignSecret returns the HMAC key for signing presigned URLs.
// Uses the orchestrator's configured token — if no token is set,
// presigned URLs cannot be generated.
func (o *Orchestrator) presignSecret() (string, error) {
	if o.runtimeCfg != nil && o.runtimeCfg.Token != "" {
		return "pinchtab-presign:" + o.runtimeCfg.Token, nil
	}
	return "", fmt.Errorf("presigned links require a configured API token")
}

// handleCreateShareLink generates a presigned live share URL for an instance.
//
// POST /instances/{id}/share
// Body: { "ttlSeconds": 3600 }  (optional, default 1h, max 24h)
// Response: { "url": "/live/{token}", "expiresAt": "..." }
func (o *Orchestrator) handleCreateShareLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	o.mu.RLock()
	inst, ok := o.instances[id]
	o.mu.RUnlock()

	if !ok {
		web.Error(w, 404, fmt.Errorf("instance %q not found", id))
		return
	}
	if inst.Status != "running" {
		web.Error(w, 503, fmt.Errorf("instance %q is not running", id))
		return
	}

	// Parse optional TTL
	ttl := time.Hour
	if r.Body != nil {
		var req struct {
			TTLSeconds int `json:"ttlSeconds"`
		}
		// Ignore decode errors — use default TTL
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TTLSeconds > 0 {
			ttl = time.Duration(req.TTLSeconds) * time.Second
			if ttl > 24*time.Hour {
				ttl = 24 * time.Hour
			}
		}
	}

	expiresAt := time.Now().Add(ttl)
	token, err := o.signPayload(id, expiresAt)
	if err != nil {
		web.Error(w, 503, err)
		return
	}

	web.JSON(w, 200, map[string]any{
		"url":        "/live/" + token,
		"expiresAt":  expiresAt.Format(time.RFC3339),
		"instanceId": id,
	})
}

// handleLiveViewer serves the React SPA for a presigned URL.
// The SPA's /live/:token route renders the fullscreen browser viewer.
// No API auth required — the presigned token in the path IS the auth.
//
// GET /live/{token}
func (o *Orchestrator) handleLiveViewer(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if _, err := o.verifyPayload(token); err != nil {
		web.ErrorCode(w, 401, "invalid_link", "This link is invalid or has expired.", false, nil)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if html := dashboard.SPAHTML(); html != nil {
		_, _ = w.Write(html)
	} else {
		// Fallback: minimal redirect to indicate dashboard not built
		http.Error(w, "Dashboard not built. Run the dashboard build first.", 503)
	}
}

// handleLiveScreencast proxies the screencast WebSocket for a presigned URL.
// No API auth required — the presigned token in the path IS the auth.
//
// GET /live/{token}/screencast
func (o *Orchestrator) handleLiveScreencast(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	payload, err := o.verifyPayload(token)
	if err != nil {
		web.ErrorCode(w, 401, "invalid_link", "This link is invalid or has expired.", false, nil)
		return
	}

	o.mu.RLock()
	inst, ok := o.instances[payload.InstanceID]
	o.mu.RUnlock()
	if !ok || inst.Status != "running" {
		web.Error(w, 404, fmt.Errorf("instance not found or not running"))
		return
	}

	targetURL, err := o.instancePathURL(inst, "/screencast", r.URL.RawQuery)
	if err != nil {
		web.Error(w, 502, err)
		return
	}

	req := r.Clone(r.Context())
	req.Header = r.Header.Clone()
	o.applyInstanceAuth(req, inst)

	handlers.ProxyWebSocket(w, req, targetURL.String())
}
