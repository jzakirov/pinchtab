package apiclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestResolveProfileBaseUsesRunningInstance(t *testing.T) {
	var authHeader string
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.EscapedPath() != "/profiles/work%20profile/instance" {
			t.Fatalf("path = %s", r.URL.EscapedPath())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    "work profile",
			"running": true,
			"status":  "running",
			"port":    "9911",
			"id":      "inst_123",
		})
	}))
	defer orch.Close()

	got := ResolveProfileBase(orch.URL, "tok123", "work profile", "127.0.0.1")
	if got != "http://127.0.0.1:9911" {
		t.Fatalf("ResolveProfileBase() = %q", got)
	}
	if authHeader != "Bearer tok123" {
		t.Fatalf("Authorization = %q", authHeader)
	}
}

func TestResolveProfileBaseStartsStoppedProfile(t *testing.T) {
	var requests []string
	var startBody string
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/profiles/tg-123/instance":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":    "tg-123",
				"running": false,
				"status":  "stopped",
				"port":    "",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/profiles/tg-123/start":
			body, _ := io.ReadAll(r.Body)
			startBody = string(body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "inst_456",
				"profileName": "tg-123",
				"port":        "9922",
				"status":      "starting",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer orch.Close()

	got := ResolveProfileBase(orch.URL, "", "tg-123", "pinchtab")
	if got != "http://pinchtab:9922" {
		t.Fatalf("ResolveProfileBase() = %q", got)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v", requests)
	}
	if startBody != `{"headless":true}` {
		t.Fatalf("start body = %q", startBody)
	}
}

func TestGetProfileInstanceStatusEscapesProfileName(t *testing.T) {
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/profiles/name%2Fwith%20space/instance" {
			t.Fatalf("path = %s", r.URL.EscapedPath())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    "name/with space",
			"running": true,
			"status":  "running",
			"port":    "9988",
		})
	}))
	defer orch.Close()

	client := orch.Client()
	status, err := getProfileInstanceStatus(client, orch.URL, "", "name/with space")
	if err != nil {
		t.Fatal(err)
	}
	if status.Port != "9988" {
		t.Fatalf("port = %q", status.Port)
	}
}

func TestResolveInstanceBase(t *testing.T) {
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/inst_789" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"port": "9999"})
	}))
	defer orch.Close()

	got := ResolveInstanceBase(orch.URL, "", "inst_789", "localhost")
	if got != "http://localhost:9999" {
		t.Fatalf("ResolveInstanceBase() = %q", got)
	}
}

func TestResolveProfileBaseWorksWithTrimmedURL(t *testing.T) {
	orch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profiles/demo/instance" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    "demo",
			"running": true,
			"status":  "running",
			"port":    "9901",
		})
	}))
	defer orch.Close()

	u, _ := url.Parse(orch.URL)
	got := ResolveProfileBase(u.String(), "", "demo", u.Hostname())
	if got != "http://"+u.Hostname()+":9901" {
		t.Fatalf("ResolveProfileBase() = %q", got)
	}
}
