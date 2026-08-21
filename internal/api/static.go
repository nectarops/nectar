// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func newStaticHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested == "." || requested == "" {
			requested = "index.html"
		}

		content, err := fs.ReadFile(assets, requested)
		if err != nil {
			requested = "index.html"
			content, err = fs.ReadFile(assets, requested)
		}
		if err != nil {
			writeError(
				w,
				http.StatusServiceUnavailable,
				"frontend_unavailable",
				"the frontend has not been built into this binary",
			)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(requested))
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		if requested == "index.html" {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(content)
		}
	})
}
