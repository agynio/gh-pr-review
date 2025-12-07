package resolver

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	pullURLRE = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/([0-9]+)(?:/.*)?$`)
)

// Identity represents a fully-resolved pull request reference.
type Identity struct {
	Owner  string
	Repo   string
	Host   string
	Number int
}

// NormalizeSelector ensures that either an explicit selector or --pr flag is present and mutually consistent.
func NormalizeSelector(selector string, prFlag int) (string, error) {
	selector = strings.TrimSpace(selector)

	switch {
	case selector != "" && prFlag > 0:
		if !matchesNumber(selector, prFlag) {
			return "", fmt.Errorf("pull request argument %q does not match --pr=%d", selector, prFlag)
		}
	case selector == "" && prFlag > 0:
		selector = strconv.Itoa(prFlag)
	}

	if selector == "" {
		return "", errors.New("must specify a pull request via --pr or selector")
	}

	if isNumeric(selector) {
		return selector, nil
	}

	if _, err := parsePullURL(selector); err == nil {
		return selector, nil
	}

	return "", fmt.Errorf("invalid pull request selector %q: must be a pull request URL or number", selector)
}

// Resolve interprets a selector, optional repo flag, and host (GH_HOST) into a concrete pull request identity.
func Resolve(selector, repoFlag, host string) (Identity, error) {
	selector = strings.TrimSpace(selector)
	repoFlag = strings.TrimSpace(repoFlag)
	host = sanitizeHost(host)

	if selector == "" {
		return Identity{}, errors.New("empty selector")
	}

	if id, err := parsePullURL(selector); err == nil {
		return id, nil
	}

	if n, err := strconv.Atoi(selector); err == nil && n > 0 {
		owner, repo, err := splitRepo(repoFlag)
		if err != nil {
			return Identity{}, fmt.Errorf("--repo must be owner/repo when using numeric selectors: %w", err)
		}
		return Identity{Owner: owner, Repo: repo, Host: host, Number: n}, nil
	}

	return Identity{}, fmt.Errorf("invalid pull request selector: %q", selector)
}

func parsePullURL(raw string) (Identity, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Identity{}, err
	}
	if u.Host == "" {
		return Identity{}, errors.New("missing host")
	}
	matches := pullURLRE.FindStringSubmatch(u.Path)
	if matches == nil {
		return Identity{}, errors.New("not a pull request url")
	}
	number, _ := strconv.Atoi(matches[3])
	return Identity{
		Owner:  matches[1],
		Repo:   matches[2],
		Host:   sanitizeHost(u.Host),
		Number: number,
	}, nil
}

func matchesNumber(selector string, target int) bool {
	if id, err := parsePullURL(selector); err == nil {
		return id.Number == target
	}
	if n, err := strconv.Atoi(selector); err == nil {
		return n == target
	}
	return false
}

func isNumeric(selector string) bool {
	if selector == "" {
		return false
	}
	for _, r := range selector {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitRepo(repoFlag string) (string, string, error) {
	if repoFlag == "" {
		return "", "", errors.New("missing --repo")
	}
	parts := strings.Split(repoFlag, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("expected owner/repo")
	}
	return parts[0], parts[1], nil
}

func sanitizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "github.com"
	}

	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			raw = u.Host
		} else {
			raw = strings.TrimPrefix(strings.TrimPrefix(lower, "http://"), "https://")
		}
	}

	if strings.Contains(raw, "/") {
		raw = strings.SplitN(raw, "/", 2)[0]
	}

	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	} else if idx := strings.Index(raw, ":"); idx >= 0 {
		raw = raw[:idx]
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "github.com"
	}

	return strings.ToLower(raw)
}
