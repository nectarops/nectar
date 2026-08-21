// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONAcceptsCharset(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"dock-weaver"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	recorder := httptest.NewRecorder()
	var body struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(recorder, request, &body); err != nil {
		t.Fatalf("decodeJSON() error = %v", err)
	}
	if body.Name != "dock-weaver" {
		t.Fatalf("decoded name = %q", body.Name)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	var body struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(recorder, request, &body); err == nil {
		t.Fatal("decodeJSON() accepted an unknown field")
	}
}
