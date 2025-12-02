package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	pullURLRE       = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/([0-9]+)(?:/.*)?$`)
	fullReferenceRE = regexp.MustCompile(`^(?:([^/]+)/([^/]+))#([0-9]+)$`)
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

	return selector, nil
}

// Resolve interprets a selector, optional repo flag, and host (GH_HOST) into a concrete pull request identity.
func Resolve(selector, repoFlag, host string) (Identity, error) {
	selector = strings.TrimSpace(selector)
	repoFlag = strings.TrimSpace(repoFlag)
	host = defaultHost(host)

	if selector == "" {
		return Identity{}, errors.New("empty selector")
	}

	if id, err := parsePullURL(selector); err == nil {
		return id, nil
	}

	if id, err := parseFullReference(selector, host); err == nil {
		return id, nil
	}

	trimmed := strings.TrimPrefix(selector, "#")
	if n, err := strconv.Atoi(trimmed); err == nil && n > 0 {
		owner, repo, err := splitRepo(repoFlag)
		if err != nil {
			return Identity{}, fmt.Errorf("--repo must be owner/repo when using numeric selectors: %w", err)
		}
		return Identity{Owner: owner, Repo: repo, Host: host, Number: n}, nil
	}

	return Identity{}, fmt.Errorf("invalid pull request selector: %q", selector)
}

func defaultHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "github.com"
	}
	return host
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
		Host:   defaultHost(u.Hostname()),
		Number: number,
	}, nil
}

func parseFullReference(raw, host string) (Identity, error) {
	matches := fullReferenceRE.FindStringSubmatch(raw)
	if matches == nil {
		return Identity{}, errors.New("invalid reference")
	}
	number, _ := strconv.Atoi(matches[3])
	return Identity{
		Owner:  matches[1],
		Repo:   matches[2],
		Host:   defaultHost(host),
		Number: number,
	}, nil
}

func matchesNumber(selector string, target int) bool {
	if id, err := parsePullURL(selector); err == nil {
		return id.Number == target
	}
	if id, err := parseFullReference(selector, ""); err == nil {
		return id.Number == target
	}
	trimmed := strings.TrimPrefix(selector, "#")
	if n, err := strconv.Atoi(trimmed); err == nil {
		return n == target
	}
	return false
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
