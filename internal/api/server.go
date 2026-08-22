// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/application"
	"github.com/nectarops/nectar/internal/domain"
)

const sessionCookieName = "dw_session"

type Readiness interface {
	Ready(context.Context) error
}

type Options struct {
	Logger        *slog.Logger
	Auth          *application.AuthService
	Cluster       *application.ClusterService
	Deployments   *application.DeploymentService
	Store         Readiness
	Docker        Readiness
	RequireDocker bool
	Assets        fs.FS
	CookieSecure  bool
	SourceURL     string
	Version       string
	Commit        string
}

type Server struct {
	logger        *slog.Logger
	auth          *application.AuthService
	cluster       *application.ClusterService
	deployments   *application.DeploymentService
	store         Readiness
	docker        Readiness
	requireDocker bool
	assets        fs.FS
	cookieSecure  bool
	sourceURL     string
	version       string
	commit        string
}

func NewServer(options Options) (*Server, error) {
	if options.Auth == nil {
		return nil, errors.New("authentication service is required")
	}
	if options.Cluster == nil {
		return nil, errors.New("cluster service is required")
	}
	if options.Deployments == nil {
		return nil, errors.New("deployment service is required")
	}
	if options.Store == nil {
		return nil, errors.New("store readiness check is required")
	}
	if options.Assets == nil {
		return nil, errors.New("frontend assets are required")
	}

	return &Server{
		logger:        loggerOrDefault(options.Logger),
		auth:          options.Auth,
		cluster:       options.Cluster,
		deployments:   options.Deployments,
		store:         options.Store,
		docker:        options.Docker,
		requireDocker: options.RequireDocker,
		assets:        options.Assets,
		cookieSecure:  options.CookieSecure,
		sourceURL:     options.SourceURL,
		version:       options.Version,
		commit:        options.Commit,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.handleLiveness)
	mux.HandleFunc("GET /health/ready", s.handleReadiness)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/setup/status", s.handleSetupStatus)
	mux.HandleFunc("POST /api/v1/setup/complete", s.handleSetupComplete)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.Handle("GET /api/v1/auth/session", s.requireAuth(http.HandlerFunc(s.handleSession)))
	mux.Handle("GET /api/v1/cluster", s.requireAuth(http.HandlerFunc(s.handleCluster)))
	mux.Handle("POST /api/v1/deployments", s.requireAuth(http.HandlerFunc(s.handleDeployment)))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "API route not found")
	})
	mux.Handle("/", newStaticHandler(s.assets))

	return s.requestLogging(s.securityHeaders(s.sameOrigin(mux)))
}

func (s *Server) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	checkCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.store.Ready(checkCtx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "database is unavailable")
		return
	}
	if s.requireDocker {
		if s.docker == nil {
			writeError(w, http.StatusServiceUnavailable, "docker_unavailable", "Docker Engine is unavailable")
			return
		}
		if err := s.docker.Ready(checkCtx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "docker_unavailable", "Docker Engine is unavailable")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":   s.version,
		"commit":    s.commit,
		"sourceUrl": s.sourceURL,
	})
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	configured, err := s.auth.SetupStatus(r.Context())
	if err != nil {
		s.logger.Error("read setup status", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "unable to read setup status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"completed": configured,
	})
}

type setupRequest struct {
	InitToken string `json:"initToken"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	ACMEEmail string `json:"acmeEmail"`
}

func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	var request setupRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := s.auth.Setup(r.Context(), application.SetupInput{
		InitToken: request.InitToken,
		Username:  request.Username,
		Password:  request.Password,
		ManagementAccess: domain.ManagementAccess{
			Domain:    request.Domain,
			ACMEEmail: request.ACMEEmail,
		},
	})
	if err != nil {
		s.handleAuthError(w, err)
		return
	}

	s.setSessionCookie(w, r, result)
	writeJSON(w, http.StatusCreated, result.User)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := s.auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		s.handleAuthError(w, err)
		return
	}

	s.setSessionCookie(w, r, result)
	writeJSON(w, http.StatusOK, result.User)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		if err := s.auth.Logout(r.Context(), cookie.Value); err != nil {
			s.logger.Warn("delete session", "error", err)
		}
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleCluster(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.cluster.Snapshot(r.Context())
	if err != nil {
		s.logger.Warn("inspect Docker cluster", "error", err)
		writeError(w, http.StatusServiceUnavailable, "cluster_unavailable", "unable to inspect Docker Engine")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleDeployment(w http.ResponseWriter, r *http.Request) {
	var request domain.DeploymentSpec
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := s.deployments.Deploy(r.Context(), request)
	if err != nil {
		s.logger.Warn("deploy Swarm service", "error", err)
		writeError(w, http.StatusBadRequest, "deployment_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAlreadyConfigured):
		writeError(w, http.StatusConflict, "already_configured", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
	case errors.Is(err, domain.ErrInvalidInitToken):
		writeError(w, http.StatusUnauthorized, "invalid_init_token", err.Error())
	case errors.Is(err, domain.ErrInitTokenMissing):
		writeError(w, http.StatusServiceUnavailable, "init_token_missing", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
	}
}

func (s *Server) setSessionCookie(
	w http.ResponseWriter,
	r *http.Request,
	result application.AuthResult,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    result.Session.Token,
		Path:     "/",
		Expires:  result.Session.ExpiresAt,
		MaxAge:   int(time.Until(result.Session.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure || requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func requestUsesHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
