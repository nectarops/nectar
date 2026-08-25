// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestBaseURLUsesForwardedHTTPSOrigin(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "http://nectar:8080/api/v1/node-enrollments", nil)
	request.Header.Set("X-Forwarded-Host", "nectar.example.com")
	request.Header.Set("X-Forwarded-Proto", "https")

	baseURL, err := requestBaseURL(request)
	if err != nil {
		t.Fatalf("requestBaseURL() error = %v", err)
	}
	if baseURL != "https://nectar.example.com" {
		t.Fatalf("requestBaseURL() = %q, want https://nectar.example.com", baseURL)
	}
}

func TestRequestBaseURLRejectsShellMetacharactersInHost(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "http://nectar/api/v1/node-enrollments", nil)
	request.Host = "nectar.example.com'$(id)"
	if _, err := requestBaseURL(request); err == nil {
		t.Fatal("requestBaseURL() accepted a host containing shell metacharacters")
	}
}

func TestShellQuoteEscapesSingleQuote(t *testing.T) {
	t.Parallel()

	if got := shellQuote("one'two"); got != `'one'"'"'two'` {
		t.Fatalf("shellQuote() = %q", got)
	}
}
