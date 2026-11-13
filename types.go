// Package beacon is the Go client library for the beacon Discord bot's HTTP
// API. It also defines the wire types both the server and the client share,
// so adding or renaming a field in one place is a compile error in the other.
package beacon

import "fmt"

// FieldKV is an optional key/value pair attached to an error embed.
type FieldKV struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// ErrorRequest is the JSON body posted to POST /v1/errors. Only Message is
// required. Level defaults to "error" server-side; Source defaults to the
// authenticated app's name.
type ErrorRequest struct {
	Level   string    `json:"level,omitempty"`
	Title   string    `json:"title,omitempty"`
	Message string    `json:"message"`
	Source  string    `json:"source,omitempty"`
	Fields  []FieldKV `json:"fields,omitempty"`
}

// DeployRequest is the JSON body posted to POST /v1/deploys. Repo and Status
// are required.
type DeployRequest struct {
	Status string `json:"status"`
	Repo   string `json:"repo"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha,omitempty"`
	Actor  string `json:"actor,omitempty"`
	RunURL string `json:"run_url,omitempty"`
}

// Response is the JSON body returned by both POST endpoints on success.
type Response struct {
	OK        bool   `json:"ok"`
	MessageID string `json:"message_id,omitempty"`
}

// APIError is returned by the client when the server responds with a non-2xx
// status. StatusCode is the HTTP status; Message is the parsed `error` field
// from the response body, when present.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("beacon: http %d", e.StatusCode)
	}
	return fmt.Sprintf("beacon: http %d: %s", e.StatusCode, e.Message)
}
