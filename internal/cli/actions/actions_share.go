package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Share generates a presigned live share URL for the given instance.
func Share(client *http.Client, base, token, instanceID string, ttlSeconds int) {
	orchestratorURL := os.Getenv("PINCHTAB_ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = base
	}
	orchestratorURL = strings.TrimRight(orchestratorURL, "/")

	body, _ := json.Marshal(map[string]any{"ttlSeconds": ttlSeconds})
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/instances/%s/share", orchestratorURL, instanceID), bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: share request failed: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || result.URL == "" {
		fmt.Fprintf(os.Stderr, "Error: failed to parse share response: %s\n", string(respBody))
		os.Exit(1)
	}

	domain := os.Getenv("DOMAIN")
	if domain != "" {
		fmt.Printf("https://browser.%s%s\n", domain, result.URL)
	} else {
		fmt.Printf("%s%s\n", orchestratorURL, result.URL)
	}
}
