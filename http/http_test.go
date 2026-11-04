//  Copyright © 2026 ABB Schweiz AG. All Rights Reserved.
//  Authors: Adam Lange, et al.

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSEnabledHandlerPreflight(t *testing.T) {
	handler := NewCORSEnabledHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight must not reach the handler")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/v2/tenants", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Methods"), "PATCH")
	for _, header := range []string{"Authorization", "Content-Type", "X-Api-Key", "Signature", "Signature-Input", "Content-Digest"} {
		assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Headers"), header)
	}
}
