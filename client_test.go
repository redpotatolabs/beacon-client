package beacon

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedRequest struct {
	method string
	path   string
	auth   string
	ctype  string
	ua     string
	body   []byte
}

func newTestServer(t *testing.T, status int, respBody string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.auth = r.Header.Get("Authorization")
		cap.ctype = r.Header.Get("Content-Type")
		cap.ua = r.Header.Get("User-Agent")
		cap.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

func TestErrorHappyPath(t *testing.T) {
	srv, cap := newTestServer(t, http.StatusOK, `{"ok":true,"message_id":"abc"}`)
	c := New(srv.URL, "bcn_xx_yy")

	resp, err := c.Error(context.Background(), ErrorRequest{
		Level:   "warn",
		Title:   "test",
		Message: "hello",
	})
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !resp.OK || resp.MessageID != "abc" {
		t.Fatalf("response: %+v", resp)
	}
	if cap.method != http.MethodPost || cap.path != "/v1/errors" {
		t.Errorf("wrong method/path: %s %s", cap.method, cap.path)
	}
	if cap.auth != "Bearer bcn_xx_yy" {
		t.Errorf("auth header: %q", cap.auth)
	}
	if cap.ctype != "application/json" {
		t.Errorf("content-type: %q", cap.ctype)
	}
	if !strings.HasPrefix(cap.ua, "beacon-client-go/") {
		t.Errorf("user-agent: %q", cap.ua)
	}

	var got ErrorRequest
	if err := json.Unmarshal(cap.body, &got); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if got.Level != "warn" || got.Title != "test" || got.Message != "hello" {
		t.Errorf("body roundtrip wrong: %+v", got)
	}
}

func TestErrorfBuildsMessage(t *testing.T) {
	srv, cap := newTestServer(t, http.StatusOK, `{"ok":true}`)
	c := New(srv.URL, "tok")

	if _, err := c.Errorf(context.Background(), "boom %d %s", 42, "things"); err != nil {
		t.Fatalf("Errorf: %v", err)
	}
	var got ErrorRequest
	_ = json.Unmarshal(cap.body, &got)
	if got.Message != "boom 42 things" {
		t.Fatalf("message: %q", got.Message)
	}
}

func TestDeployHappyPath(t *testing.T) {
	srv, cap := newTestServer(t, http.StatusOK, `{"ok":true,"message_id":"m1"}`)
	c := New(srv.URL, "tok")

	resp, err := c.Deploy(context.Background(), DeployRequest{
		Status: "success",
		Repo:   "redpotatolabs/beacon",
		SHA:    "abc1234",
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if resp.MessageID != "m1" {
		t.Fatalf("message id: %q", resp.MessageID)
	}
	if cap.path != "/v1/deploys" {
		t.Errorf("wrong path: %q", cap.path)
	}
}

func TestAPIErrorOn4xx(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusUnauthorized, `{"error":"invalid token"}`)
	c := New(srv.URL, "tok")

	_, err := c.Error(context.Background(), ErrorRequest{Message: "x"})
	if err == nil {
		t.Fatalf("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code: %d", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid token" {
		t.Errorf("message: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Error(), "401") || !strings.Contains(apiErr.Error(), "invalid token") {
		t.Errorf("Error() string: %q", apiErr.Error())
	}
}

func TestAPIErrorFallsBackToRawBody(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusBadGateway, "not json")
	c := New(srv.URL, "tok")

	_, err := c.Error(context.Background(), ErrorRequest{Message: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Message != "not json" {
		t.Errorf("fallback message: %q", apiErr.Message)
	}
}

func TestRequiresBaseURLAndToken(t *testing.T) {
	c := &Client{}
	if _, err := c.Error(context.Background(), ErrorRequest{Message: "x"}); err == nil {
		t.Errorf("expected error for missing BaseURL")
	}
	c = &Client{BaseURL: "http://x"}
	if _, err := c.Error(context.Background(), ErrorRequest{Message: "x"}); err == nil {
		t.Errorf("expected error for missing Token")
	}
}

func TestContextCancellation(t *testing.T) {
	srv, _ := newTestServer(t, http.StatusOK, `{"ok":true}`)
	c := New(srv.URL, "tok")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Error(ctx, ErrorRequest{Message: "x"})
	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
}

func TestDeployFromGitHubEnv(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "redpotatolabs/beacon")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_RUN_ID", "42")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_SHA", "abc1234")
	t.Setenv("GITHUB_ACTOR", "rlarsen")

	srv, cap := newTestServer(t, http.StatusOK, `{"ok":true}`)
	c := New(srv.URL, "tok")

	if _, err := c.DeployFromGitHubEnv(context.Background(), "success"); err != nil {
		t.Fatalf("DeployFromGitHubEnv: %v", err)
	}
	var got DeployRequest
	_ = json.Unmarshal(cap.body, &got)
	if got.Status != "success" || got.Repo != "redpotatolabs/beacon" {
		t.Errorf("wrong request: %+v", got)
	}
	if got.RunURL != "https://github.com/redpotatolabs/beacon/actions/runs/42" {
		t.Errorf("run url: %q", got.RunURL)
	}
}

func TestBaseURLTrailingSlashTrimmed(t *testing.T) {
	c := New("https://beacon.example/", "tok")
	if c.BaseURL != "https://beacon.example" {
		t.Errorf("BaseURL not trimmed: %q", c.BaseURL)
	}
}
