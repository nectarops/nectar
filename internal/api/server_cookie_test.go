// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClearSessionCookieUsesRequestTransport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		url            string
		forwardedProto string
		cookieSecure   bool
		wantSecure     bool
	}{
		{
			name:       "direct HTTP setup",
			url:        "http://192.0.2.10/",
			wantSecure: false,
		},
		{
			name:       "direct HTTPS",
			url:        "https://nectar.example.com/",
			wantSecure: true,
		},
		{
			name:           "HTTPS through trusted proxy",
			url:            "http://nectar:8080/",
			forwardedProto: "https",
			wantSecure:     true,
		},
		{
			name:         "forced secure cookie",
			url:          "http://192.0.2.10/",
			cookieSecure: true,
			wantSecure:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, test.url, nil)
			if test.forwardedProto != "" {
				request.Header.Set("X-Forwarded-Proto", test.forwardedProto)
			}
			recorder := httptest.NewRecorder()
			server := &Server{cookieSecure: test.cookieSecure}

			server.clearSessionCookie(recorder, request)

			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookie count = %d, want 1", len(cookies))
			}
			cookie := cookies[0]
			if cookie.Secure != test.wantSecure {
				t.Fatalf("Secure = %t, want %t", cookie.Secure, test.wantSecure)
			}
			if cookie.Name != sessionCookieName || cookie.Value != "" || cookie.Path != "/" {
				t.Fatalf("cleared cookie = %#v", cookie)
			}
			if cookie.MaxAge != -1 || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cleared cookie attributes = %#v", cookie)
			}
		})
	}
}
