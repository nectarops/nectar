// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/domain"
	"github.com/nectarops/nectar/internal/nodeclient"
)

const maxNodeEnrollmentBody = 16 << 10

type createNodeEnrollmentRequest struct {
	Role domain.NodeRole `json:"role"`
}

func (s *Server) handleNodeClientScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `inline; filename="client.sh"`)
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write(nodeclient.Script())
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.nodeEnrollments.Nodes(r.Context())
	if err != nil {
		s.logger.Warn("list Swarm nodes", "error", err)
		writeError(w, http.StatusServiceUnavailable, "nodes_unavailable", "unable to list Swarm nodes")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

func (s *Server) handleNodeEnrollments(w http.ResponseWriter, r *http.Request) {
	enrollments, err := s.nodeEnrollments.List(r.Context())
	if err != nil {
		s.logger.Warn("list node enrollments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "unable to list node enrollments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enrollments": enrollments})
}

func (s *Server) handleNodeEnrollmentCreate(w http.ResponseWriter, r *http.Request) {
	var request createNodeEnrollmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}

	baseURL, err := requestBaseURL(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request host is invalid")
		return
	}
	credential, err := s.nodeEnrollments.Create(r.Context(), user.ID, request.Role)
	if err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	command := fmt.Sprintf(
		"curl -fsSL %s/client.sh | sudo env NECTAR_SERVER_URL=%s bash -s -- %s",
		shellQuote(baseURL),
		shellQuote(baseURL),
		shellQuote(credential.Token),
	)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{
		"enrollment": credential.Enrollment,
		"command":    command,
	})
}

func (s *Server) handleNodeEnrollmentRevoke(w http.ResponseWriter, r *http.Request) {
	enrollment, err := s.nodeEnrollments.Revoke(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, enrollment)
}

func (s *Server) handleNodeEnrollmentClaim(w http.ResponseWriter, r *http.Request) {
	if err := parseEnrollmentForm(w, r); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	bootstrap, err := s.nodeEnrollments.Claim(
		r.Context(),
		bearerToken(r),
		domain.NodeEnrollmentClaim{
			Hostname:         r.Form.Get("hostname"),
			MachineIDHash:    r.Form.Get("machineIdHash"),
			OperatingSystem:  r.Form.Get("operatingSystem"),
			Architecture:     r.Form.Get("architecture"),
			AdvertiseAddress: r.Form.Get("advertiseAddress"),
			DataPathAddress:  r.Form.Get("dataPathAddress"),
			DockerVersion:    r.Form.Get("dockerVersion"),
		},
	)
	if err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(
		w,
		"ENROLLMENT_ID=%s\nREQUESTED_ROLE=%s\nDOCKER_TARGET_VERSION=%s\nMANAGER_ADDRESS=%s\nSWARM_CLUSTER_ID=%s\nWORKER_JOIN_TOKEN=%s\n",
		bootstrap.Enrollment.ID,
		bootstrap.Enrollment.RequestedRole,
		bootstrap.DockerTargetVersion,
		bootstrap.ManagerAddress,
		bootstrap.SwarmClusterID,
		bootstrap.WorkerJoinToken,
	)
}

func (s *Server) handleNodeEnrollmentProgress(w http.ResponseWriter, r *http.Request) {
	if err := parseEnrollmentForm(w, r); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	err := s.nodeEnrollments.Progress(
		r.Context(),
		bearerToken(r),
		r.Form.Get("machineIdHash"),
		r.Form.Get("phase"),
		r.Form.Get("dockerVersion"),
	)
	if err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNodeEnrollmentComplete(w http.ResponseWriter, r *http.Request) {
	if err := parseEnrollmentForm(w, r); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	enrollment, err := s.nodeEnrollments.Complete(
		r.Context(),
		bearerToken(r),
		r.Form.Get("machineIdHash"),
		r.Form.Get("nodeId"),
	)
	if errors.Is(err, domain.ErrManagerVersionMismatch) {
		writeJSON(w, http.StatusAccepted, enrollment)
		return
	}
	if err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, enrollment)
}

func (s *Server) handleNodeEnrollmentEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	afterID := int64(0)
	if lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID")); lastID != "" {
		parsed, err := strconv.ParseInt(lastID, 10, 64)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Last-Event-ID must be a positive integer")
			return
		}
		afterID = parsed
	}
	if _, err := s.nodeEnrollments.Enrollment(r.Context(), id); err != nil {
		s.writeNodeEnrollmentError(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unavailable", "event streaming is unavailable")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()
	for {
		events, err := s.nodeEnrollments.Events(r.Context(), id, afterID)
		if err != nil {
			return
		}
		for _, event := range events {
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				return
			}
			_, _ = fmt.Fprintf(w, "id: %d\nevent: enrollment\ndata: %s\n\n", event.ID, payload)
			afterID = event.ID
		}
		if len(events) > 0 {
			flusher.Flush()
		}
		enrollment, err := s.nodeEnrollments.Enrollment(r.Context(), id)
		if err != nil {
			return
		}
		if enrollment.Terminal() {
			if enrollment.Status == domain.NodeEnrollmentExpired {
				expired := domain.NodeEnrollmentEvent{
					EnrollmentID: enrollment.ID,
					Status:       enrollment.Status,
					Message:      enrollment.Message,
					CreatedAt:    time.Now().UTC(),
				}
				payload, marshalErr := json.Marshal(expired)
				if marshalErr == nil {
					_, _ = fmt.Fprintf(w, "event: enrollment\ndata: %s\n\n", payload)
					flusher.Flush()
				}
			}
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		case <-ticker.C:
		}
	}
}

func (s *Server) writeNodeEnrollmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEnrollment):
		writeError(w, http.StatusBadRequest, "invalid_enrollment", err.Error())
	case errors.Is(err, domain.ErrInvalidEnrollmentToken):
		w.Header().Set("WWW-Authenticate", `Bearer realm="node-enrollment"`)
		writeError(w, http.StatusUnauthorized, "invalid_enrollment_token", "invalid node enrollment token")
	case errors.Is(err, domain.ErrEnrollmentExpired):
		writeError(w, http.StatusGone, "enrollment_expired", err.Error())
	case errors.Is(err, domain.ErrEnrollmentClaimed),
		errors.Is(err, domain.ErrEnrollmentTerminal),
		errors.Is(err, domain.ErrNodeNotReady),
		errors.Is(err, domain.ErrManagerVersionMismatch):
		writeError(w, http.StatusConflict, "enrollment_conflict", err.Error())
	default:
		s.logger.Warn("node enrollment", "error", err)
		writeError(w, http.StatusServiceUnavailable, "enrollment_unavailable", "node enrollment is unavailable")
	}
}

func parseEnrollmentForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxNodeEnrollmentBody)
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse enrollment form: %w", err)
	}
	return nil
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	prefix, token, ok := strings.Cut(value, " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func requestBaseURL(r *http.Request) (string, error) {
	host := requestHost(r)
	if host == "" || strings.ContainsAny(host, "\r\n\t /\\'") {
		return "", errors.New("invalid request host")
	}
	scheme := "http"
	if requestUsesHTTPS(r) {
		scheme = "https"
	}
	return (&url.URL{Scheme: scheme, Host: host}).String(), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
