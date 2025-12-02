package ghcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Client executes GitHub API requests through the `gh` CLI to reuse
// the authenticated context and host configuration provided by the user.
type Client struct {
	Host string
}

// API defines the subset of GitHub API interactions required by the command logic.
type API interface {
	REST(method, path string, params map[string]string, body interface{}, result interface{}) error
	GraphQL(query string, variables map[string]interface{}, result interface{}) error
}

// APIError wraps errors returned by the `gh api` command, exposing the HTTP status code when detected.
type APIError struct {
	StatusCode int
	Message    string
	Stderr     string
	Err        error
}

func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("gh api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("gh api error: %s", e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

var statusRE = regexp.MustCompile(`HTTP\s+(\d{3})\b`)

func wrapError(err error, stderr string) error {
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = err.Error()
	}

	apiErr := &APIError{Message: message, Stderr: stderr, Err: err}
	if matches := statusRE.FindStringSubmatch(stderr); len(matches) == 2 {
		if code, convErr := strconv.Atoi(matches[1]); convErr == nil {
			apiErr.StatusCode = code
		}
	}
	return apiErr
}

// REST invokes the REST API using `gh api`.
// The result parameter must be a pointer and will be unmarshaled from JSON.
func (c *Client) REST(method, path string, params map[string]string, body interface{}, result interface{}) error {
	args := []string{"api"}
	if host := strings.TrimSpace(c.Host); host != "" {
		args = append(args, "--hostname", host)
	}

	args = append(args, path, "-X", method)

	for key, value := range params {
		args = append(args, "-f", fmt.Sprintf("%s=%s", key, value))
	}

	var stdinData []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		stdinData = data
		args = append(args, "--input", "-")
	}

	stdout, stderr, err := runGh(args, stdinData)
	if err != nil {
		return wrapError(err, stderr)
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(stdout, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}

// GraphQL issues a GraphQL operation through `gh api graphql`.
func (c *Client) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	payload := map[string]interface{}{
		"query": query,
	}
	if len(variables) > 0 {
		payload["variables"] = variables
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal graphql payload: %w", err)
	}

	args := []string{"api", "graphql"}
	if host := strings.TrimSpace(c.Host); host != "" {
		args = append(args, "--hostname", host)
	}
	args = append(args, "--input", "-")

	stdout, stderr, err := runGh(args, data)
	if err != nil {
		return wrapError(err, stderr)
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(stdout, result); err != nil {
		return fmt.Errorf("unmarshal graphql response: %w", err)
	}
	return nil
}

// runGh executes the `gh` CLI command with provided arguments and optional stdin data.
func runGh(args []string, stdin []byte) ([]byte, string, error) {
	cmd := exec.Command("gh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, stderr.String(), err
	}

	return stdout.Bytes(), stderr.String(), nil
}
