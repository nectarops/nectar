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
	Logger           *slog.Logger
	Auth             *application.AuthService
	Cluster          *application.ClusterService
	ManagementAccess *application.ManagementAccessService
	Deployments      *application.DeploymentService
	NodeEnrollments  *application.NodeEnrollmentService
	Store            Readiness
	Docker           Readiness
	RequireDocker    bool
	Assets           fs.FS
	CookieSecure     bool
	SourceURL        string
	Version          string
	Commit           string
}

type Server struct {
	logger           *slog.Logger
	auth             *application.AuthService
	cluster          *application.ClusterService
	managementAccess *application.ManagementAccessService
	deployments      *application.DeploymentService
	nodeEnrollments  *application.NodeEnrollmentService
	store            Readiness
	docker           Readiness
	requireDocker    bool
	assets           fs.FS
	cookieSecure     bool
	sourceURL        string
	version          string
	commit           string
}

func NewServer(options Options) (*Server, error) {
	if options.Auth == nil {
		return nil, errors.New("authentication service is required")
	}
	if options.Cluster == nil {
		return nil, errors.New("cluster service is required")
	}
	if options.ManagementAccess == nil {
		return nil, errors.New("management access service is required")
	}
	if options.Deployments == nil {
		return nil, errors.New("deployment service is required")
	}
	if options.NodeEnrollments == nil {
		return nil, errors.New("node enrollment service is required")
	}
	if options.Store == nil {
		return nil, errors.New("store readiness check is required")
	}
	if options.Assets == nil {
		return nil, errors.New("frontend assets are required")
	}

	return &Server{
		logger:           loggerOrDefault(options.Logger),
		auth:             options.Auth,
		cluster:          options.Cluster,
		managementAccess: options.ManagementAccess,
		deployments:      options.Deployments,
		nodeEnrollments:  options.NodeEnrollments,
		store:            options.Store,
		docker:           options.Docker,
		requireDocker:    options.RequireDocker,
		assets:           options.Assets,
		cookieSecure:     options.CookieSecure,
		sourceURL:        options.SourceURL,
		version:          options.Version,
		commit:           options.Commit,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.handleLiveness)
	mux.HandleFunc("GET /health/ready", s.handleReadiness)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /client.sh", s.handleNodeClientScript)
	mux.HandleFunc("GET /api/v1/setup/status", s.handleSetupStatus)
	mux.HandleFunc("POST /api/v1/setup/complete", s.handleSetupComplete)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.Handle("GET /api/v1/auth/session", s.requireAuth(http.HandlerFunc(s.handleSession)))
	mux.Handle("GET /api/v1/cluster", s.requireAuth(http.HandlerFunc(s.handleCluster)))
	mux.Handle("GET /api/v1/management-access", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleManagementAccess))))
	mux.Handle("PUT /api/v1/management-access", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleManagementAccessUpdate))))
	mux.Handle("POST /api/v1/deployments", s.requireAuth(http.HandlerFunc(s.handleDeployment)))
	mux.Handle("GET /api/v1/nodes", s.requireAuth(http.HandlerFunc(s.handleNodes)))
	mux.Handle("GET /api/v1/node-enrollments", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleNodeEnrollments))))
	mux.Handle("POST /api/v1/node-enrollments", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleNodeEnrollmentCreate))))
	mux.Handle("DELETE /api/v1/node-enrollments/{id}", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleNodeEnrollmentRevoke))))
	mux.Handle("GET /api/v1/node-enrollments/{id}/events", s.requireAuth(s.requireOwner(http.HandlerFunc(s.handleNodeEnrollmentEvents))))
	mux.HandleFunc("POST /api/v1/node-enrollments/claim", s.handleNodeEnrollmentClaim)
	mux.HandleFunc("POST /api/v1/node-enrollments/progress", s.handleNodeEnrollmentProgress)
	mux.HandleFunc("POST /api/v1/node-enrollments/complete", s.handleNodeEnrollmentComplete)
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
	s.clearSessionCookie(w, r)
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

func (s *Server) handleManagementAccess(w http.ResponseWriter, r *http.Request) {
	access, err := s.managementAccess.Current(r.Context())
	if err != nil {
		s.logger.Warn("read management access", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "unable to read management access")
		return
	}
	writeJSON(w, http.StatusOK, access)
}

func (s *Server) handleManagementAccessUpdate(w http.ResponseWriter, r *http.Request) {
	var request domain.ManagementAccess
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	access, err := s.managementAccess.Configure(r.Context(), request)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidManagementAccess) {
			writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
			return
		}
		s.logger.Warn("configure management access", "error", err)
		writeError(w, http.StatusServiceUnavailable, "traefik_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, access)
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

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure || requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}
