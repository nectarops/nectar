// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/domain"
	"github.com/nectarops/nectar/internal/security"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID, err := security.NewToken()
		if err != nil {
			requestID = "unavailable"
		}
		w.Header().Set("X-Request-ID", requestID)

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(recorder, r)

		s.logger.Info(
			"http request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func (s *Server) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		parsed, err := url.Parse(origin)
		if err != nil || !sameHost(parsed.Host, requestHost(r)) {
			writeError(w, http.StatusForbidden, "origin_rejected", "request origin is not allowed")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}

		user, err := s.auth.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			if !errors.Is(err, domain.ErrUnauthenticated) {
				s.logger.Warn("authenticate request", "error", err)
			}
			s.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}

		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

func (s *Server) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		if user.Role != "owner" {
			writeError(w, http.StatusForbidden, "forbidden", "owner access is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestHost(r *http.Request) string {
	forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if forwarded != "" {
		return forwarded
	}

	return r.Host
}

func sameHost(first, second string) bool {
	return strings.EqualFold(strings.TrimSpace(first), strings.TrimSpace(second))
}

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}

	return slog.Default()
}
