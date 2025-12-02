package cmd

import "github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"

var apiClientFactory = func(host string) ghcli.API {
	return &ghcli.Client{Host: host}
}
