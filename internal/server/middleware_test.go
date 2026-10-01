package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverPanic(t *testing.T) {
	server := newTestServer(t)
	handler := server.recoverPanic(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	status := w.Result().StatusCode
	if status != http.StatusInternalServerError {
		t.Errorf("recoverPanic() status = %d, want %d", status, http.StatusInternalServerError)
	}
}
