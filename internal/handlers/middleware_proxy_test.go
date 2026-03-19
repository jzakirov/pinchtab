package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginRequest_UsesForwardedProtoAndHost(t *testing.T) {
	req := httptest.NewRequest("GET", "http://pinchtab/health", nil)
	req.Host = "pinchtab:9867"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "browser.example.com")

	if !sameOriginRequest("https://browser.example.com/dashboard", req) {
		t.Fatal("expected same-origin request when forwarded proto/host match browser origin")
	}
}

func TestSameOriginRequest_UsesForwardedHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://pinchtab/health", nil)
	req.Host = "pinchtab:9867"
	req.Header.Set("Forwarded", `for=127.0.0.1;proto=https;host=browser.example.com`)

	if !sameOriginRequest("https://browser.example.com/dashboard", req) {
		t.Fatal("expected same-origin request when RFC 7239 Forwarded header matches browser origin")
	}
}

