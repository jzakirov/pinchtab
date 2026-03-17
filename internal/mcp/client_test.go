package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:9867", "tok123")
	if c.BaseURL != "http://localhost:9867" {
		t.Fatalf("BaseURL = %q, want %q", c.BaseURL, "http://localhost:9867")
	}
	if c.Token != "tok123" {
		t.Fatalf("Token = %q, want %q", c.Token, "tok123")
	}
	if c.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}
}

func TestNewClientReadsProfileIDFromEnv(t *testing.T) {
	t.Setenv("PINCHTAB_PROFILE_ID", "prof_123")
	c := NewClient("http://localhost:9867", "tok123")
	if c.Profile != "prof_123" {
		t.Fatalf("Profile = %q, want %q", c.Profile, "prof_123")
	}
}

func TestNewClientIgnoresLegacyProfileEnv(t *testing.T) {
	t.Setenv("PINCHTAB_PROFILE", "name-profile")
	c := NewClient("http://localhost:9867", "tok123")
	if c.Profile != "" {
		t.Fatalf("Profile = %q, want empty", c.Profile)
	}
}

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			t.Errorf("no auth header")
		}
		if r.URL.Query().Get("tabId") != "t1" {
			t.Errorf("tabId = %q, want t1", r.URL.Query().Get("tabId"))
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "testtoken")
	body, code, err := c.Get(context.Background(), "/health", url.Values{"tabId": {"t1"}})
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 {
		t.Fatalf("code = %d, want 200", code)
	}
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("body = %q", body)
	}
}

func TestClientGetNoQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, code, err := c.Get(context.Background(), "/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 {
		t.Fatalf("code = %d", code)
	}
}

func TestClientPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"url"`) {
			t.Errorf("body missing url field: %s", body)
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"navigated":true}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	body, code, err := c.Post(context.Background(), "/navigate", map[string]any{"url": "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(string(body), "navigated") {
		t.Fatalf("body = %q", body)
	}
}

func TestClientPostNilPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %s", body)
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, code, err := c.Post(context.Background(), "/shutdown", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 {
		t.Fatalf("code = %d", code)
	}
}

func TestClientAuthHeaderAbsentWhenNoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h := r.Header.Get("Authorization"); h != "" {
			t.Errorf("unexpected Authorization header: %s", h)
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, _, err := c.Get(context.Background(), "/health", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientProfileInstancePath(t *testing.T) {
	c := NewClient("http://localhost:9867", "")
	got := c.profileInstancePath("work profile")
	want := "/profiles/work%20profile/instance"
	if got != want {
		t.Fatalf("profileInstancePath = %q, want %q", got, want)
	}
}

func TestClientDashboardProfilesURL(t *testing.T) {
	c := NewClient("http://localhost:9867/", "")
	got := c.dashboardProfilesURL()
	want := "http://localhost:9867/dashboard/profiles"
	if got != want {
		t.Fatalf("dashboardProfilesURL = %q, want %q", got, want)
	}
}

func TestClientBrowserGetResolvesProfileInstance(t *testing.T) {
	t.Setenv("PINCHTAB_PROFILE_ID", "tg-555")

	var browserAuth string
	instance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		browserAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/snapshot" {
			t.Fatalf("browser path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("tabId") != "tab-1" {
			t.Fatalf("tabId = %q", r.URL.Query().Get("tabId"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer instance.Close()

	instanceURL, err := url.Parse(instance.URL)
	if err != nil {
		t.Fatal(err)
	}

	var orchestrationCalls []string
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orchestrationCalls = append(orchestrationCalls, r.Method+" "+r.URL.Path)
		if r.URL.Path != "/profiles/tg-555/instance" {
			t.Fatalf("orchestrator path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"name":"tg-555","running":true,"status":"running","port":"`+instanceURL.Port()+`","id":"inst_555"}`)
	}))
	defer orch.Close()

	c := NewClient(orch.URL, "tok123")
	body, code, err := c.BrowserGet(context.Background(), "/snapshot", url.Values{"tabId": {"tab-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q", body)
	}
	if browserAuth != "Bearer tok123" {
		t.Fatalf("browser Authorization = %q", browserAuth)
	}
	if len(orchestrationCalls) != 1 {
		t.Fatalf("orchestrationCalls = %v", orchestrationCalls)
	}
}
