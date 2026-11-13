package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultUserAgent  = "beacon-client-go/0.1"
	maxResponseBodyKB = 64
)

// Client is a beacon HTTP client. Safe for concurrent use; reuse a single
// Client across goroutines. Construct with New.
type Client struct {
	// BaseURL of the beacon server, e.g. "https://beacon.example".
	// No trailing slash; routes are appended directly.
	BaseURL string
	// Token is the bearer token issued by /beacon admin add or rotate.
	Token string
	// HTTPClient is the underlying client. If nil, a default with a 10s
	// timeout is used.
	HTTPClient *http.Client
	// UserAgent is sent on every request. Defaults to beacon-client-go/0.1.
	UserAgent string
}

// New returns a Client configured against baseURL using token. Trailing
// slashes on baseURL are trimmed.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		UserAgent:  defaultUserAgent,
	}
}

// Error posts an error event to /v1/errors. Returns the server's response
// (including the Discord message id) on success, or an *APIError on a non-2xx
// status, or a transport/encoding error otherwise.
func (c *Client) Error(ctx context.Context, req ErrorRequest) (*Response, error) {
	return c.post(ctx, "/v1/errors", req)
}

// Errorf is a convenience wrapper that posts a simple error event whose
// message is built with fmt.Sprintf(format, args...).
func (c *Client) Errorf(ctx context.Context, format string, args ...any) (*Response, error) {
	return c.Error(ctx, ErrorRequest{Message: fmt.Sprintf(format, args...)})
}

// Deploy posts a deploy event to /v1/deploys.
func (c *Client) Deploy(ctx context.Context, req DeployRequest) (*Response, error) {
	return c.post(ctx, "/v1/deploys", req)
}

// DeployFromGitHubEnv reads the standard GITHUB_* environment variables and
// posts a deploy event. Intended for use from a `run:` step in GitHub Actions
// where these env vars are already set by the runner.
//
// status is passed through verbatim; pass "success", "failure", "pending",
// or "cancelled" (matching the values /v1/deploys accepts).
func (c *Client) DeployFromGitHubEnv(ctx context.Context, status string) (*Response, error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	server := os.Getenv("GITHUB_SERVER_URL")
	if server == "" {
		server = "https://github.com"
	}
	runID := os.Getenv("GITHUB_RUN_ID")
	runURL := ""
	if repo != "" && runID != "" {
		runURL = fmt.Sprintf("%s/%s/actions/runs/%s", server, repo, runID)
	}
	return c.Deploy(ctx, DeployRequest{
		Status: status,
		Repo:   repo,
		Ref:    os.Getenv("GITHUB_REF"),
		SHA:    os.Getenv("GITHUB_SHA"),
		Actor:  os.Getenv("GITHUB_ACTOR"),
		RunURL: runURL,
	})
}

func (c *Client) post(ctx context.Context, path string, body any) (*Response, error) {
	if c.BaseURL == "" {
		return nil, errors.New("beacon: BaseURL is required")
	}
	if c.Token == "" {
		return nil, errors.New("beacon: Token is required")
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("beacon: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("beacon: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	ua := c.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}
	req.Header.Set("User-Agent", ua)

	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("beacon: post %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyKB*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var body struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &body) == nil && body.Error != "" {
			apiErr.Message = body.Error
		} else {
			apiErr.Message = strings.TrimSpace(string(raw))
		}
		return nil, apiErr
	}
	var out Response
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("beacon: decode response: %w", err)
		}
	}
	return &out, nil
}
