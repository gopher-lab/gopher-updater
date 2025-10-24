package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ClientInterface defines the methods to interact with a Cosmos chain.
type ClientInterface interface {
	GetLatestBlockHeight(ctx context.Context) (int64, error)
	GetUpgradePlans(ctx context.Context) ([]Plan, error)
}

// Client for interacting with the Cosmos REST API.
type Client struct {
	rpcURL     *url.URL
	httpClient *http.Client
}

// NewClient creates a new Cosmos client.
func NewClient(rpcURL *url.URL, httpClient *http.Client) *Client {
	return &Client{
		rpcURL:     rpcURL,
		httpClient: httpClient,
	}
}

var _ ClientInterface = (*Client)(nil)

// Structs for parsing Cosmos API responses.
// Simplified for what we need.

type BlockHeader struct {
	Height string `json:"height"`
}

type Block struct {
	Header BlockHeader `json:"header"`
}

type LatestBlockResponse struct {
	Block Block `json:"block"`
}

// GetLatestBlockHeight returns the latest block height of the chain.
func (c *Client) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.rpcURL.JoinPath("/blocks/latest").String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var latestBlockResp LatestBlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&latestBlockResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	height, err := strconv.ParseInt(latestBlockResp.Block.Header.Height, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %w", err)
	}
	return height, nil
}

type Plan struct {
	Name   string `json:"name"`
	Height string `json:"height"`
}

type ProposalContent struct {
	Type string `json:"@type"`
	Plan Plan   `json:"plan"`
}

type Proposal struct {
	Status  string          `json:"status"`
	Content ProposalContent `json:"content"`
}

type ProposalsResponse struct {
	Proposals []Proposal `json:"proposals"`
}

// GetUpgradePlans finds all passed software upgrade proposals and returns their plans.
func (c *Client) GetUpgradePlans(ctx context.Context) ([]Plan, error) {
	reqURL := c.rpcURL.JoinPath("/cosmos/gov/v1/proposals")
	q := reqURL.Query()
	q.Set("proposal_status", "3") // PROPOSAL_STATUS_PASSED
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposals: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var proposalsResp ProposalsResponse
	if err := json.NewDecoder(resp.Body).Decode(&proposalsResp); err != nil {
		return nil, fmt.Errorf("failed to decode proposals response: %w", err)
	}

	var plans []Plan
	for _, p := range proposalsResp.Proposals {
		if p.Status == "PROPOSAL_STATUS_PASSED" && p.Content.Type == "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal" {
			plans = append(plans, p.Content.Plan)
		}
	}

	return plans, nil
}
