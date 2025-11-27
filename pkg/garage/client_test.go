package garage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:3903", "test-token")
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	if client.endpoint != "http://localhost:3903" {
		t.Errorf("Expected endpoint 'http://localhost:3903', got '%s'", client.endpoint)
	}
	if client.adminToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", client.adminToken)
	}
}

func TestCreateBucket(t *testing.T) {
	tests := []struct {
		name           string
		globalAlias    *string
		responseStatus int
		responseBody   Bucket
		expectError    bool
	}{
		{
			name:           "successful creation with global alias",
			globalAlias:    stringPtr("test-bucket"),
			responseStatus: http.StatusOK,
			responseBody: Bucket{
				ID:            "bucket-123",
				GlobalAliases: []string{"test-bucket"},
			},
			expectError: false,
		},
		{
			name:           "successful creation without alias",
			globalAlias:    nil,
			responseStatus: http.StatusOK,
			responseBody: Bucket{
				ID:            "bucket-456",
				GlobalAliases: []string{},
			},
			expectError: false,
		},
		{
			name:           "server error",
			globalAlias:    stringPtr("test-bucket"),
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/v1/bucket" {
					t.Errorf("Expected path '/v1/bucket', got '%s'", r.URL.Path)
				}

				// Verify authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer test-token" {
					t.Errorf("Expected 'Bearer test-token', got '%s'", authHeader)
				}

				w.WriteHeader(tt.responseStatus)
				if tt.responseStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			req := &CreateBucketRequest{
				GlobalAlias: tt.globalAlias,
			}

			bucket, err := client.CreateBucket(context.Background(), req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if bucket == nil {
					t.Fatal("Expected bucket, got nil")
				}
				if bucket.ID != tt.responseBody.ID {
					t.Errorf("Expected ID '%s', got '%s'", tt.responseBody.ID, bucket.ID)
				}
			}
		})
	}
}

func TestGetBucket(t *testing.T) {
	tests := []struct {
		name           string
		bucketID       string
		responseStatus int
		responseBody   Bucket
		expectError    bool
	}{
		{
			name:           "successful get",
			bucketID:       "bucket-123",
			responseStatus: http.StatusOK,
			responseBody: Bucket{
				ID:            "bucket-123",
				GlobalAliases: []string{"my-bucket"},
			},
			expectError: false,
		},
		{
			name:           "bucket not found",
			bucketID:       "nonexistent",
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				expectedPath := "/v1/bucket"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.responseStatus)
				if tt.responseStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			bucket, err := client.GetBucket(context.Background(), tt.bucketID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if bucket.ID != tt.responseBody.ID {
					t.Errorf("Expected ID '%s', got '%s'", tt.responseBody.ID, bucket.ID)
				}
			}
		})
	}
}

func TestDeleteBucket(t *testing.T) {
	tests := []struct {
		name           string
		bucketID       string
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful deletion",
			bucketID:       "bucket-123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "bucket not found",
			bucketID:       "nonexistent",
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(tt.responseStatus)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.DeleteBucket(context.Background(), tt.bucketID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCreateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/key" {
			t.Errorf("Expected path '/v1/key', got '%s'", r.URL.Path)
		}

		response := Key{
			AccessKeyID:     "GK123456",
			Name:            "test-key",
			SecretAccessKey: "secret123",
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	req := &CreateKeyRequest{Name: "test-key"}

	key, err := client.CreateKey(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if key.Name != "test-key" {
		t.Errorf("Expected name 'test-key', got '%s'", key.Name)
	}
	if key.AccessKeyID != "GK123456" {
		t.Errorf("Expected access key 'GK123456', got '%s'", key.AccessKeyID)
	}
}

func TestGrantKeyAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/bucket/allow" {
			t.Errorf("Expected path '/v1/bucket/allow', got '%s'", r.URL.Path)
		}

		response := Bucket{
			ID:            "bucket-123",
			GlobalAliases: []string{"test-bucket"},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	req := &GrantKeyAccessRequest{
		BucketID:    "bucket-123",
		AccessKeyID: "GK123456",
		Permissions: struct {
			Read  bool `json:"read"`
			Write bool `json:"write"`
			Owner bool `json:"owner"`
		}{
			Read:  true,
			Write: true,
			Owner: false,
		},
	}

	bucket, err := client.GrantKeyAccess(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if bucket.ID != "bucket-123" {
		t.Errorf("Expected ID 'bucket-123', got '%s'", bucket.ID)
	}
}

func stringPtr(s string) *string {
	return &s
}
