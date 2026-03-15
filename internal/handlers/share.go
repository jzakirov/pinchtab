package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pinchtab/pinchtab/internal/web"
)

// ShareToken represents a time-limited token granting viewer access to a tab.
type ShareToken struct {
	Token     string    `json:"token"`
	TabID     string    `json:"tabId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ShareTokenStore manages share tokens with automatic expiry.
type ShareTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*ShareToken
}

// NewShareTokenStore creates a new token store and starts a background cleanup goroutine.
func NewShareTokenStore() *ShareTokenStore {
	s := &ShareTokenStore{
		tokens: make(map[string]*ShareToken),
	}
	go s.cleanup()
	return s
}

func (s *ShareTokenStore) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, t := range s.tokens {
			if now.After(t.ExpiresAt) {
				delete(s.tokens, k)
			}
		}
		s.mu.Unlock()
	}
}

// Issue creates a new share token for the given tab with the specified TTL.
func (s *ShareTokenStore) Issue(tabID string, ttl time.Duration) (*ShareToken, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := &ShareToken{
		Token:     hex.EncodeToString(b),
		TabID:     tabID,
		ExpiresAt: time.Now().Add(ttl),
	}
	s.mu.Lock()
	s.tokens[token.Token] = token
	s.mu.Unlock()
	return token, nil
}

// Validate checks if a token is valid and returns its associated tab ID.
func (s *ShareTokenStore) Validate(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tokens[token]
	if !ok || time.Now().After(t.ExpiresAt) {
		return "", false
	}
	return t.TabID, true
}

// Revoke removes a token.
func (s *ShareTokenStore) Revoke(token string) {
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}

// HandleShare creates a shareable viewer URL for a tab.
//
// POST /share
// Body: { "tabId": "...", "ttlSeconds": 3600 }
// Response: { "token": "...", "tabId": "...", "viewerUrl": "/viewer?token=...", "expiresAt": "..." }
func (h *Handlers) HandleShare(w http.ResponseWriter, r *http.Request) {
	if !h.Config.AllowShareUrls {
		web.ErrorCode(w, 403, "share_disabled", web.DisabledEndpointMessage("share", "security.allowShareUrls"), false, map[string]any{
			"setting": "security.allowShareUrls",
		})
		return
	}
	if !h.Config.AllowScreencast {
		web.ErrorCode(w, 403, "screencast_disabled", "share requires screencast to be enabled (security.allowScreencast)", false, map[string]any{
			"setting": "security.allowScreencast",
		})
		return
	}

	var req struct {
		TabID      string `json:"tabId"`
		TTLSeconds int    `json:"ttlSeconds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		web.Error(w, 400, fmt.Errorf("decode: %w", err))
		return
	}

	// Default TTL: 1 hour, max: 24 hours
	ttl := time.Hour
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
		if ttl > 24*time.Hour {
			ttl = 24 * time.Hour
		}
	}

	// Resolve tab
	tabID := req.TabID
	if tabID == "" {
		targets, err := h.Bridge.ListTargets()
		if err == nil && len(targets) > 0 {
			tabID = string(targets[0].TargetID)
		}
	}
	if tabID == "" {
		web.Error(w, 404, fmt.Errorf("no tabs available"))
		return
	}

	if _, _, err := h.Bridge.TabContext(tabID); err != nil {
		web.Error(w, 404, fmt.Errorf("tab not found: %w", err))
		return
	}

	token, err := h.ShareTokens.Issue(tabID, ttl)
	if err != nil {
		web.Error(w, 500, fmt.Errorf("failed to issue token: %w", err))
		return
	}

	web.JSON(w, 200, map[string]any{
		"token":     token.Token,
		"tabId":     token.TabID,
		"viewerUrl": fmt.Sprintf("/viewer?token=%s", token.Token),
		"expiresAt": token.ExpiresAt.Format(time.RFC3339),
	})
}

// HandleShareRevoke revokes a share token.
//
// DELETE /share?token=...
func (h *Handlers) HandleShareRevoke(w http.ResponseWriter, r *http.Request) {
	if !h.Config.AllowShareUrls {
		web.ErrorCode(w, 403, "share_disabled", web.DisabledEndpointMessage("share", "security.allowShareUrls"), false, map[string]any{
			"setting": "security.allowShareUrls",
		})
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		web.Error(w, 400, fmt.Errorf("token query parameter required"))
		return
	}

	h.ShareTokens.Revoke(token)
	web.JSON(w, 200, map[string]any{"revoked": true})
}
