package cmd

import (
	"fmt"

	"github.com/agynio/gh-pr-review/internal/autodetect"
	"github.com/agynio/gh-pr-review/internal/ghcli"
)

var apiClientFactory = func(host string) ghcli.API {
	return &ghcli.Client{Host: host}
}

// applyAutoDetection attempts to auto-detect repository and PR when not explicitly provided.
// It modifies the selector, repo, and pull parameters in place.
func applyAutoDetection(selector *string, repo *string, pull *int) {
	autoCtx, _ := autodetect.Detect() // Ignore errors, may not be in repo

	// If no explicit selector/repo, try using auto-detected values
	if *selector == "" && *pull == 0 && autoCtx.Number > 0 {
		*pull = autoCtx.Number
	}
	if *repo == "" && autoCtx.Owner != "" {
		*repo = fmt.Sprintf("%s/%s", autoCtx.Owner, autoCtx.Repo)
	}
}
