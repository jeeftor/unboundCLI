package api

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jeeftor/unboundCLI/internal/logging"
)

// Config represents the API configuration
type Config struct {
	APIKey    string `json:"api_key"    mapstructure:"api_key"`
	APISecret string `json:"api_secret" mapstructure:"api_secret"`
	BaseURL   string `json:"base_url"   mapstructure:"base_url"`
	Insecure  bool   `json:"insecure"   mapstructure:"insecure"`
}

// DNSOverride represents a single DNS override entry
type DNSOverride struct {
	UUID        string `json:"uuid,omitempty"`
	Enabled     string `json:"enabled"`
	Host        string `json:"hostname"`
	Domain      string `json:"domain"`
	RR          string `json:"rr,omitempty"`
	MXPrio      string `json:"mxprio,omitempty"`
	MX          string `json:"mx,omitempty"`
	Server      string `json:"server"`
	Description string `json:"description"`
}

// APIResponse represents the response from the OPNSense API
type APIResponse struct {
	Status   string          `json:"status,omitempty"`
	Result   string          `json:"result,omitempty"`
	Message  string          `json:"message,omitempty"`
	UUID     string          `json:"uuid,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Rows     json.RawMessage `json:"rows,omitempty"`
	RowCount int             `json:"rowCount,omitempty"`
	Total    int             `json:"total,omitempty"`
	Current  int             `json:"current,omitempty"`
}

// Client handles API communication with OPNSense
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new OPNSense API client
func NewClient(config Config) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Insecure},
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
			// Enable following redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Preserve the original Authorization header when following redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				for k, v := range via[0].Header {
					if k != "Authorization" && k != "Content-Type" {
						continue
					}
					req.Header[k] = v
				}
				return nil
			},
		},
	}
}

// generateCurlCommand creates a curl command equivalent to the HTTP request
func generateCurlCommand(method, url string, headers map[string]string, body io.Reader) string {
	// Start building the curl command
	cmd := fmt.Sprintf("curl -X %s", method)

	// Add insecure flag if needed (for https URLs)
	if strings.HasPrefix(url, "https") {
		cmd += " -k"
	}

	// Add follow redirects flag
	cmd += " -L"

	// Add headers
	for k, v := range headers {
		cmd += fmt.Sprintf(" -H '%s: %s'", k, v)
	}

	// Add request body if present
	if body != nil {
		var bodyBytes []byte
		if bodyBuffer, ok := body.(*bytes.Buffer); ok {
			bodyBytes = bodyBuffer.Bytes()
		}

		if len(bodyBytes) > 0 {
			cmd += fmt.Sprintf(" -d '%s'", string(bodyBytes))
		}
	}

	// Add URL
	cmd += fmt.Sprintf(" '%s'", url)

	return cmd
}

// makeRequest handles the HTTP request to OPNSense API
func (c *Client) makeRequest(method, endpoint string, body io.Reader) (*APIResponse, error) {
	// Ensure the base URL uses HTTPS
	baseURL := c.config.BaseURL
	if strings.HasPrefix(baseURL, "http://") {
		baseURL = "https://" + strings.TrimPrefix(baseURL, "http://")
		logging.Debug("Converted base URL to HTTPS", "original", c.config.BaseURL, "new", baseURL)
	}

	url := baseURL + endpoint

	// Log the request details at debug level
	logging.Debug("Making API request",
		"method", method,
		"url", url,
		"endpoint", endpoint,
	)

	// Create the request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logging.Error("Failed to create request", "error", err)
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set basic auth header
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.APIKey + ":" + c.config.APISecret))
	req.Header.Set("Authorization", "Basic "+auth)

	// Only set Content-Type for POST requests or when body is not nil
	headers := map[string]string{
		"Authorization": "Basic " + auth,
	}

	if method == "POST" || body != nil {
		req.Header.Set("Content-Type", "application/json")
		headers["Content-Type"] = "application/json"
	}

	// Generate equivalent curl command for debugging
	curlCmd := generateCurlCommand(method, url, headers, body)

	// Log headers, auth details, and curl command at debug level
	logging.Debug("Request headers",
		"auth_header", "Basic "+auth,
		"api_key", c.config.APIKey,
	)
	logging.Debug("Equivalent curl command", "curl", curlCmd)

	// Make the request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		logging.Error("Request failed",
			"error", err,
			"duration_ms", requestDuration.Milliseconds(),
		)
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Error("Failed to read response body", "error", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Log response details
	logging.Debug("Received API response",
		"status_code", resp.StatusCode,
		"duration_ms", requestDuration.Milliseconds(),
		"response_body", string(responseData),
		"final_url", resp.Request.URL.String(),
	)

	// Check if we were redirected to the login page
	if strings.Contains(resp.Request.URL.String(), "/?url=") ||
		strings.Contains(string(responseData), "<title>Login") {
		logging.Error("Redirected to login page - authentication failed",
			"final_url", resp.Request.URL.String(),
		)
		return nil, fmt.Errorf("authentication failed: redirected to login page")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logging.Error("API returned error status",
			"status_code", resp.StatusCode,
			"url", url,
			"response_body", string(responseData),
		)
		return nil, fmt.Errorf(
			"API returned error status: %d - %s",
			resp.StatusCode,
			string(responseData),
		)
	}

	// Parse the response
	var apiResp APIResponse
	if err := json.Unmarshal(responseData, &apiResp); err != nil {
		logging.Error("Failed to unmarshal response",
			"error", err,
			"response_body", string(responseData),
		)
		return nil, fmt.Errorf(
			"error unmarshaling response: %w - Response: %s",
			err,
			string(responseData),
		)
	}

	return &apiResp, nil
}

// GetOverrides retrieves all DNS overrides
func (c *Client) GetOverrides() ([]DNSOverride, error) {
	logging.Debug("Fetching DNS overrides")

	resp, err := c.makeRequest("GET", "/api/unbound/settings/searchHostOverride", nil)
	if err != nil {
		return nil, err
	}

	// Check if we have rows in the response (new API format)
	if len(resp.Rows) > 0 {
		var rows []DNSOverride
		if err := json.Unmarshal(resp.Rows, &rows); err != nil {
			logging.Error("Failed to parse rows",
				"error", err,
				"data", string(resp.Rows),
			)
			return nil, fmt.Errorf("error parsing rows: %w - Data: %s", err, string(resp.Rows))
		}
		logging.Debug("Successfully fetched DNS overrides", "count", len(rows))
		return rows, nil
	}

	// Fall back to old format if rows is empty
	var result map[string]DNSOverride
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		logging.Error("Failed to parse overrides",
			"error", err,
			"data", string(resp.Data),
		)
		return nil, fmt.Errorf("error parsing overrides: %w - Data: %s", err, string(resp.Data))
	}

	overrides := make([]DNSOverride, 0, len(result))
	for uuid, override := range result {
		override.UUID = uuid
		overrides = append(overrides, override)
	}

	logging.Debug("Successfully fetched DNS overrides", "count", len(overrides))
	return overrides, nil
}

// IsOverrideExists checks if a DNS override with the same host and domain already exists
func (c *Client) IsOverrideExists(host, domain string) (bool, string, error) {
	overrides, err := c.GetOverrides()
	if err != nil {
		return false, "", fmt.Errorf("error checking existing overrides: %w", err)
	}

	for _, override := range overrides {
		if strings.EqualFold(override.Host, host) && strings.EqualFold(override.Domain, domain) {
			return true, override.UUID, nil
		}
	}

	return false, "", nil
}

// AddOverride creates a new DNS override
func (c *Client) AddOverride(override DNSOverride) (string, error) {
	// Check if override already exists
	exists, uuid, err := c.IsOverrideExists(override.Host, override.Domain)
	if err != nil {
		return "", err
	}

	if exists {
		logging.Info(
			"DNS override already exists",
			"host",
			override.Host,
			"domain",
			override.Domain,
			"uuid",
			uuid,
		)
		return uuid, fmt.Errorf(
			"DNS override for %s.%s already exists with UUID %s",
			override.Host,
			override.Domain,
			uuid,
		)
	}

	// Prepare the data for the API request
	jsonData, err := json.Marshal(map[string]DNSOverride{
		"host": override,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling override: %w", err)
	}

	resp, err := c.makeRequest(
		"POST",
		"/api/unbound/settings/addHostOverride",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}

	// Check result field instead of status
	if resp.Result != "saved" {
		logging.Error("API returned error", "result", resp.Result, "message", resp.Message)
		return "", fmt.Errorf("API returned error: %s", resp.Result)
	}

	//// Extract UUID from the response
	//var uuid string

	// Try to get UUID directly from response
	if resp.UUID != "" {
		uuid = resp.UUID
	} else {
		// If not available directly, try to extract from Data field
		var result struct {
			UUID string `json:"uuid"`
		}
		if len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, &result); err != nil {
				logging.Error(
					"Failed to parse UUID from response",
					"error",
					err,
					"data",
					string(resp.Data),
				)
				return "", fmt.Errorf("failed to parse UUID from response: %w", err)
			}
			uuid = result.UUID
		}
	}

	if uuid == "" {
		logging.Error("No UUID returned from API", "response", resp)
		return "", fmt.Errorf("no UUID returned from API")
	}

	logging.Info("Successfully added DNS override", "uuid", uuid)
	return uuid, nil
}

// UpdateOverride updates an existing DNS override
func (c *Client) UpdateOverride(override DNSOverride) error {
	if override.UUID == "" {
		logging.Error("UUID is required for update")
		return fmt.Errorf("UUID is required for update")
	}

	logging.Info("Updating DNS override",
		"uuid", override.UUID,
		"host", override.Host,
		"domain", override.Domain,
		"server", override.Server,
	)

	jsonData, err := json.Marshal(override)
	if err != nil {
		logging.Error("Failed to marshal override", "error", err)
		return fmt.Errorf("error marshaling override: %w", err)
	}

	resp, err := c.makeRequest(
		"POST",
		"/api/unbound/settings/setHostOverride/"+override.UUID,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}

	// Check the result field instead of status
	if resp.Result != "saved" && resp.Status != "ok" {
		logging.Error(
			"API returned error",
			"result",
			resp.Result,
			"status",
			resp.Status,
			"message",
			resp.Message,
		)
		return fmt.Errorf("API returned error: %s", resp.Message)
	}

	logging.Info("Successfully updated DNS override", "uuid", override.UUID)
	return nil
}

// DeleteOverride removes a DNS override
func (c *Client) DeleteOverride(uuid string) error {
	logging.Info("Deleting DNS override", "uuid", uuid)

	resp, err := c.makeRequest("POST", "/api/unbound/settings/delHostOverride/"+uuid, nil)
	if err != nil {
		return err
	}

	// Check both result and status fields
	if resp.Result != "deleted" && resp.Status != "ok" {
		logging.Error(
			"API returned error",
			"result",
			resp.Result,
			"status",
			resp.Status,
			"message",
			resp.Message,
		)
		return fmt.Errorf("API returned error: %s", resp.Message)
	}

	logging.Info("Successfully deleted DNS override", "uuid", uuid)
	return nil
}

// ApplyChanges applies all DNS changes to the Unbound service
func (c *Client) ApplyChanges() error {
	logging.Info("Applying changes to Unbound service")

	// Create an empty JSON object as the request body
	emptyJSON := bytes.NewBufferString("{}")
	resp, err := c.makeRequest("POST", "/api/unbound/service/reconfigure", emptyJSON)
	if err != nil {
		return err
	}

	// Check both result and status fields
	if resp.Result != "saved" && resp.Status != "ok" {
		logging.Error(
			"API returned error",
			"result",
			resp.Result,
			"status",
			resp.Status,
			"message",
			resp.Message,
		)
		return fmt.Errorf("API returned error: %s", resp.Message)
	}

	logging.Info("Successfully applied changes to Unbound service")
	return nil
}
