package rest

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerRecordsResponse(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	handler := &Handler{log: slog.New(slog.NewJSONHandler(&output, nil))}
	wrapped := requestIDMiddleware(handler.requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("invalid"))
	})))

	recorder := httptest.NewRecorder()
	wrapped.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/notifications", nil))
	log := output.String()
	for _, expected := range []string{`"msg":"http request completed"`, `"method":"POST"`, `"path":"/api/v1/notifications"`, `"status":422`, `"bytes":7`, `"request_id":"`} {
		if !strings.Contains(log, expected) {
			t.Fatalf("log %q does not contain %q", log, expected)
		}
	}
}
