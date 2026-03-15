package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pinchtab/pinchtab/internal/web"
)

// inputEvent represents a mouse/keyboard/scroll event from the viewer client.
type inputEvent struct {
	Type string `json:"type"`

	// Mouse fields
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Button string  `json:"button,omitempty"` // "left", "middle", "right"

	// Keyboard fields
	Key  string `json:"key,omitempty"`  // W3C key value ("a", "Enter", etc.)
	Code string `json:"code,omitempty"` // Physical key code ("KeyA", "Enter", etc.)

	// Scroll fields
	DeltaX float64 `json:"deltaX,omitempty"`
	DeltaY float64 `json:"deltaY,omitempty"`

	// Viewport dimensions from the client for coordinate scaling
	ViewportWidth  float64 `json:"viewportWidth,omitempty"`
	ViewportHeight float64 `json:"viewportHeight,omitempty"`
}

// HandleScreencast upgrades to WebSocket and streams screencast frames for a tab.
// When security.allowRemoteInput is enabled, it also accepts input events from
// the client (mouse, keyboard, scroll) and dispatches them via CDP.
// Query params: tabId (required), quality (1-100, default 40), maxWidth (default 800), fps (1-30, default 5)
func (h *Handlers) HandleScreencast(w http.ResponseWriter, r *http.Request) {
	if !h.Config.AllowScreencast {
		web.ErrorCode(w, 403, "screencast_disabled", web.DisabledEndpointMessage("screencast", "security.allowScreencast"), false, map[string]any{
			"setting": "security.allowScreencast",
		})
		return
	}
	tabID := r.URL.Query().Get("tabId")
	if tabID == "" {
		targets, err := h.Bridge.ListTargets()
		if err == nil && len(targets) > 0 {
			tabID = string(targets[0].TargetID)
		}
	}

	ctx, _, err := h.Bridge.TabContext(tabID)
	if err != nil {
		http.Error(w, "tab not found", 404)
		return
	}

	quality := queryParamInt(r, "quality", 30)
	maxWidth := queryParamInt(r, "maxWidth", 800)
	everyNth := queryParamInt(r, "everyNthFrame", 4)
	fps := queryParamInt(r, "fps", 1)
	if fps > 30 {
		fps = 30
	}
	minFrameInterval := time.Second / time.Duration(fps)

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}
	defer func() { _ = conn.Close() }()

	if ctx == nil {
		return
	}

	frameCh := make(chan []byte, 3)
	var once sync.Once
	done := make(chan struct{})

	// Listen for screencast frames with rate limiting
	var lastFrame time.Time
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *page.EventScreencastFrame:
			go func() {
				_ = chromedp.Run(ctx,
					chromedp.ActionFunc(func(c context.Context) error {
						return page.ScreencastFrameAck(e.SessionID).Do(c)
					}),
				)
			}()

			now := time.Now()
			if now.Sub(lastFrame) < minFrameInterval {
				return
			}
			lastFrame = now

			data, err := base64.StdEncoding.DecodeString(e.Data)
			if err != nil {
				return
			}

			select {
			case frameCh <- data:
			default:
			}
		}
	})

	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(c context.Context) error {
			return page.StartScreencast().
				WithFormat(page.ScreencastFormatJpeg).
				WithQuality(int64(quality)).
				WithMaxWidth(int64(maxWidth)).
				WithMaxHeight(int64(maxWidth * 3 / 4)).
				WithEveryNthFrame(int64(everyNth)).
				Do(c)
		}),
	)
	if err != nil {
		slog.Error("start screencast failed", "err", err, "tab", tabID)
		return
	}

	defer func() {
		once.Do(func() { close(done) })
		_ = chromedp.Run(ctx,
			chromedp.ActionFunc(func(c context.Context) error {
				return page.StopScreencast().Do(c)
			}),
		)
	}()

	allowInput := h.Config.AllowRemoteInput
	slog.Info("screencast started", "tab", tabID, "quality", quality, "maxWidth", maxWidth, "remoteInput", allowInput)

	// Read goroutine: handles client messages (input events or disconnect detection)
	go func() {
		for {
			data, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			// Binary frames are screencast data (server→client only); ignore if received
			if op == ws.OpBinary {
				continue
			}
			// Text frames are input events from the viewer
			if !allowInput || len(data) == 0 {
				continue
			}
			var evt inputEvent
			if err := json.Unmarshal(data, &evt); err != nil {
				slog.Debug("screencast: invalid input event", "err", err)
				continue
			}
			if dispatchErr := dispatchInputEvent(ctx, evt); dispatchErr != nil {
				slog.Debug("screencast: input dispatch failed", "type", evt.Type, "err", dispatchErr)
			}
		}
	}()

	for {
		select {
		case frame := <-frameCh:
			if err := wsutil.WriteServerBinary(conn, frame); err != nil {
				return
			}
		case <-done:
			return
		case <-time.After(10 * time.Second):
			if err := wsutil.WriteServerMessage(conn, ws.OpPing, nil); err != nil {
				return
			}
		}
	}
}

// dispatchInputEvent translates a viewer input event into CDP Input domain calls.
func dispatchInputEvent(ctx context.Context, evt inputEvent) error {
	switch evt.Type {
	case "mousemove":
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			return input.DispatchMouseEvent(input.MouseMoved, evt.X, evt.Y).Do(c)
		}))

	case "mousedown":
		btn := toMouseButton(evt.Button)
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			return input.DispatchMouseEvent(input.MousePressed, evt.X, evt.Y).
				WithButton(btn).
				WithClickCount(1).
				Do(c)
		}))

	case "mouseup":
		btn := toMouseButton(evt.Button)
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			return input.DispatchMouseEvent(input.MouseReleased, evt.X, evt.Y).
				WithButton(btn).
				WithClickCount(1).
				Do(c)
		}))

	case "click":
		btn := toMouseButton(evt.Button)
		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(c context.Context) error {
				return input.DispatchMouseEvent(input.MousePressed, evt.X, evt.Y).
					WithButton(btn).
					WithClickCount(1).
					Do(c)
			}),
			chromedp.ActionFunc(func(c context.Context) error {
				return input.DispatchMouseEvent(input.MouseReleased, evt.X, evt.Y).
					WithButton(btn).
					WithClickCount(1).
					Do(c)
			}),
		)
		return err

	case "keydown":
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			p := input.DispatchKeyEvent(input.KeyRawDown).
				WithKey(evt.Key).
				WithCode(evt.Code)
			if len(evt.Key) == 1 {
				p = p.WithText(evt.Key)
			}
			return p.Do(c)
		}))

	case "keyup":
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			return input.DispatchKeyEvent(input.KeyUp).
				WithKey(evt.Key).
				WithCode(evt.Code).
				Do(c)
		}))

	case "keypress":
		// For printable characters, send the full keydown + char + keyup sequence
		return chromedp.Run(ctx,
			chromedp.ActionFunc(func(c context.Context) error {
				return input.DispatchKeyEvent(input.KeyRawDown).
					WithKey(evt.Key).
					WithCode(evt.Code).
					Do(c)
			}),
			chromedp.ActionFunc(func(c context.Context) error {
				if len(evt.Key) == 1 {
					return input.InsertText(evt.Key).Do(c)
				}
				return nil
			}),
			chromedp.ActionFunc(func(c context.Context) error {
				return input.DispatchKeyEvent(input.KeyUp).
					WithKey(evt.Key).
					WithCode(evt.Code).
					Do(c)
			}),
		)

	case "scroll":
		return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			return input.DispatchMouseEvent(input.MouseWheel, evt.X, evt.Y).
				WithDeltaX(evt.DeltaX).
				WithDeltaY(evt.DeltaY).
				Do(c)
		}))

	default:
		return fmt.Errorf("unknown input event type: %s", evt.Type)
	}
}

func toMouseButton(btn string) input.MouseButton {
	switch btn {
	case "middle":
		return input.Middle
	case "right":
		return input.Right
	default:
		return input.Left
	}
}

// HandleScreencastAll returns info for building a multi-tab screencast view.
func (h *Handlers) HandleScreencastAll(w http.ResponseWriter, r *http.Request) {
	if !h.Config.AllowScreencast {
		web.ErrorCode(w, 403, "screencast_disabled", web.DisabledEndpointMessage("screencast", "security.allowScreencast"), false, map[string]any{
			"setting": "security.allowScreencast",
		})
		return
	}
	type tabInfo struct {
		ID    string `json:"id"`
		URL   string `json:"url,omitempty"`
		Title string `json:"title,omitempty"`
	}

	targets, err := h.Bridge.ListTargets()
	if err != nil {
		web.JSON(w, 200, []tabInfo{})
		return
	}

	tabs := make([]tabInfo, 0)
	for _, t := range targets {
		tabs = append(tabs, tabInfo{
			ID:    string(t.TargetID),
			URL:   t.URL,
			Title: t.Title,
		})
	}

	web.JSON(w, 200, tabs)
}

func queryParamInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}
