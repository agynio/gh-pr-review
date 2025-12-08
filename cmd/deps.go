package cmd

import "github.com/agynio/gh-pr-review/internal/ghcli"

var apiClientFactory = func(host string) ghcli.API {
	return &ghcli.Client{Host: host}
}
