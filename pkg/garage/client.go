// Package garage provides a client for the Garage Admin API v2
package garage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a client for the Garage Admin API v2
type Client struct {
	endpoint   string
	adminToken string
	httpClient *http.Client
}

// NewClient creates a new Garage Admin API client
func NewClient(endpoint, adminToken string) *Client {
	return &Client{
		endpoint:   endpoint,
		adminToken: adminToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request to the Garage Admin API
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Bucket represents a Garage bucket
type Bucket struct {
	ID            string            `json:"id"`
	GlobalAliases []string          `json:"globalAliases"`
	LocalAliases  map[string]string `json:"localAliases,omitempty"`
	Keys          []BucketKeyPerm   `json:"keys,omitempty"`
	Quotas        *BucketQuotas     `json:"quotas,omitempty"`
}

// BucketKeyPerm represents permissions for a key on a bucket
type BucketKeyPerm struct {
	AccessKeyID string `json:"accessKeyId"`
	Name        string `json:"name"`
	Permissions struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
		Owner bool `json:"owner"`
	} `json:"permissions"`
}

// BucketQuotas represents bucket quotas
type BucketQuotas struct {
	MaxSize    *int64 `json:"maxSize,omitempty"`
	MaxObjects *int64 `json:"maxObjects,omitempty"`
}

// CreateBucketRequest is the request to create a bucket
type CreateBucketRequest struct {
	GlobalAlias *string `json:"globalAlias,omitempty"`
	LocalAlias  *struct {
		AccessKeyID string `json:"accessKeyId"`
		Alias       string `json:"alias"`
	} `json:"localAlias,omitempty"`
}

// CreateBucket creates a new bucket
func (c *Client) CreateBucket(ctx context.Context, req *CreateBucketRequest) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "POST", "/v1/bucket", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBucket retrieves a bucket by ID
func (c *Client) GetBucket(ctx context.Context, bucketID string) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "GET", "/v1/bucket?id="+bucketID, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBucketByAlias retrieves a bucket by global alias
func (c *Client) GetBucketByAlias(ctx context.Context, globalAlias string) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "GET", "/v1/bucket?globalAlias="+globalAlias, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteBucket deletes a bucket
func (c *Client) DeleteBucket(ctx context.Context, bucketID string) error {
	return c.doRequest(ctx, "DELETE", "/v1/bucket?id="+bucketID, nil, nil)
}

// UpdateBucketRequest is the request to update a bucket
type UpdateBucketRequest struct {
	ID          string `json:"id"`
	GlobalAlias *struct {
		Add    *string `json:"add,omitempty"`
		Remove *string `json:"remove,omitempty"`
	} `json:"globalAlias,omitempty"`
	LocalAlias *struct {
		AccessKeyID string  `json:"accessKeyId"`
		Add         *string `json:"add,omitempty"`
		Remove      *string `json:"remove,omitempty"`
	} `json:"localAlias,omitempty"`
	Quotas *BucketQuotas `json:"quotas,omitempty"`
}

// UpdateBucket updates a bucket
func (c *Client) UpdateBucket(ctx context.Context, req *UpdateBucketRequest) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "PUT", "/v1/bucket", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Key represents a Garage access key
type Key struct {
	AccessKeyID     string           `json:"accessKeyId"`
	Name            string           `json:"name"`
	SecretAccessKey string           `json:"secretAccessKey,omitempty"`
	Permissions     KeyPermissions   `json:"permissions"`
	Buckets         []KeyBucketPerms `json:"buckets,omitempty"`
}

// KeyPermissions represents global permissions for a key
type KeyPermissions struct {
	CreateBucket bool `json:"createBucket"`
}

// KeyBucketPerms represents permissions for a key on buckets
type KeyBucketPerms struct {
	ID            string   `json:"id"`
	GlobalAliases []string `json:"globalAliases"`
	Permissions   struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
		Owner bool `json:"owner"`
	} `json:"permissions"`
}

// CreateKeyRequest is the request to create a key
type CreateKeyRequest struct {
	Name string `json:"name"`
}

// CreateKey creates a new access key
func (c *Client) CreateKey(ctx context.Context, req *CreateKeyRequest) (*Key, error) {
	var result Key
	err := c.doRequest(ctx, "POST", "/v1/key", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKey retrieves a key by ID
func (c *Client) GetKey(ctx context.Context, accessKeyID string) (*Key, error) {
	var result Key
	err := c.doRequest(ctx, "GET", "/v1/key?id="+accessKeyID, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKeyByName searches for a key by name pattern and returns it if exactly one match is found
func (c *Client) GetKeyByName(ctx context.Context, name string) (*Key, error) {
	var results []KeyInfo
	err := c.doRequest(ctx, "GET", "/v1/key?search="+name, nil, &results)
	if err != nil {
		return nil, err
	}
	// Find exact match
	for _, k := range results {
		if k.Name == name {
			// Get full key details
			return c.GetKey(ctx, k.ID)
		}
	}
	return nil, fmt.Errorf("key with name %q not found", name)
}

// KeyInfo represents basic key information from list/search
type KeyInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DeleteKey deletes a key
func (c *Client) DeleteKey(ctx context.Context, accessKeyID string) error {
	return c.doRequest(ctx, "DELETE", "/v1/key?id="+accessKeyID, nil, nil)
}

// UpdateKeyRequest is the request to update a key
type UpdateKeyRequest struct {
	AccessKeyID string          `json:"accessKeyId"`
	Name        *string         `json:"name,omitempty"`
	Allow       *KeyPermissions `json:"allow,omitempty"`
	Deny        *KeyPermissions `json:"deny,omitempty"`
}

// UpdateKey updates a key
func (c *Client) UpdateKey(ctx context.Context, req *UpdateKeyRequest) (*Key, error) {
	var result Key
	err := c.doRequest(ctx, "PUT", "/v1/key", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GrantKeyAccessRequest is the request to grant key access to a bucket
type GrantKeyAccessRequest struct {
	BucketID    string `json:"bucketId"`
	AccessKeyID string `json:"accessKeyId"`
	Permissions struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
		Owner bool `json:"owner"`
	} `json:"permissions"`
}

// GrantKeyAccess grants a key access to a bucket
func (c *Client) GrantKeyAccess(ctx context.Context, req *GrantKeyAccessRequest) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "POST", "/v1/bucket/allow", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeKeyAccessRequest is the request to revoke key access from a bucket
type RevokeKeyAccessRequest struct {
	BucketID    string `json:"bucketId"`
	AccessKeyID string `json:"accessKeyId"`
}

// RevokeKeyAccess revokes a key's access from a bucket
func (c *Client) RevokeKeyAccess(ctx context.Context, req *RevokeKeyAccessRequest) (*Bucket, error) {
	var result Bucket
	err := c.doRequest(ctx, "POST", "/v1/bucket/deny", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
