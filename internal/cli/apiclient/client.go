package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pinchtab/pinchtab/internal/activity"
)

const profileInstanceReadyTimeout = 20 * time.Second

func DoGet(client *http.Client, base, token, path string, params url.Values) map[string]any {
	u := base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, _ := http.NewRequest("GET", u, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set(activity.HeaderAgentID, "cli")
	resp, err := client.Do(req)
	if err != nil {
		fatal("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Pretty-print JSON if possible
	var buf bytes.Buffer
	if json.Indent(&buf, body, "", "  ") == nil {
		fmt.Println(buf.String())
	} else {
		fmt.Println(string(body))
	}

	// Parse and return result
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("warning: error unmarshaling response: %v", err)
	}
	return result
}

func DoGetRaw(client *http.Client, base, token, path string, params url.Values) []byte {
	u := base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, _ := http.NewRequest("GET", u, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set(activity.HeaderAgentID, "cli")
	resp, err := client.Do(req)
	if err != nil {
		fatal("Request failed: %v", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
	return body
}

func DoPost(client *http.Client, base, token, path string, body map[string]any) map[string]any {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", base+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set(activity.HeaderAgentID, "cli")
	resp, err := client.Do(req)
	if err != nil {
		fatal("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var buf bytes.Buffer
	if json.Indent(&buf, respBody, "", "  ") == nil {
		fmt.Println(buf.String())
	} else {
		fmt.Println(string(respBody))
	}

	// Parse and return result for suggestions
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("warning: error unmarshaling response: %v", err)
	}
	return result
}

// ResolveInstanceBase fetches the named instance from the orchestrator and returns
// a base URL pointing directly at that instance's API port.
func ResolveInstanceBase(orchBase, token, instanceID, bind string) string {
	c := &http.Client{Timeout: 10 * time.Second}
	body := DoGetRaw(c, orchBase, token, fmt.Sprintf("/instances/%s", instanceID), nil)

	var inst struct {
		Port string `json:"port"`
	}
	if err := json.Unmarshal(body, &inst); err != nil {
		fatal("failed to parse instance %q: %v", instanceID, err)
	}
	if inst.Port == "" {
		fatal("instance %q has no port assigned (is it still starting?)", instanceID)
	}
	return fmt.Sprintf("http://%s:%s", bind, inst.Port)
}

func ResolveProfileBase(orchBase, token, profile, bind string) string {
	c := &http.Client{Timeout: 15 * time.Second}

	status, err := getProfileInstanceStatus(c, orchBase, token, profile)
	if err != nil {
		fatal("failed to resolve profile %q: %v", profile, err)
	}
	if status.Port != "" && (status.Running || status.Status == "starting") {
		base := fmt.Sprintf("http://%s:%s", bind, status.Port)
		if err := waitForInstanceReady(c, base, token, profileInstanceReadyTimeout); err != nil {
			fatal("profile %q instance not ready: %v", profile, err)
		}
		return base
	}

	started, err := startProfileInstance(c, orchBase, token, profile)
	if err != nil {
		fatal("failed to start profile %q: %v", profile, err)
	}
	if started.Port == "" {
		fatal("profile %q started without a port", profile)
	}
	base := fmt.Sprintf("http://%s:%s", bind, started.Port)
	if err := waitForInstanceReady(c, base, token, profileInstanceReadyTimeout); err != nil {
		fatal("profile %q instance not ready after start: %v", profile, err)
	}
	return base
}

type profileInstanceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
	Port    string `json:"port"`
	ID      string `json:"id"`
}

type startedProfileInstance struct {
	ID          string `json:"id"`
	ProfileName string `json:"profileName"`
	Port        string `json:"port"`
	Status      string `json:"status"`
}

func getProfileInstanceStatus(client *http.Client, orchBase, token, profile string) (*profileInstanceStatus, error) {
	req, err := http.NewRequest(http.MethodGet, orchBase+"/profiles/"+url.PathEscape(profile)+"/instance", nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var status profileInstanceStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("parse profile instance status: %w", err)
	}
	return &status, nil
}

func startProfileInstance(client *http.Client, orchBase, token, profile string) (*startedProfileInstance, error) {
	payload := []byte(`{"headless":true}`)
	req, err := http.NewRequest(http.MethodPost, orchBase+"/profiles/"+url.PathEscape(profile)+"/start", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var started startedProfileInstance
	if err := json.Unmarshal(body, &started); err != nil {
		return nil, fmt.Errorf("parse started profile instance: %w", err)
	}
	return &started, nil
}

func waitForInstanceReady(client *http.Client, base, token string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, base+"/health", nil)
		if err != nil {
			return err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		time.Sleep(250 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out after %s", timeout)
	}
	return lastErr
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
