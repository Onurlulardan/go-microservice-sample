package permission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PermissionCheck represents a single permission check request
type PermissionCheck struct {
	UserID       string `json:"user_id"`
	ResourceSlug string `json:"resource_slug"`
	ActionSlug   string `json:"action_slug"`
}

// PermissionCheckResponse represents the response from permission service
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// BatchPermissionCheckRequest represents batch permission check request
type BatchPermissionCheckRequest struct {
	UserID string                `json:"user_id"`
	Checks []ResourceActionCheck `json:"checks"`
}

type ResourceActionCheck struct {
	ResourceSlug string `json:"resource_slug"`
	ActionSlug   string `json:"action_slug"`
}

// BatchPermissionCheckResponse represents batch permission check response
type BatchPermissionCheckResponse struct {
	Results map[string]bool `json:"results"` // key: "resource:action", value: allowed
}

// PermissionClient handles communication with permission service
type PermissionClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPermissionClient creates a new permission service client
func NewPermissionClient(baseURL string) *PermissionClient {
	return &PermissionClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CheckPermission checks if user has permission for specific resource and action
func (pc *PermissionClient) CheckPermission(userID, resourceSlug, actionSlug string) (bool, error) {
	check := PermissionCheck{
		UserID:       userID,
		ResourceSlug: resourceSlug,
		ActionSlug:   actionSlug,
	}

	jsonData, err := json.Marshal(check)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := pc.httpClient.Post(
		fmt.Sprintf("%s/api/permissions/check", pc.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("permission service returned status: %d", resp.StatusCode)
	}

	var result PermissionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Allowed, nil
}

// BatchCheckPermissions checks multiple permissions at once
func (pc *PermissionClient) BatchCheckPermissions(userID string, checks []ResourceActionCheck) (map[string]bool, error) {
	request := BatchPermissionCheckRequest{
		UserID: userID,
		Checks: checks,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := pc.httpClient.Post(
		fmt.Sprintf("%s/api/permissions/batch-check", pc.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("permission service returned status: %d", resp.StatusCode)
	}

	var result BatchPermissionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Results, nil
}

// Global permission client instance
var defaultClient *PermissionClient

// InitPermissionClient initializes the global permission client
func InitPermissionClient(baseURL string) {
	defaultClient = NewPermissionClient(baseURL)
}

// CheckPermission is a convenience function using the global client
func CheckPermission(userID, resourceSlug, actionSlug string) (bool, error) {
	if defaultClient == nil {
		return false, fmt.Errorf("permission client not initialized")
	}
	return defaultClient.CheckPermission(userID, resourceSlug, actionSlug)
}

// BatchCheckPermissions is a convenience function using the global client
func BatchCheckPermissions(userID string, checks []ResourceActionCheck) (map[string]bool, error) {
	if defaultClient == nil {
		return nil, fmt.Errorf("permission client not initialized")
	}
	return defaultClient.BatchCheckPermissions(userID, checks)
}
