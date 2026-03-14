package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultFileConfig(t *testing.T) {
	fc := DefaultFileConfig()
	if fc.Server.Port != "9867" {
		t.Errorf("DefaultFileConfig.Server.Port = %v, want 9867", fc.Server.Port)
	}
	if fc.Server.Bind != "127.0.0.1" {
		t.Errorf("DefaultFileConfig.Server.Bind = %v, want 127.0.0.1", fc.Server.Bind)
	}
	if fc.InstanceDefaults.Mode != "headless" {
		t.Errorf("DefaultFileConfig.InstanceDefaults.Mode = %v, want headless", fc.InstanceDefaults.Mode)
	}
	if fc.MultiInstance.Strategy != "always-on" {
		t.Errorf("DefaultFileConfig.MultiInstance.Strategy = %v, want always-on", fc.MultiInstance.Strategy)
	}
	if len(fc.Security.Attach.AllowSchemes) != 2 || fc.Security.Attach.AllowSchemes[0] != "ws" || fc.Security.Attach.AllowSchemes[1] != "wss" {
		t.Errorf("DefaultFileConfig.Security.Attach.AllowSchemes = %v, want [ws wss]", fc.Security.Attach.AllowSchemes)
	}
	if fc.Security.AllowEvaluate == nil || *fc.Security.AllowEvaluate {
		t.Errorf("DefaultFileConfig.Security.AllowEvaluate = %v, want explicit false", formatBoolPtr(fc.Security.AllowEvaluate))
	}
	if fc.Security.AllowMacro == nil || *fc.Security.AllowMacro {
		t.Errorf("DefaultFileConfig.Security.AllowMacro = %v, want explicit false", formatBoolPtr(fc.Security.AllowMacro))
	}
	if fc.Security.AllowScreencast == nil || *fc.Security.AllowScreencast {
		t.Errorf("DefaultFileConfig.Security.AllowScreencast = %v, want explicit false", formatBoolPtr(fc.Security.AllowScreencast))
	}
	if fc.Security.AllowDownload == nil || *fc.Security.AllowDownload {
		t.Errorf("DefaultFileConfig.Security.AllowDownload = %v, want explicit false", formatBoolPtr(fc.Security.AllowDownload))
	}
	if fc.Security.AllowUpload == nil || *fc.Security.AllowUpload {
		t.Errorf("DefaultFileConfig.Security.AllowUpload = %v, want explicit false", formatBoolPtr(fc.Security.AllowUpload))
	}
	if !fc.Security.IDPI.Enabled {
		t.Errorf("DefaultFileConfig.Security.IDPI.Enabled = %v, want true", fc.Security.IDPI.Enabled)
	}
	if len(fc.Security.IDPI.AllowedDomains) != 3 || fc.Security.IDPI.AllowedDomains[0] != "127.0.0.1" {
		t.Errorf("DefaultFileConfig.Security.IDPI.AllowedDomains = %v, want local-only allowlist", fc.Security.IDPI.AllowedDomains)
	}
	if !fc.Security.IDPI.StrictMode {
		t.Errorf("DefaultFileConfig.Security.IDPI.StrictMode = %v, want true", fc.Security.IDPI.StrictMode)
	}
	if !fc.Security.IDPI.ScanContent {
		t.Errorf("DefaultFileConfig.Security.IDPI.ScanContent = %v, want true", fc.Security.IDPI.ScanContent)
	}
	if !fc.Security.IDPI.WrapContent {
		t.Errorf("DefaultFileConfig.Security.IDPI.WrapContent = %v, want true", fc.Security.IDPI.WrapContent)
	}
}

// TestIsLegacyConfig tests the format detection logic.
func TestIsLegacyConfig(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		isLegacy bool
	}{
		{
			name:     "nested format with server",
			json:     `{"server": {"port": "9867"}}`,
			isLegacy: false,
		},
		{
			name:     "nested format with instanceDefaults",
			json:     `{"instanceDefaults": {"mode": "headless"}}`,
			isLegacy: false,
		},
		{
			name:     "nested format with security.attach",
			json:     `{"security": {"attach": {"enabled": true}}}`,
			isLegacy: false,
		},
		{
			name:     "legacy format with port",
			json:     `{"port": "9867"}`,
			isLegacy: true,
		},
		{
			name:     "legacy format with headless",
			json:     `{"headless": true}`,
			isLegacy: true,
		},
		{
			name:     "empty object",
			json:     `{}`,
			isLegacy: false,
		},
		{
			name:     "mixed - nested wins",
			json:     `{"server": {"port": "8888"}, "port": "7777"}`,
			isLegacy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLegacyConfig([]byte(tt.json))
			if got != tt.isLegacy {
				t.Errorf("isLegacyConfig(%s) = %v, want %v", tt.json, got, tt.isLegacy)
			}
		})
	}
}

// TestConvertLegacyConfig tests the legacy to nested conversion.
func TestConvertLegacyConfig(t *testing.T) {
	h := false
	maxTabs := 25
	lc := &legacyFileConfig{
		Port:          "7777",
		Headless:      &h,
		MaxTabs:       &maxTabs,
		AllowEvaluate: boolPtr(true),
		TimeoutSec:    45,
		NavigateSec:   90,
	}

	fc := convertLegacyConfig(lc)

	if fc.Server.Port != "7777" {
		t.Errorf("converted Server.Port = %v, want 7777", fc.Server.Port)
	}
	if fc.InstanceDefaults.Mode != "headed" {
		t.Errorf("converted InstanceDefaults.Mode = %v, want headed", fc.InstanceDefaults.Mode)
	}
	if *fc.InstanceDefaults.MaxTabs != 25 {
		t.Errorf("converted InstanceDefaults.MaxTabs = %v, want 25", *fc.InstanceDefaults.MaxTabs)
	}
	if *fc.Security.AllowEvaluate != true {
		t.Errorf("converted Security.AllowEvaluate = %v, want true", *fc.Security.AllowEvaluate)
	}
	if fc.Timeouts.ActionSec != 45 {
		t.Errorf("converted Timeouts.ActionSec = %v, want 45", fc.Timeouts.ActionSec)
	}
	if fc.Timeouts.NavigateSec != 90 {
		t.Errorf("converted Timeouts.NavigateSec = %v, want 90", fc.Timeouts.NavigateSec)
	}
}

// TestDefaultFileConfigJSON tests that DefaultFileConfig serializes correctly.
func TestDefaultFileConfigJSON(t *testing.T) {
	fc := DefaultFileConfig()
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal DefaultFileConfig: %v", err)
	}

	// Verify it can be parsed back
	var parsed FileConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal DefaultFileConfig output: %v", err)
	}

	if parsed.Server.Port != "9867" {
		t.Errorf("round-trip Server.Port = %v, want 9867", parsed.Server.Port)
	}
	if parsed.InstanceDefaults.Mode != "headless" {
		t.Errorf("round-trip InstanceDefaults.Mode = %v, want headless", parsed.InstanceDefaults.Mode)
	}
	if parsed.Security.AllowEvaluate == nil || *parsed.Security.AllowEvaluate {
		t.Errorf("round-trip Security.AllowEvaluate = %v, want explicit false", formatBoolPtr(parsed.Security.AllowEvaluate))
	}
	if parsed.Security.AllowMacro == nil || *parsed.Security.AllowMacro {
		t.Errorf("round-trip Security.AllowMacro = %v, want explicit false", formatBoolPtr(parsed.Security.AllowMacro))
	}
	if parsed.Security.AllowScreencast == nil || *parsed.Security.AllowScreencast {
		t.Errorf("round-trip Security.AllowScreencast = %v, want explicit false", formatBoolPtr(parsed.Security.AllowScreencast))
	}
	if parsed.Security.AllowDownload == nil || *parsed.Security.AllowDownload {
		t.Errorf("round-trip Security.AllowDownload = %v, want explicit false", formatBoolPtr(parsed.Security.AllowDownload))
	}
	if parsed.Security.AllowUpload == nil || *parsed.Security.AllowUpload {
		t.Errorf("round-trip Security.AllowUpload = %v, want explicit false", formatBoolPtr(parsed.Security.AllowUpload))
	}
	if !parsed.Security.IDPI.Enabled {
		t.Errorf("round-trip Security.IDPI.Enabled = %v, want true", parsed.Security.IDPI.Enabled)
	}
	if len(parsed.Security.IDPI.AllowedDomains) != 3 || parsed.Security.IDPI.AllowedDomains[0] != "127.0.0.1" {
		t.Errorf("round-trip Security.IDPI.AllowedDomains = %v, want local-only allowlist", parsed.Security.IDPI.AllowedDomains)
	}
	if !parsed.Security.IDPI.StrictMode {
		t.Errorf("round-trip Security.IDPI.StrictMode = %v, want true", parsed.Security.IDPI.StrictMode)
	}
	if !parsed.Security.IDPI.ScanContent {
		t.Errorf("round-trip Security.IDPI.ScanContent = %v, want true", parsed.Security.IDPI.ScanContent)
	}
	if !parsed.Security.IDPI.WrapContent {
		t.Errorf("round-trip Security.IDPI.WrapContent = %v, want true", parsed.Security.IDPI.WrapContent)
	}
}

func TestFileConfigJSONPreservesExplicitZeroValues(t *testing.T) {
	fc := DefaultFileConfig()
	fc.Server.Bind = ""
	fc.Browser.ExtensionPaths = []string{}
	fc.InstanceDefaults.UserAgent = ""
	fc.Security.IDPI.StrictMode = false
	fc.Security.IDPI.AllowedDomains = []string{}
	fc.Security.IDPI.CustomPatterns = []string{}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("json.Marshal(FileConfig) error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal(FileConfig JSON) error = %v", err)
	}

	server := raw["server"].(map[string]any)
	if bind, ok := server["bind"]; !ok || bind != "" {
		t.Fatalf("server.bind = %#v, want explicit empty string", bind)
	}

	browser := raw["browser"].(map[string]any)
	if ext, ok := browser["extensionPaths"]; !ok {
		t.Fatal("browser.extensionPaths missing from JSON")
	} else if items, ok := ext.([]any); !ok || len(items) != 0 {
		t.Fatalf("browser.extensionPaths = %#v, want explicit empty list", ext)
	}

	security := raw["security"].(map[string]any)
	idpi := security["idpi"].(map[string]any)
	if strictMode, ok := idpi["strictMode"]; !ok || strictMode != false {
		t.Fatalf("security.idpi.strictMode = %#v, want explicit false", strictMode)
	}
	if allowedDomains, ok := idpi["allowedDomains"]; !ok {
		t.Fatal("security.idpi.allowedDomains missing from JSON")
	} else if items, ok := allowedDomains.([]any); !ok || len(items) != 0 {
		t.Fatalf("security.idpi.allowedDomains = %#v, want explicit empty list", allowedDomains)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
