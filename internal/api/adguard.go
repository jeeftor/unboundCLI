package api

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jeeftor/unboundCLI/internal/logging"
)

// AdguardClient handles communication with the AdguardHome DNS rewrite API
type AdguardClient struct {
	BaseURL  string
	Username string
	Password string
	client   *http.Client
	Prompt   bool // Enable interactive prompting for API calls
}

// AdguardConfig represents configuration for AdguardHome API
type AdguardConfig struct {
	BaseURL  string `json:"base_url" mapstructure:"base_url"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	Insecure bool   `json:"insecure" mapstructure:"insecure"`
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
}

// Rewrite represents a DNS rewrite rule in AdguardHome
type Rewrite struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

// RewriteUpdate represents the structure for updating a rewrite rule
type RewriteUpdate struct {
	Target Rewrite `json:"target"`
	Update Rewrite `json:"update"`
}

// NewAdguardClient creates a new AdguardHome API client
func NewAdguardClient(config AdguardConfig) *AdguardClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Handle insecure TLS if specified in config
	if config.Insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &AdguardClient{
		BaseURL:  config.BaseURL,
		Username: config.Username,
		Password: config.Password,
		client:   client,
	}
}

// NewAdguardClientFromConfig creates an AdguardHome client from existing API config
func NewAdguardClientFromConfig(config Config, adguardBaseURL string) *AdguardClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Handle insecure TLS if specified in config
	if config.Insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &AdguardClient{
		BaseURL:  adguardBaseURL,
		Username: config.APIKey,    // Use API key as username
		Password: config.APISecret, // Use API secret as password
		client:   client,
	}
}

// promptForContinue prompts the user to continue with the API call
func (a *AdguardClient) promptForContinue(method, url string, jsonData []byte) bool {
	fmt.Printf("\n🔍 About to make AdguardHome API call:\n")
	fmt.Printf("📡 Method: %s\n", method)
	fmt.Printf("🌐 URL: %s\n", url)

	if jsonData != nil {
		// Pretty print JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, jsonData, "", "  "); err == nil {
			fmt.Printf("📄 JSON Data:\n%s\n", prettyJSON.String())
		} else {
			fmt.Printf("📄 JSON Data:\n%s\n", string(jsonData))
		}
	} else {
		fmt.Printf("📄 JSON Data: (none)\n")
	}

	fmt.Printf("\nContinue? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes"
}

// makeRequest performs an HTTP request with Basic Auth
func (a *AdguardClient) makeRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", a.BaseURL, endpoint)

	var body io.Reader
	var jsonData []byte
	if payload != nil {
		var err error
		jsonData, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	// Prompt user if enabled
	if a.Prompt {
		if !a.promptForContinue(method, url, jsonData) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Basic Auth header
	auth := fmt.Sprintf("%s:%s", a.Username, a.Password)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
	req.Header.Add("Content-Type", "application/json")

	logging.Debug("Making AdguardHome API request",
		"method", method,
		"url", url,
		"usernameLength", len(a.Username),
		"passwordLength", len(a.Password))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

// AddRewrite adds a new DNS rewrite rule
func (a *AdguardClient) AddRewrite(domain, answer string) error {
	rewrite := Rewrite{
		Domain: domain,
		Answer: answer,
	}

	resp, err := a.makeRequest("POST", "/control/rewrite/add", rewrite)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error("AdguardHome API request failed",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"domain", domain,
			"answer", answer)
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	logging.Debug("Successfully added DNS rewrite", "domain", domain, "answer", answer)
	return nil
}

// ListRewrites retrieves all DNS rewrite rules
func (a *AdguardClient) ListRewrites() ([]Rewrite, error) {
	resp, err := a.makeRequest("GET", "/control/rewrite/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error("AdguardHome API request failed",
			"statusCode", resp.StatusCode,
			"responseBody", string(body))
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var rewrites []Rewrite
	if err := json.NewDecoder(resp.Body).Decode(&rewrites); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logging.Debug("Retrieved DNS rewrites", "count", len(rewrites))
	return rewrites, nil
}

// UpdateRewrite updates an existing DNS rewrite rule
func (a *AdguardClient) UpdateRewrite(target, update Rewrite) error {
	updatePayload := RewriteUpdate{
		Target: target,
		Update: update,
	}

	resp, err := a.makeRequest("POST", "/control/rewrite/update", updatePayload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error("AdguardHome API request failed",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"targetDomain", target.Domain,
			"updateDomain", update.Domain)
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	logging.Debug("Successfully updated DNS rewrite",
		"targetDomain", target.Domain,
		"targetAnswer", target.Answer,
		"updateDomain", update.Domain,
		"updateAnswer", update.Answer)
	return nil
}

// DeleteRewrite removes a DNS rewrite rule
func (a *AdguardClient) DeleteRewrite(domain, answer string) error {
	rewrite := Rewrite{
		Domain: domain,
		Answer: answer,
	}

	resp, err := a.makeRequest("POST", "/control/rewrite/delete", rewrite)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error("AdguardHome API request failed",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"domain", domain,
			"answer", answer)
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	logging.Debug("Successfully deleted DNS rewrite", "domain", domain, "answer", answer)
	return nil
}

// RewriteExists checks if a specific rewrite rule exists
func (a *AdguardClient) RewriteExists(domain, answer string) (bool, error) {
	rewrites, err := a.ListRewrites()
	if err != nil {
		return false, err
	}

	for _, rewrite := range rewrites {
		if rewrite.Domain == domain && rewrite.Answer == answer {
			return true, nil
		}
	}

	return false, nil
}

// GetRewritesForDomain returns all rewrite rules for a specific domain
func (a *AdguardClient) GetRewritesForDomain(domain string) ([]Rewrite, error) {
	allRewrites, err := a.ListRewrites()
	if err != nil {
		return nil, err
	}

	var domainRewrites []Rewrite
	for _, rewrite := range allRewrites {
		if rewrite.Domain == domain {
			domainRewrites = append(domainRewrites, rewrite)
		}
	}

	return domainRewrites, nil
}
