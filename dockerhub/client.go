package dockerhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ClientInterface defines the methods to interact with DockerHub.
type ClientInterface interface {
	RetagImage(ctx context.Context, repoPath, sourceTag, targetTag string) error
	TagExists(ctx context.Context, repoPath, tag string) (bool, error)
}

// Client for interacting with the DockerHub API.
type Client struct {
	user       string
	password   string
	httpClient *http.Client
}

// NewClient creates a new DockerHub client.
func NewClient(user, password string, httpClient *http.Client) *Client {
	return &Client{
		user:       user,
		password:   password,
		httpClient: httpClient,
	}
}

var _ ClientInterface = (*Client)(nil)

type authResponse struct {
	Token string `json:"token"`
}

func (c *Client) getBearerToken(ctx context.Context, scope string) (string, error) {
	authURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=%s", url.QueryEscape(scope))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth failed with status: %s", resp.Status)
	}

	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}
	return authResp.Token, nil
}

// TagExists checks if a specific tag exists for a repository.
func (c *Client) TagExists(ctx context.Context, repoPath, tag string) (bool, error) {
	scope := fmt.Sprintf("repository:%s:pull", repoPath)
	token, err := c.getBearerToken(ctx, scope)
	if err != nil {
		return false, fmt.Errorf("failed to get auth token: %w", err)
	}

	const registryAPI = "https://registry-1.docker.io/v2"

	manifestURL := fmt.Sprintf("%s/%s/manifests/%s", registryAPI, repoPath, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create manifest head request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code when checking tag: %s", resp.Status)
}

// RetagImage retags a Docker image from a source tag to a target tag.
func (c *Client) RetagImage(ctx context.Context, repoPath, sourceTag, targetTag string) error {
	scope := fmt.Sprintf("repository:%s:pull,push", repoPath)
	token, err := c.getBearerToken(ctx, scope)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	const registryAPI = "https://registry-1.docker.io/v2"

	manifestURL := fmt.Sprintf("%s/%s/manifests/%s", registryAPI, repoPath, sourceTag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create manifest get request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get manifest, status: %s, body: %s", resp.Status, string(body))
	}

	manifest, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest body: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")

	// Now PUT the manifest with the new tag
	targetURL := fmt.Sprintf("%s/%s/manifests/%s", registryAPI, repoPath, targetTag)
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, targetURL, bytes.NewBuffer(manifest))
	if err != nil {
		return fmt.Errorf("failed to create manifest put request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to put manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to put manifest, status: %s, body: %s", resp.Status, string(body))
	}

	return nil
}
