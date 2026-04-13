package cmd

import "github.com/agynio/gh-pr-review/internal/ghcli"

var apiClientFactory = func(host string) ghcli.API {
	return &ghcli.Client{Host: host}
}

// inferRepo fills in repo from the current git context when not explicitly provided.
func inferRepo(repo *string) {
	if *repo == "" {
		if r, err := ghcli.CurrentRepo(); err == nil {
			*repo = r
		}
	}
}

// inferPR fills in the PR number from the current branch when not explicitly provided.
func inferPR(selector string, pr *int) {
	if selector == "" && *pr == 0 {
		if n, err := ghcli.CurrentPR(); err == nil {
			*pr = n
		}
	}
}
