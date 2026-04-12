// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrQGANotRunning is returned when the QEMU Guest Agent is not running.
var ErrQGANotRunning = errors.New("QEMU guest agent is not running")

// PveConnection holds connection details for the Proxmox REST API.
type PveConnection struct {
	id     uint32
	Host   string
	Token  string
	client *http.Client
}

// ID and ParentID implement the plugin.Connection interface.
func (c *PveConnection) ID() uint32       { return c.id }
func (c *PveConnection) ParentID() uint32 { return 0 }

func NewConnection(id uint32, host, token string, insecure bool) *PveConnection {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &PveConnection{
		id:     id,
		Host:   strings.TrimRight(host, "/"),
		Token:  strings.ReplaceAll(token, `\!`, "!"),
		client: &http.Client{Transport: tr, Timeout: 30 * time.Second},
	}
}

// Verify checks whether the connection to the Proxmox API is working.
func (c *PveConnection) Verify() error {
	var result any
	return c.apiGet("/version", &result)
}

// GetVersion returns the Proxmox version information as a map.
func (c *PveConnection) GetVersion() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.apiGet("/version", &result); err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// apiGet performs a GET request against the Proxmox REST API.
func (c *PveConnection) apiGet(path string, result any) error {
	fullURL := fmt.Sprintf("%s/api2/json%s", c.Host, path)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", c.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("proxmox API unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Proxmox wraps all responses in {"data": ...}
	// QGA errors return {"data": null, "message": "QEMU guest agent is not running"}
	var wrapper struct {
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("proxmox API error %d for %s", resp.StatusCode, path)
		}
		return fmt.Errorf("JSON parsing failed: %w", err)
	}

	// Check for QGA-specific error messages (returned on 500 or 596)
	if strings.Contains(wrapper.Message, "guest agent is not running") ||
		strings.Contains(wrapper.Message, "QEMU guest agent is not running") {
		return ErrQGANotRunning
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("proxmox API error %d: %s", resp.StatusCode, wrapper.Message)
	}

	return json.Unmarshal(wrapper.Data, result)
}

// apiPostForm performs a POST request with url.Values.
func (c *PveConnection) apiPostForm(path string, form url.Values, result any) error {
	fullURL := fmt.Sprintf("%s/api2/json%s", c.Host, path)

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST response: %w", err)
	}

	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("POST JSON parsing failed: %w", err)
	}

	if result != nil {
		return json.Unmarshal(wrapper.Data, result)
	}
	return nil
}
