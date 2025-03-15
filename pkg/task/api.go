package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is a client for making API calls
type APIClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

// APIRequest represents an API request
type APIRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	Query   map[string]string
}

// APIResponse represents an API response
type APIResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string, timeout time.Duration) *APIClient {
	return &APIClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

// SetHeader sets a header for all requests
func (c *APIClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders sets multiple headers for all requests
func (c *APIClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

// Request makes an API request
func (c *APIClient) Request(ctx context.Context, req APIRequest) (*APIResponse, error) {
	// Prepare URL
	url := c.baseURL + req.Path

	// Prepare body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set content type if not set and body is not nil
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set query parameters
	if req.Query != nil {
		q := httpReq.URL.Query()
		for k, v := range req.Query {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Make the request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// Get makes a GET request
func (c *APIClient) Get(ctx context.Context, path string, query map[string]string, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, APIRequest{
		Method:  http.MethodGet,
		Path:    path,
		Query:   query,
		Headers: headers,
	})
}

// Post makes a POST request
func (c *APIClient) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, APIRequest{
		Method:  http.MethodPost,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Put makes a PUT request
func (c *APIClient) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, APIRequest{
		Method:  http.MethodPut,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Delete makes a DELETE request
func (c *APIClient) Delete(ctx context.Context, path string, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, APIRequest{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: headers,
	})
}

// APITask creates a task function for making an API request
func APITask(client *APIClient, req APIRequest) TaskFunc {
	return func(ctx context.Context, params interface{}) (interface{}, error) {
		// If params is provided, use it as the request body
		if params != nil {
			req.Body = params
		}

		// Make the request
		resp, err := client.Request(ctx, req)
		if err != nil {
			return nil, err
		}

		// Check if the response is successful
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(resp.Body))
		}

		// Parse the response body if it's JSON
		var result interface{}
		if len(resp.Body) > 0 && resp.Headers.Get("Content-Type") == "application/json" {
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return nil, fmt.Errorf("failed to parse response body: %w", err)
			}
			return result, nil
		}

		return resp.Body, nil
	}
}
