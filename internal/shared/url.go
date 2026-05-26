package shared

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

const forbiddenURLCharacters = "`$();|&<>\n\r\t\"'\\ "

type URLPolicy struct {
	allowedHosts []string
}

type URLOption func(*URLPolicy)

func WithAllowedHosts(hosts ...string) URLOption {
	return func(p *URLPolicy) {
		p.allowedHosts = hosts
	}
}

func ValidateHTTPSURL(raw string, options ...URLOption) error {
	if raw == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if strings.ContainsAny(raw, forbiddenURLCharacters) {
		return fmt.Errorf("URL contains forbidden characters")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL host cannot be empty")
	}
	if parsed.User != nil {
		return fmt.Errorf("URL must not contain userinfo")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("URL must not contain fragment")
	}

	policy := URLPolicy{}
	for _, apply := range options {
		apply(&policy)
	}
	if len(policy.allowedHosts) > 0 {
		host := parsed.Hostname()
		if !slices.Contains(policy.allowedHosts, host) {
			return fmt.Errorf("URL host %q is not in the allowlist", host)
		}
	}

	return nil
}
