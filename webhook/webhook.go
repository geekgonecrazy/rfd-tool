package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

// EventType represents the type of webhook event
type EventType string

const (
	EventRFDCreated EventType = "rfd.created"
	EventRFDUpdated EventType = "rfd.updated"
)

// Config holds webhook configuration
type Config struct {
	URL    string `yaml:"url" json:"url"`
	Secret string `yaml:"secret" json:"secret"`
}

// Payload is the webhook payload sent to the configured URL
type Payload struct {
	Event          EventType   `json:"event"`
	Timestamp      time.Time   `json:"timestamp"`
	RFD            *models.RFD `json:"rfd"`
	Link           string      `json:"link"`
	Changes        *RFDChanges `json:"changes,omitempty"`
	SkipDiscussion bool        `json:"skip_discussion,omitempty"`
}

// Response is the expected response from the webhook endpoint
type Response struct {
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
	Discussion *DiscussionInfo   `json:"discussion,omitempty"`
}

// DiscussionInfo contains information about the created discussion
type DiscussionInfo struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// RFDChanges tracks what changed in an update
type RFDChanges struct {
	Title      *FieldChange `json:"title,omitempty"`
	State      *FieldChange `json:"state,omitempty"`
	Authors    *FieldChange `json:"authors,omitempty"`
	Tags       *FieldChange `json:"tags,omitempty"`
	Discussion *FieldChange `json:"discussion,omitempty"`
	Content    bool         `json:"content,omitempty"`
}

// FieldChange represents a changed field with old and new values
type FieldChange struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// Client handles sending webhooks
type Client struct {
	config     *Config
	httpClient *http.Client
	siteURL    string
}

// NewClient creates a new webhook client
func NewClient(cfg *Config, siteURL string) *Client {
	if cfg == nil || cfg.URL == "" {
		return nil
	}
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		siteURL: siteURL,
	}
}

// IsConfigured returns true if the webhook client is properly configured
func (c *Client) IsConfigured() bool {
	return c != nil && c.config != nil && c.config.URL != ""
}

// SendCreated sends a webhook for a newly created RFD and returns the discussion URL if provided
func (c *Client) SendCreated(rfd *models.RFD) (*Response, error) {
	if !c.IsConfigured() {
		return nil, nil
	}

	payload := &Payload{
		Event:     EventRFDCreated,
		Timestamp: time.Now().UTC(),
		RFD:       rfd,
		Link:      fmt.Sprintf("%s/rfd/%s", c.siteURL, rfd.ID),
	}

	// For created events, we wait synchronously to get the discussion URL
	return c.sendSync(payload)
}

// SendUpdated sends a webhook for an updated RFD if there are changes
// Returns the response in case a discussion was created
func (c *Client) SendUpdated(old, new *models.RFD) (*Response, error) {
	if !c.IsConfigured() {
		return nil, nil
	}

	changes := detectChanges(old, new)
	if changes == nil {
		// No changes, don't send webhook
		return nil, nil
	}

	payload := &Payload{
		Event:     EventRFDUpdated,
		Timestamp: time.Now().UTC(),
		RFD:       new,
		Link:      fmt.Sprintf("%s/rfd/%s", c.siteURL, new.ID),
		Changes:   changes,
	}

	// Send sync to get response (may contain discussion URL if one was created)
	return c.sendSync(payload)
}

// detectChanges compares old and new RFD and returns changes, or nil if no changes
func detectChanges(old, new *models.RFD) *RFDChanges {
	changes := &RFDChanges{}
	hasChanges := false

	if old.Title != new.Title {
		changes.Title = &FieldChange{Old: old.Title, New: new.Title}
		hasChanges = true
	}

	if old.State != new.State {
		changes.State = &FieldChange{Old: old.State, New: new.State}
		hasChanges = true
	}

	if old.Discussion != new.Discussion {
		changes.Discussion = &FieldChange{Old: old.Discussion, New: new.Discussion}
		hasChanges = true
	}

	if !stringSliceEqual(old.Authors, new.Authors) {
		changes.Authors = &FieldChange{Old: old.Authors, New: new.Authors}
		hasChanges = true
	}

	if !stringSliceEqual(old.Tags, new.Tags) {
		changes.Tags = &FieldChange{Old: old.Tags, New: new.Tags}
		hasChanges = true
	}

	if old.ContentMD != new.ContentMD {
		changes.Content = true
		hasChanges = true
	}

	if !hasChanges {
		return nil
	}

	return changes
}

// stringSliceEqual compares two string slices
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	return reflect.DeepEqual(a, b)
}

// sendSync delivers the webhook payload and waits for the response
func (c *Client) sendSync(payload *Payload) (*Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RFD-Tool-Webhook/1.0")

	// Add HMAC signature if secret is configured
	if c.config.Secret != "" {
		signature := computeHMAC(body, c.config.Secret)
		req.Header.Set("X-RFD-Signature", "sha256="+signature)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook delivery failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var webhookResp Response
	if err := json.Unmarshal(respBody, &webhookResp); err != nil {
		// Response might not be JSON, that's okay
		log.Printf("Webhook response was not JSON: %s", string(respBody))
		return &Response{Success: true}, nil
	}

	return &webhookResp, nil
}

// sendAsync delivers the webhook payload without waiting for response
func (c *Client) sendAsync(payload *Payload) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", c.config.URL, bytes.NewReader(body))
	if err != nil {
		log.Printf("Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RFD-Tool-Webhook/1.0")

	// Add HMAC signature if secret is configured
	if c.config.Secret != "" {
		signature := computeHMAC(body, c.config.Secret)
		req.Header.Set("X-RFD-Signature", "sha256="+signature)
	}

	// Send async to not block the main request
	go func() {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("Webhook delivery failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Printf("Webhook delivery returned status %d", resp.StatusCode)
		}
	}()
}

// computeHMAC creates an HMAC-SHA256 signature
func computeHMAC(message []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies an incoming webhook signature (useful for testing)
func VerifySignature(body []byte, signature, secret string) bool {
	expected := "sha256=" + computeHMAC(body, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
