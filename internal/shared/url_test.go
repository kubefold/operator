package shared

import "testing"

func TestValidateHTTPSURL(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		options  []URLOption
		wantPass bool
	}{
		{"valid https", "https://example.com/weights.zst", nil, true},
		{"valid https with port", "https://example.com:8443/weights.zst", nil, true},
		{"empty", "", nil, false},
		{"http scheme", "http://example.com/weights", nil, false},
		{"ftp scheme", "ftp://example.com/weights", nil, false},
		{"shell injection semicolon", "https://example.com/weights;rm -rf /", nil, false},
		{"shell injection backtick", "https://example.com/`whoami`", nil, false},
		{"shell injection dollar", "https://example.com/$(whoami)", nil, false},
		{"shell injection pipe", "https://example.com/x|cat", nil, false},
		{"shell injection ampersand", "https://example.com/x&&id", nil, false},
		{"newline", "https://example.com/x\n", nil, false},
		{"userinfo", "https://user:pass@example.com/weights", nil, false},
		{"fragment", "https://example.com/weights#frag", nil, false},
		{"allowlist match", "https://trusted.example.com/file", []URLOption{WithAllowedHosts("trusted.example.com")}, true},
		{"allowlist miss", "https://evil.example.com/file", []URLOption{WithAllowedHosts("trusted.example.com")}, false},
		{"allowlist multiple match", "https://b.com/x", []URLOption{WithAllowedHosts("a.com", "b.com")}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHTTPSURL(tc.url, tc.options...)
			if tc.wantPass && err != nil {
				t.Fatalf("expected pass, got error: %v", err)
			}
			if !tc.wantPass && err == nil {
				t.Fatalf("expected error for url %q", tc.url)
			}
		})
	}
}
