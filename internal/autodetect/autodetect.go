package autodetect

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
)

// Context represents the auto-detected repository and pull request context.
type Context struct {
	Owner  string
	Repo   string
	Number int // 0 if no PR detected
}

// Detect uses the gh CLI to auto-detect the current repository and pull request.
// It returns a Context with repository information even if no PR is found.
// Returns an error only if not in a git repository or if gh CLI fails critically.
func Detect() (Context, error) {
	ctx := Context{}

	// Detect repository first
	repoData, err := detectRepo()
	if err != nil {
		return ctx, err
	}
	ctx.Owner = repoData.Owner.Login
	ctx.Repo = repoData.Name

	// Try to detect PR (may fail if current branch has no PR)
	prNumber, _ := detectPR()
	ctx.Number = prNumber

	return ctx, nil
}

type repoResponse struct {
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name string `json:"name"`
}

func detectRepo() (repoResponse, error) {
	var result repoResponse

	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr == "" {
			stderrStr = err.Error()
		}
		return result, &DetectionError{
			Operation: "detect repository",
			Message:   stderrStr,
			Err:       err,
		}
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return result, &DetectionError{
			Operation: "parse repository data",
			Message:   err.Error(),
			Err:       err,
		}
	}

	return result, nil
}

type prResponse struct {
	Number int `json:"number"`
}

func detectPR() (int, error) {
	var result prResponse

	cmd := exec.Command("gh", "pr", "view", "--json", "number")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// This is expected when the current branch has no associated PR
		return 0, nil
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// If we can't parse, just return 0 (no PR)
		return 0, nil
	}

	return result.Number, nil
}

// DetectionError represents an error during auto-detection.
type DetectionError struct {
	Operation string
	Message   string
	Err       error
}

func (e *DetectionError) Error() string {
	return "auto-detect " + e.Operation + ": " + e.Message
}

func (e *DetectionError) Unwrap() error {
	return e.Err
}
