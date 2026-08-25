// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusRecorderPreservesStreaming(t *testing.T) {
	t.Parallel()

	underlying := httptest.NewRecorder()
	recorder := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}
	var flusher http.Flusher = recorder

	flusher.Flush()

	if !underlying.Flushed {
		t.Fatal("Flush() did not reach the underlying response writer")
	}
	if recorder.Unwrap() != underlying {
		t.Fatal("Unwrap() did not return the underlying response writer")
	}
}
